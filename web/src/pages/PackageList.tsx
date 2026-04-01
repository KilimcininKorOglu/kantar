import { useState, useEffect } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { api } from '../api/client'
import type { Package, PaginatedResponse } from '../api/types'
import { Search, ChevronLeft, ChevronRight } from 'lucide-react'

const registries = ['docker', 'npm', 'pypi', 'gomod', 'cargo', 'maven', 'nuget', 'helm']

export default function PackageList() {
  const { t } = useTranslation()
  const [activeRegistry, setActiveRegistry] = useState('npm')
  const [search, setSearch] = useState('')
  const [packages, setPackages] = useState<Package[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const perPage = 50

  useEffect(() => {
    setLoading(true)
    const params = new URLSearchParams({ registry: activeRegistry, page: String(page), perPage: String(perPage) })
    if (search) params.set('search', search)
    api.get<PaginatedResponse<Package>>(`/packages?${params}`)
      .then((res) => { setPackages(res.data as Package[]); setTotal(res.total) })
      .catch(() => { setPackages([]); setTotal(0) })
      .finally(() => setLoading(false))
  }, [activeRegistry, search, page])

  const totalPages = Math.max(1, Math.ceil(total / perPage))

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
          <button key={reg} onClick={() => { setActiveRegistry(reg); setPage(1) }}
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
        {total > perPage && (
          <div className="flex items-center justify-between px-4 py-2.5 border-t border-border">
            <span className="text-[11px] text-text-dim">{page} / {totalPages} ({total})</span>
            <div className="flex gap-1">
              <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page <= 1}
                className="px-2 py-1 bg-surface-2 border border-border rounded text-text-muted text-[11px] disabled:opacity-30 cursor-pointer">
                <ChevronLeft className="w-3 h-3" />
              </button>
              <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page >= totalPages}
                className="px-2 py-1 bg-surface-2 border border-border rounded text-text-muted text-[11px] disabled:opacity-30 cursor-pointer">
                <ChevronRight className="w-3 h-3" />
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const cls = status === 'approved' ? 'text-success' : status === 'blocked' ? 'text-danger' : 'text-warning'
  return <span className={`text-[11px] font-medium ${cls}`}>{status}</span>
}
