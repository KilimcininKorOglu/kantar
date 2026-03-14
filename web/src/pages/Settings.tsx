import { useState, useEffect } from 'react'
import type { SystemStatus } from '../api/types'

export default function Settings() {
  const [status, setStatus] = useState<SystemStatus | null>(null)

  useEffect(() => {
    fetch('/api/v1/system/status')
      .then((r) => r.json())
      .then(setStatus)
      .catch(() => {})
  }, [])

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-semibold text-white">Settings</h2>

      {/* System Info */}
      <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
        <h3 className="text-sm font-medium text-slate-400 mb-4">System Information</h3>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <InfoRow label="Status" value={status?.status || '—'} />
          <InfoRow label="Version" value={status?.version || '—'} />
          <InfoRow label="Uptime" value={status?.uptime || '—'} />
          <InfoRow label="Go Version" value={status?.goVersion || '—'} />
          <InfoRow label="CPUs" value={String(status?.numCpu || '—')} />
          <InfoRow label="Goroutines" value={String(status?.goroutines || '—')} />
          <InfoRow label="Memory (Alloc)" value={status ? `${(status.memory.allocBytes / 1048576).toFixed(1)} MB` : '—'} />
          <InfoRow label="Memory (Sys)" value={status ? `${(status.memory.sysBytes / 1048576).toFixed(1)} MB` : '—'} />
        </div>
      </div>

      {/* Configuration */}
      <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
        <h3 className="text-sm font-medium text-slate-400 mb-4">Configuration</h3>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <InfoRow label="Database" value="SQLite" />
          <InfoRow label="Storage" value="Filesystem" />
          <InfoRow label="Cache" value="In-Memory" />
          <InfoRow label="Auth" value="Local" />
        </div>
      </div>

      {/* System Actions */}
      <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
        <h3 className="text-sm font-medium text-slate-400 mb-4">Actions</h3>
        <div className="flex gap-3">
          <button className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-white text-sm rounded transition-colors cursor-pointer">
            Garbage Collection
          </button>
          <button className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-white text-sm rounded transition-colors cursor-pointer">
            Create Backup
          </button>
        </div>
      </div>
    </div>
  )
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between py-1 border-b border-slate-800/50">
      <span className="text-slate-500">{label}</span>
      <span className="text-white">{value}</span>
    </div>
  )
}
