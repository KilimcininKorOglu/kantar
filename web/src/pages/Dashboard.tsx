import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api/client'
import type { SystemStatus, AuditLogEntry } from '../api/types'
import { formatDateTime } from '../utils/date'
import { Activity, Cpu, HardDrive, Box, AlertTriangle, Download, Timer } from 'lucide-react'

const ecosystems = ['Docker', 'npm', 'PyPI', 'Go Mod', 'Cargo', 'Maven', 'NuGet', 'Helm']

export default function Dashboard() {
  const { t } = useTranslation()
  const [status, setStatus] = useState<SystemStatus | null>(null)
  const [recentActivity, setRecentActivity] = useState<AuditLogEntry[]>([])

  useEffect(() => {
    const fetchStatus = () => {
      api.get<SystemStatus>('/system/status').then(setStatus).catch(() => {})
    }
    fetchStatus()
    const interval = setInterval(fetchStatus, 30000)
    return () => clearInterval(interval)
  }, [])

  useEffect(() => {
    api.get<AuditLogEntry[]>('/audit?limit=5').then(setRecentActivity).catch(() => {})
  }, [])

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard icon={Box} label={t('dashboard.packages')} value="—" sub={t('dashboard.total')} />
        <StatCard icon={AlertTriangle} label={t('dashboard.pending')} value="—" sub={t('dashboard.awaitingApproval')} />
        <StatCard icon={Download} label={t('dashboard.downloads')} value="—" sub={t('dashboard.today')} />
        <StatCard icon={Timer} label={t('dashboard.uptime')} value={status?.uptime || '—'} sub={status?.status || t('dashboard.loadingStatus')} />
      </div>

      <div>
        <h3 className="text-xs font-semibold text-text-dim uppercase tracking-wider mb-3">{t('dashboard.registryHealth')}</h3>
        <div className="grid grid-cols-4 lg:grid-cols-8 gap-2">
          {ecosystems.map((eco) => (
            <div key={eco} className="bg-surface border border-border rounded px-3 py-2.5 text-center">
              <div className="text-[11px] text-text-dim mb-1">{eco}</div>
              <div className="flex items-center justify-center gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-success" />
                <span className="text-xs font-medium text-text">{t('dashboard.ok')}</span>
              </div>
            </div>
          ))}
        </div>
      </div>

      <div>
        <h3 className="text-xs font-semibold text-text-dim uppercase tracking-wider mb-3">{t('dashboard.pendingApprovals')}</h3>
        <div className="bg-surface border border-border rounded overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-text-dim">
                <th className="text-left px-4 py-2.5 text-xs font-medium">{t('dashboard.package')}</th>
                <th className="text-left px-4 py-2.5 text-xs font-medium">{t('dashboard.version')}</th>
                <th className="text-left px-4 py-2.5 text-xs font-medium">{t('dashboard.registry')}</th>
                <th className="text-left px-4 py-2.5 text-xs font-medium">{t('dashboard.requestedBy')}</th>
                <th className="text-right px-4 py-2.5 text-xs font-medium">{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              <tr><td colSpan={5} className="px-4 py-8 text-center text-text-dim text-xs">{t('dashboard.noPendingApprovals')}</td></tr>
            </tbody>
          </table>
        </div>
      </div>

      <div>
        <h3 className="text-xs font-semibold text-text-dim uppercase tracking-wider mb-3">{t('dashboard.recentActivity')}</h3>
        <div className="bg-surface border border-border rounded p-4">
          {recentActivity.length === 0 ? (
            <p className="text-xs text-text-dim text-center py-4">{t('dashboard.noRecentActivity')}</p>
          ) : (
            <div className="space-y-1">
              {recentActivity.map((a) => (
                <div key={a.id} className="flex items-center justify-between py-1.5 border-b border-border last:border-0">
                  <div className="flex items-center gap-3">
                    <Activity className="w-3 h-3 text-text-dim" />
                    <span className="text-xs font-mono text-accent">{a.event}</span>
                    <span className="text-xs text-text">{a.actorUsername}</span>
                  </div>
                  <span className="text-[11px] text-text-dim font-mono">{formatDateTime(a.timestamp)}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {status && (
        <div>
          <h3 className="text-xs font-semibold text-text-dim uppercase tracking-wider mb-3">{t('dashboard.system')}</h3>
          <div className="bg-surface border border-border rounded p-4 grid grid-cols-2 md:grid-cols-4 gap-4 text-xs">
            <div className="flex items-center gap-2"><Cpu className="w-3.5 h-3.5 text-text-dim" /><span className="text-text-muted">Go:</span><span className="text-text font-mono">{status.goVersion}</span></div>
            <div className="flex items-center gap-2"><Cpu className="w-3.5 h-3.5 text-text-dim" /><span className="text-text-muted">CPUs:</span><span className="text-text font-mono">{status.numCpu}</span></div>
            <div className="flex items-center gap-2"><Activity className="w-3.5 h-3.5 text-text-dim" /><span className="text-text-muted">Goroutines:</span><span className="text-text font-mono">{status.goroutines}</span></div>
            <div className="flex items-center gap-2"><HardDrive className="w-3.5 h-3.5 text-text-dim" /><span className="text-text-muted">Memory:</span><span className="text-text font-mono">{(status.memory.allocBytes / 1048576).toFixed(1)} MB</span></div>
          </div>
        </div>
      )}
    </div>
  )
}

function StatCard({ icon: Icon, label, value, sub }: { icon: typeof Box; label: string; value: string; sub: string }) {
  return (
    <div className="bg-surface border border-border rounded p-4">
      <div className="flex items-center gap-2 mb-2">
        <Icon className="w-3.5 h-3.5 text-text-dim" />
        <span className="text-[11px] text-text-dim uppercase tracking-wider">{label}</span>
      </div>
      <div className="text-xl font-bold text-text font-mono">{value}</div>
      <div className="text-[11px] text-text-dim mt-0.5">{sub}</div>
    </div>
  )
}
