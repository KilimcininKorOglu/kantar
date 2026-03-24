import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router'
import { api } from '../api/client'
import type { Package } from '../api/types'

export default function PackageDetail() {
  const { registry, name } = useParams()
  const [pkg, setPkg] = useState<Package | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [actionLoading, setActionLoading] = useState(false)

  const loadPackage = () => {
    setLoading(true)
    api.get<Package>(`/packages/by-name/${registry}/${name}`)
      .then(setPkg)
      .catch(() => setError('Package not found'))
      .finally(() => setLoading(false))
  }

  useEffect(() => { loadPackage() }, [registry, name])

  const handleApprove = async () => {
    if (!pkg) return
    setActionLoading(true)
    try {
      await api.post(`/packages/${pkg.id}/approve`)
      loadPackage()
    } catch {
      setError('Failed to approve')
    } finally {
      setActionLoading(false)
    }
  }

  const handleBlock = async () => {
    if (!pkg) return
    const reason = prompt('Block reason:')
    if (!reason) return
    setActionLoading(true)
    try {
      await api.post(`/packages/${pkg.id}/block`, { reason })
      loadPackage()
    } catch {
      setError('Failed to block')
    } finally {
      setActionLoading(false)
    }
  }

  if (loading) {
    return <div className="text-slate-500 py-12 text-center">Loading...</div>
  }

  if (!pkg) {
    return (
      <div className="space-y-4">
        <Link to="/packages" className="text-blue-400 hover:text-blue-300 text-sm">&larr; Back to packages</Link>
        <div className="text-slate-500 py-12 text-center">{error || 'Package not found'}</div>
      </div>
    )
  }

  const statusBadge = (status: string) => {
    const cls = status === 'approved' ? 'bg-emerald-900/40 text-emerald-300'
      : status === 'blocked' ? 'bg-red-900/40 text-red-300'
      : 'bg-yellow-900/40 text-yellow-300'
    return <span className={`px-2 py-0.5 text-xs rounded ${cls}`}>{status}</span>
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <Link to="/packages" className="text-blue-400 hover:text-blue-300 text-xs">&larr; Back</Link>
          <div className="text-xs text-slate-500 mt-2">{pkg.registryType}</div>
          <h2 className="text-xl font-semibold text-white">{pkg.name}</h2>
          <div className="mt-1">{statusBadge(pkg.status)}</div>
        </div>
        <div className="flex gap-2">
          {pkg.status !== 'approved' && (
            <button
              onClick={handleApprove}
              disabled={actionLoading}
              className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-sm rounded transition-colors cursor-pointer"
            >Approve</button>
          )}
          {pkg.status !== 'blocked' && (
            <button
              onClick={handleBlock}
              disabled={actionLoading}
              className="px-3 py-1.5 bg-red-600 hover:bg-red-500 disabled:opacity-50 text-white text-sm rounded transition-colors cursor-pointer"
            >Block</button>
          )}
        </div>
      </div>

      {error && (
        <div className="bg-red-900/30 border border-red-800 text-red-300 text-sm rounded px-3 py-2">{error}</div>
      )}

      {/* Info Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <h3 className="text-sm font-medium text-slate-400 mb-3">Details</h3>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between"><span className="text-slate-500">License</span><span className="text-white">{pkg.license || '—'}</span></div>
            <div className="flex justify-between"><span className="text-slate-500">Homepage</span><span className="text-white truncate ml-2">{pkg.homepage || '—'}</span></div>
            <div className="flex justify-between"><span className="text-slate-500">Repository</span><span className="text-white truncate ml-2">{pkg.repository || '—'}</span></div>
          </div>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <h3 className="text-sm font-medium text-slate-400 mb-3">Status</h3>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between"><span className="text-slate-500">Requested By</span><span className="text-white">{pkg.requestedBy || '—'}</span></div>
            <div className="flex justify-between"><span className="text-slate-500">Approved By</span><span className="text-white">{pkg.approvedBy || '—'}</span></div>
            {pkg.blockedReason && (
              <div className="flex justify-between"><span className="text-slate-500">Block Reason</span><span className="text-red-300">{pkg.blockedReason}</span></div>
            )}
          </div>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <h3 className="text-sm font-medium text-slate-400 mb-3">Dates</h3>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between"><span className="text-slate-500">Created</span><span className="text-white">{new Date(pkg.createdAt).toLocaleDateString()}</span></div>
            <div className="flex justify-between"><span className="text-slate-500">Updated</span><span className="text-white">{new Date(pkg.updatedAt).toLocaleDateString()}</span></div>
          </div>
        </div>
      </div>

      {pkg.description && (
        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <h3 className="text-sm font-medium text-slate-400 mb-2">Description</h3>
          <p className="text-sm text-slate-300">{pkg.description}</p>
        </div>
      )}
    </div>
  )
}
