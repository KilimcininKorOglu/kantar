import { useState, useEffect } from 'react'
import { api } from '../api/client'
import type { AuditLogEntry } from '../api/types'
import { formatDateTime } from '../utils/date'

export default function AuditLog() {
  const [logs, setLogs] = useState<AuditLogEntry[]>([])
  const [eventFilter, setEventFilter] = useState('')
  const [actorFilter, setActorFilter] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.get<AuditLogEntry[]>('/audit?limit=200')
      .then(setLogs)
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const filteredLogs = logs.filter((l) => {
    if (eventFilter && l.event !== eventFilter) return false
    if (actorFilter && !l.actorUsername.toLowerCase().includes(actorFilter.toLowerCase())) return false
    return true
  })

  const handleVerify = async () => {
    try {
      const result = await api.get<{ valid: boolean; totalEntries: number }>('/audit/verify')
      alert(result.valid
        ? `Chain verified: ${result.totalEntries} entries, all valid.`
        : `Chain verification FAILED at ${result.totalEntries} entries!`)
    } catch {
      alert('Verification failed — server error')
    }
  }

  const handleExport = (format: 'csv' | 'json') => {
    const data = format === 'json'
      ? JSON.stringify(filteredLogs, null, 2)
      : [
          'Timestamp,Event,Actor,Resource,Result',
          ...filteredLogs.map(l =>
            `${l.timestamp},${l.event},${l.actorUsername},${l.resourcePackage || l.resourceRegistry},${l.result}`
          ),
        ].join('\n')

    const blob = new Blob([data], { type: format === 'json' ? 'application/json' : 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `audit-log.${format}`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-white">Audit Log</h2>
        <div className="flex gap-2">
          <button
            onClick={() => handleExport('csv')}
            className="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs rounded transition-colors cursor-pointer"
          >Export CSV</button>
          <button
            onClick={() => handleExport('json')}
            className="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs rounded transition-colors cursor-pointer"
          >Export JSON</button>
          <button
            onClick={handleVerify}
            className="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs rounded transition-colors cursor-pointer"
          >Verify Chain</button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex gap-3">
        <select
          value={eventFilter}
          onChange={(e) => setEventFilter(e.target.value)}
          className="bg-slate-800 border border-slate-700 rounded px-3 py-1.5 text-sm text-slate-300"
        >
          <option value="">All Events</option>
          <option value="user.login">user.login</option>
          <option value="user.create">user.create</option>
          <option value="package.approve">package.approve</option>
          <option value="package.block">package.block</option>
          <option value="package.download">package.download</option>
          <option value="registry.sync">registry.sync</option>
          <option value="policy.violation">policy.violation</option>
        </select>
        <input
          type="text"
          placeholder="Filter by actor..."
          value={actorFilter}
          onChange={(e) => setActorFilter(e.target.value)}
          className="bg-slate-800 border border-slate-700 rounded px-3 py-1.5 text-sm text-white focus:outline-none focus:border-blue-500"
        />
      </div>

      {/* Audit Table */}
      <div className="bg-slate-900 border border-slate-800 rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-800 text-slate-500">
              <th className="text-left px-4 py-3 font-medium">Timestamp</th>
              <th className="text-left px-4 py-3 font-medium">Event</th>
              <th className="text-left px-4 py-3 font-medium">Actor</th>
              <th className="text-left px-4 py-3 font-medium">Resource</th>
              <th className="text-left px-4 py-3 font-medium">Result</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={5} className="px-4 py-12 text-center text-slate-600">Loading...</td>
              </tr>
            ) : filteredLogs.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-12 text-center text-slate-600">
                  No audit log entries
                </td>
              </tr>
            ) : (
              filteredLogs.map((l) => (
                <tr key={l.id} className="border-b border-slate-800/50">
                  <td className="px-4 py-3 text-slate-400 text-xs">
                    {formatDateTime(l.timestamp)}
                  </td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 text-xs rounded ${
                      l.event.startsWith('user.') ? 'bg-blue-900/40 text-blue-300' :
                      l.event.startsWith('package.') ? 'bg-purple-900/40 text-purple-300' :
                      'bg-slate-800 text-slate-300'
                    }`}>{l.event}</span>
                  </td>
                  <td className="px-4 py-3 text-white">{l.actorUsername || '—'}</td>
                  <td className="px-4 py-3 text-slate-400">
                    {l.resourcePackage || l.resourceRegistry || '—'}
                  </td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 text-xs rounded ${
                      l.result === 'success' ? 'bg-emerald-900/40 text-emerald-300' : 'bg-red-900/40 text-red-300'
                    }`}>{l.result}</span>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
