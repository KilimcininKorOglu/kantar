import { useState, useEffect } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { api } from '../api/client'
import type { Package } from '../api/types'
import { Search } from 'lucide-react'

const registries = ['docker', 'npm', 'pypi', 'gomod', 'cargo', 'maven', 'nuget', 'helm']

export default function PackageList() {
  const { t } = useTranslation()
  const [activeRegistry, setActiveRegistry] = useState('npm')
  const [search, setSearch] = useState('')
  const [packages, setPackages] = useState<Package[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    const params = new URLSearchParams({ registry: activeRegistry, limit: '100' })
    if (search) params.set('search', search)
    api.get<Package[]>(`/packages?${params}`)
      .then(setPackages).catch(() => setPackages([])).finally(() => setLoading(false))
  }, [activeRegistry, search])

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div />
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-text-dim" />
          <input type="text" placeholder={t('packages.searchPackages')} value={search} onChange={(e) => setSearch(e.target.value)}
            className="bg-surface-2 border border-border rounded pl-9 pr-3 py-1.5 text-xs text-text w-56 focus:outline-none focus:border-accent" />
        </div>
      </div>

      <div className="flex gap-0.5 bg-surface border border-border rounded p-0.5">
        {registries.map((reg) => (
          <button key={reg} onClick={() => setActiveRegistry(reg)}
            className={`px-3 py-1.5 text-[11px] font-medium rounded cursor-pointer ${
              activeRegistry === reg ? 'bg-accent text-white' : 'text-text-muted hover:text-text hover:bg-surface-2'
            }`}>{reg}</button>
        ))}
      </div>

      <div className="bg-surface border border-border rounded overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-text-dim">
              <th className="text-left px-4 py-2.5 text-xs font-medium">{t('common.name')}</th>
              <th className="text-left px-4 py-2.5 text-xs font-medium">{t('common.status')}</th>
              <th className="text-left px-4 py-2.5 text-xs font-medium">{t('packages.license')}</th>
              <th className="text-left px-4 py-2.5 text-xs font-medium">{t('packages.requestedBy')}</th>
              <th className="text-right px-4 py-2.5 text-xs font-medium">{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={5} className="px-4 py-10 text-center text-text-dim text-xs">{t('common.loading')}</td></tr>
            ) : packages.length === 0 ? (
              <tr><td colSpan={5} className="px-4 py-10 text-center text-text-dim text-xs">
                {t('packages.noPackagesIn', { registry: activeRegistry })}
                {search && <> {t('packages.matching', { search })}</>}
              </td></tr>
            ) : packages.map((pkg) => (
              <tr key={pkg.id} className="border-b border-border last:border-0">
                <td className="px-4 py-2.5">
                  <Link to={`/packages/${pkg.registryType}/${encodeURIComponent(pkg.name)}`} className="text-xs text-accent hover:underline font-medium">{pkg.name}</Link>
                </td>
                <td className="px-4 py-2.5"><StatusBadge status={pkg.status} /></td>
                <td className="px-4 py-2.5 text-xs text-text-muted font-mono">{pkg.license || '—'}</td>
                <td className="px-4 py-2.5 text-xs text-text-muted">{pkg.requestedBy || '—'}</td>
                <td className="px-4 py-2.5 text-right">
                  <Link to={`/packages/${pkg.registryType}/${encodeURIComponent(pkg.name)}`} className="text-[11px] text-text-dim hover:text-accent">{t('common.view')}</Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const cls = status === 'approved' ? 'text-success' : status === 'blocked' ? 'text-danger' : 'text-warning'
  return <span className={`text-[11px] font-medium ${cls}`}>{status}</span>
}
