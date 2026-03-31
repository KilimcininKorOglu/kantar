import { useState, useEffect } from 'react'
import { api } from '../api/client'
import type { AuditLogEntry } from '../api/types'
import { formatDateTime } from '../utils/date'
import { Download, ShieldCheck, Filter } from 'lucide-react'

export default function AuditLog() {
  const [logs, setLogs] = useState<AuditLogEntry[]>([])
  const [eventFilter, setEventFilter] = useState('')
  const [actorFilter, setActorFilter] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.get<AuditLogEntry[]>('/audit?limit=200')
      .then(setLogs).catch(() => {}).finally(() => setLoading(false))
  }, [])

  const filteredLogs = logs.filter((l) => {
    if (eventFilter && l.event !== eventFilter) return false
    if (actorFilter && !l.actorUsername.toLowerCase().includes(actorFilter.toLowerCase())) return false
    return true
  })

  const handleVerify = async () => {
    try {
      const result = await api.get<{ valid: boolean; totalEntries: number }>('/audit/verify')
      alert(result.valid ? `Chain verified: ${result.totalEntries} entries, all valid.` : `Chain verification FAILED!`)
    } catch { alert('Verification failed') }
  }

  const handleExport = (format: 'csv' | 'json') => {
    const data = format === 'json' ? JSON.stringify(filteredLogs, null, 2) :
      ['Timestamp,Event,Actor,Resource,Result', ...filteredLogs.map(l => `${l.timestamp},${l.event},${l.actorUsername},${l.resourcePackage || l.resourceRegistry},${l.result}`)].join('\n')
    const blob = new Blob([data], { type: format === 'json' ? 'application/json' : 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a'); a.href = url; a.download = `audit-log.${format}`; a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div />
        <div className="flex gap-2">
          <button onClick={() => handleExport('csv')} className="flex items-center gap-1.5 px-3 py-1.5 bg-surface-2 border border-border text-text-muted text-xs rounded cursor-pointer"><Download className="w-3 h-3" /> CSV</button>
          <button onClick={() => handleExport('json')} className="flex items-center gap-1.5 px-3 py-1.5 bg-surface-2 border border-border text-text-muted text-xs rounded cursor-pointer"><Download className="w-3 h-3" /> JSON</button>
          <button onClick={handleVerify} className="flex items-center gap-1.5 px-3 py-1.5 bg-surface-2 border border-border text-text-muted text-xs rounded cursor-pointer"><ShieldCheck className="w-3 h-3" /> Verify</button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex gap-3 items-center">
        <Filter className="w-3.5 h-3.5 text-text-dim" />
        <select value={eventFilter} onChange={(e) => setEventFilter(e.target.value)} className="bg-surface-2 border border-border rounded px-3 py-1.5 text-xs text-text">
          <option value="">All Events</option>
          {['user.login', 'user.create', 'package.approve', 'package.block', 'package.download', 'registry.sync', 'policy.violation'].map(e => <option key={e} value={e}>{e}</option>)}
        </select>
        <input type="text" placeholder="Filter by actor..." value={actorFilter} onChange={(e) => setActorFilter(e.target.value)} className="bg-surface-2 border border-border rounded px-3 py-1.5 text-xs text-text w-48 focus:outline-none focus:border-accent" />
      </div>

      {/* Table */}
      <div className="bg-surface border border-border rounded overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-text-dim">
              <th className="text-left px-4 py-2.5 text-xs font-medium">Timestamp</th>
              <th className="text-left px-4 py-2.5 text-xs font-medium">Event</th>
              <th className="text-left px-4 py-2.5 text-xs font-medium">Actor</th>
              <th className="text-left px-4 py-2.5 text-xs font-medium">Resource</th>
              <th className="text-left px-4 py-2.5 text-xs font-medium">Result</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={5} className="px-4 py-10 text-center text-text-dim text-xs">Loading...</td></tr>
            ) : filteredLogs.length === 0 ? (
              <tr><td colSpan={5} className="px-4 py-10 text-center text-text-dim text-xs">No audit log entries</td></tr>
            ) : filteredLogs.map((l) => (
              <tr key={l.id} className="border-b border-border last:border-0">
                <td className="px-4 py-2 text-[11px] text-text-muted font-mono">{formatDateTime(l.timestamp)}</td>
                <td className="px-4 py-2"><span className="text-[11px] font-mono text-accent">{l.event}</span></td>
                <td className="px-4 py-2 text-xs text-text">{l.actorUsername || '—'}</td>
                <td className="px-4 py-2 text-xs text-text-muted">{l.resourcePackage || l.resourceRegistry || '—'}</td>
                <td className="px-4 py-2">
                  <span className={`text-[11px] font-medium ${l.result === 'success' ? 'text-success' : 'text-danger'}`}>{l.result}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
