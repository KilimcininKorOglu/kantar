import { useState, useEffect, useCallback } from 'react'
import { useParams, Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { api } from '../api/client'
import type { Package, PackageVersion } from '../api/types'
import { formatDate } from '../utils/date'
import { ArrowLeft, Check, Ban } from 'lucide-react'

export default function PackageDetail() {
  const { t } = useTranslation()
  const { registry, name } = useParams()
  const [pkg, setPkg] = useState<Package | null>(null)
  const [versions, setVersions] = useState<PackageVersion[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [actionLoading, setActionLoading] = useState(false)

  const loadPackage = useCallback(() => {
    setLoading(true)
    api.get<Package>(`/packages/by-name/${registry}/${encodeURIComponent(name!)}`)
      .then((p) => {
        setPkg(p)
        api.get<PackageVersion[]>(`/packages/${p.id}/versions`).then(setVersions).catch(() => {})
      })
      .catch(() => setError(t('packageDetail.packageNotFound')))
      .finally(() => setLoading(false))
  }, [registry, name, t])

  useEffect(() => { loadPackage() }, [loadPackage])

  const handleApprove = async () => {
    if (!pkg) return
    setError('')
    setActionLoading(true)
    try { await api.post(`/packages/${pkg.id}/approve`); loadPackage() }
    catch { setError(t('packageDetail.failedToApprove')) }
    setActionLoading(false)
  }

  const handleBlock = async () => {
    if (!pkg) return
    const reason = prompt(t('packageDetail.blockReason'))
    if (!reason) return
    setError('')
    setActionLoading(true)
    try { await api.post(`/packages/${pkg.id}/block`, { reason }); loadPackage() }
    catch { setError(t('packageDetail.failedToBlock')) }
    setActionLoading(false)
  }

  if (loading) return <div className="text-text-dim py-12 text-center text-xs">{t('common.loading')}</div>
  if (!pkg) return (
    <div className="space-y-4">
      <Link to="/packages" className="flex items-center gap-1 text-accent text-xs"><ArrowLeft className="w-3 h-3" /> {t('common.back')}</Link>
      <div className="text-text-dim py-12 text-center text-xs">{error || t('packageDetail.packageNotFound')}</div>
    </div>
  )

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <Link to="/packages" className="flex items-center gap-1 text-accent text-xs mb-2"><ArrowLeft className="w-3 h-3" /> {t('common.back')}</Link>
          <p className="text-[11px] text-text-dim uppercase tracking-wider font-mono">{pkg.registryType}</p>
          <h2 className="text-lg font-bold text-text">{pkg.name}</h2>
          <span className={`text-[11px] font-medium ${pkg.status === 'approved' ? 'text-success' : pkg.status === 'blocked' ? 'text-danger' : 'text-warning'}`}>{pkg.status}</span>
        </div>
        <div className="flex gap-2">
          {pkg.status !== 'approved' && (
            <button onClick={handleApprove} disabled={actionLoading} className="flex items-center gap-1.5 px-3 py-1.5 bg-success/20 text-success text-xs font-medium rounded cursor-pointer disabled:opacity-50">
              <Check className="w-3.5 h-3.5" /> {t('packageDetail.approve')}
            </button>
          )}
          {pkg.status !== 'blocked' && (
            <button onClick={handleBlock} disabled={actionLoading} className="flex items-center gap-1.5 px-3 py-1.5 bg-danger/20 text-danger text-xs font-medium rounded cursor-pointer disabled:opacity-50">
              <Ban className="w-3.5 h-3.5" /> {t('packageDetail.block')}
            </button>
          )}
        </div>
      </div>

      {error && <div className="bg-danger/10 border border-danger/20 text-danger text-xs rounded px-3 py-2">{error}</div>}

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <InfoCard title={t('packageDetail.details')} items={[
          [t('packages.license'), pkg.license || '—'],
          [t('packageDetail.homepage'), pkg.homepage || '—'],
          [t('packageDetail.repository'), pkg.repository || '—'],
        ]} />
        <InfoCard title={t('common.status')} items={[
          [t('packageDetail.requestedBy'), pkg.requestedBy || '—'],
          [t('packageDetail.approvedBy'), pkg.approvedBy || '—'],
          ...(pkg.blockedReason ? [[t('packageDetail.blockReasonLabel'), pkg.blockedReason] as [string, string]] : []),
        ]} />
        <InfoCard title={t('packageDetail.dates')} items={[
          [t('packageDetail.created'), formatDate(pkg.createdAt)],
          [t('packageDetail.updated'), formatDate(pkg.updatedAt)],
        ]} />
      </div>

      {pkg.description && (
        <div className="bg-surface border border-border rounded p-4">
          <h3 className="text-xs font-semibold text-text-dim uppercase tracking-wider mb-2">{t('packageDetail.description')}</h3>
          <p className="text-xs text-text-muted">{pkg.description}</p>
        </div>
      )}

      {versions.length > 0 && (
        <div className="bg-surface border border-border rounded p-4">
          <h3 className="text-xs font-semibold text-text-dim uppercase tracking-wider mb-3">{t('packageDetail.versions')}</h3>
          <table className="w-full">
            <thead>
              <tr className="border-b border-border">
                <th className="text-left text-[10px] text-text-dim uppercase px-3 py-2">{t('dashboard.version')}</th>
                <th className="text-left text-[10px] text-text-dim uppercase px-3 py-2">{t('packageDetail.size')}</th>
                <th className="text-left text-[10px] text-text-dim uppercase px-3 py-2">{t('common.status')}</th>
                <th className="text-left text-[10px] text-text-dim uppercase px-3 py-2">{t('packageDetail.created')}</th>
              </tr>
            </thead>
            <tbody>
              {versions.map(v => (
                <tr key={v.id} className="border-b border-border last:border-0">
                  <td className="px-3 py-2 text-xs text-text font-mono">{v.version}</td>
                  <td className="px-3 py-2 text-xs text-text-muted">{v.size > 0 ? `${(v.size / 1024).toFixed(1)} KB` : '—'}</td>
                  <td className="px-3 py-2 text-xs">
                    {v.deprecated && <span className="text-warning">deprecated</span>}
                    {v.yanked && <span className="text-danger">yanked</span>}
                    {!v.deprecated && !v.yanked && <span className="text-success">ok</span>}
                  </td>
                  <td className="px-3 py-2 text-xs text-text-muted">{formatDate(v.createdAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function InfoCard({ title, items }: { title: string; items: [string, string][] }) {
  return (
    <div className="bg-surface border border-border rounded p-4">
      <h3 className="text-xs font-semibold text-text-dim uppercase tracking-wider mb-3">{title}</h3>
      <div className="space-y-2">
        {items.map(([k, v]) => (
          <div key={k} className="flex justify-between text-xs">
            <span className="text-text-dim">{k}</span>
            <span className="text-text font-mono truncate ml-2 max-w-[180px]">{v}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
