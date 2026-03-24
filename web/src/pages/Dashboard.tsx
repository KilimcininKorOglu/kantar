import { useState, useEffect } from 'react'
import { api } from '../api/client'
import type { SystemStatus } from '../api/types'

const ecosystems = [
  { name: 'Docker', key: 'docker' },
  { name: 'npm', key: 'npm' },
  { name: 'PyPI', key: 'pypi' },
  { name: 'Go Mod', key: 'gomod' },
  { name: 'Cargo', key: 'cargo' },
  { name: 'Maven', key: 'maven' },
  { name: 'NuGet', key: 'nuget' },
  { name: 'Helm', key: 'helm' },
]

export default function Dashboard() {
  const [status, setStatus] = useState<SystemStatus | null>(null)

  useEffect(() => {
    const fetchStatus = () => {
      api.get<SystemStatus>('/system/status')
        .then(setStatus)
        .catch(() => {})
    }
    fetchStatus()
    const interval = setInterval(fetchStatus, 30000)
    return () => clearInterval(interval)
  }, [])

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-semibold text-white">Dashboard</h2>

      {/* Stats Row */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard label="Packages" value="—" sub="total" />
        <StatCard label="Pending" value="—" sub="awaiting approval" />
        <StatCard label="Downloads" value="—" sub="today" />
        <StatCard label="Uptime" value={status?.uptime || '—'} sub={status?.status || 'loading'} />
      </div>

      {/* Registry Health */}
      <div>
        <h3 className="text-sm font-medium text-slate-400 mb-3">Registry Health</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-8 gap-3">
          {ecosystems.map((eco) => (
            <div key={eco.key} className="bg-slate-900 border border-slate-800 rounded-lg p-3 text-center">
              <div className="text-xs text-slate-500 mb-1">{eco.name}</div>
              <div className="flex items-center justify-center gap-1.5">
                <span className="w-2 h-2 rounded-full bg-emerald-500" />
                <span className="text-sm text-white font-medium">OK</span>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Pending Approvals */}
      <div>
        <h3 className="text-sm font-medium text-slate-400 mb-3">Pending Approvals</h3>
        <div className="bg-slate-900 border border-slate-800 rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-800 text-slate-500">
                <th className="text-left px-4 py-2 font-medium">Package</th>
                <th className="text-left px-4 py-2 font-medium">Version</th>
                <th className="text-left px-4 py-2 font-medium">Registry</th>
                <th className="text-left px-4 py-2 font-medium">Requested By</th>
                <th className="text-right px-4 py-2 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-slate-600">
                  No pending approvals
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      {/* Recent Activity */}
      <div>
        <h3 className="text-sm font-medium text-slate-400 mb-3">Recent Activity</h3>
        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <p className="text-sm text-slate-600 text-center py-4">No recent activity</p>
        </div>
      </div>

      {/* System Info */}
      {status && (
        <div>
          <h3 className="text-sm font-medium text-slate-400 mb-3">System</h3>
          <div className="bg-slate-900 border border-slate-800 rounded-lg p-4 grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div><span className="text-slate-500">Go:</span> <span className="text-white">{status.goVersion}</span></div>
            <div><span className="text-slate-500">CPUs:</span> <span className="text-white">{status.numCpu}</span></div>
            <div><span className="text-slate-500">Goroutines:</span> <span className="text-white">{status.goroutines}</span></div>
            <div><span className="text-slate-500">Memory:</span> <span className="text-white">{(status.memory.allocBytes / 1048576).toFixed(1)} MB</span></div>
          </div>
        </div>
      )}
    </div>
  )
}

function StatCard({ label, value, sub }: { label: string; value: string; sub: string }) {
  return (
    <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
      <div className="text-xs text-slate-500 mb-1">{label}</div>
      <div className="text-2xl font-bold text-white">{value}</div>
      <div className="text-xs text-slate-500 mt-1">{sub}</div>
    </div>
  )
}
