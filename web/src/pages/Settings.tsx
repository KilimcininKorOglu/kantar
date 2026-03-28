import { useState, useEffect } from 'react'
import { api } from '../api/client'
import type { SystemStatus } from '../api/types'
import { useAuth } from '../hooks/useAuth'
import { getTimezone, setTimezone as setTz, getTimezoneList } from '../utils/date'

interface Setting {
  key: string
  value: string
  category: string
  description: string
  updatedAt: string
}

export default function Settings() {
  const { user } = useAuth()
  const [status, setStatus] = useState<SystemStatus | null>(null)
  const [settings, setSettings] = useState<Setting[]>([])
  const [editingKey, setEditingKey] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const [saving, setSaving] = useState(false)
  const [timezone, setTimezone] = useState(getTimezone())

  useEffect(() => {
    api.get<SystemStatus>('/system/status').then(setStatus).catch(() => {})
    api.get<Setting[]>('/settings').then(setSettings).catch(() => {})
  }, [])

  const handleTimezoneChange = async (tz: string) => {
    setTimezone(tz)
    setTz(tz)
    if (user) {
      try {
        await api.put(`/users/${user.id}`, { timezone: tz })
      } catch { /* ignore */ }
    }
  }

  const categories = [...new Set(settings.map(s => s.category))]

  const handleSave = async (key: string) => {
    setSaving(true)
    try {
      await api.put(`/settings/${key}`, { value: editValue })
      setSettings(prev => prev.map(s => s.key === key ? { ...s, value: editValue } : s))
      setEditingKey(null)
    } catch { /* ignore */ }
    setSaving(false)
  }

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

      {/* Display Preferences */}
      <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
        <h3 className="text-sm font-medium text-slate-400 mb-4">Display</h3>
        <div className="flex items-center justify-between">
          <div>
            <div className="text-sm text-white">Timezone</div>
            <div className="text-xs text-slate-500">All dates and times will be displayed in this timezone</div>
          </div>
          <select
            value={timezone}
            onChange={e => handleTimezoneChange(e.target.value)}
            className="bg-slate-800 border border-slate-700 rounded px-3 py-1.5 text-sm text-white w-64"
          >
            {getTimezoneList().map(tz => (
              <option key={tz} value={tz}>{tz}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Dynamic Settings by Category */}
      {categories.map(cat => (
        <div key={cat} className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <h3 className="text-sm font-medium text-slate-400 mb-4 capitalize">{cat}</h3>
          <div className="space-y-2">
            {settings.filter(s => s.category === cat).map(s => (
              <div key={s.key} className="flex items-center justify-between py-2 border-b border-slate-800/50 last:border-0">
                <div className="flex-1">
                  <div className="text-sm text-white">{s.key}</div>
                  {s.description && <div className="text-xs text-slate-500">{s.description}</div>}
                </div>
                <div className="flex items-center gap-2">
                  {editingKey === s.key ? (
                    <>
                      <input
                        value={editValue}
                        onChange={e => setEditValue(e.target.value)}
                        onKeyDown={e => e.key === 'Enter' && handleSave(s.key)}
                        className="bg-slate-800 border border-slate-600 rounded px-2 py-1 text-sm text-white w-48 focus:outline-none focus:border-blue-500"
                        autoFocus
                      />
                      <button
                        onClick={() => handleSave(s.key)}
                        disabled={saving}
                        className="px-2 py-1 bg-blue-600 hover:bg-blue-500 text-white text-xs rounded cursor-pointer"
                      >Save</button>
                      <button
                        onClick={() => setEditingKey(null)}
                        className="px-2 py-1 bg-slate-700 hover:bg-slate-600 text-white text-xs rounded cursor-pointer"
                      >Cancel</button>
                    </>
                  ) : (
                    <>
                      <span className="text-sm text-slate-300 font-mono">{s.value}</span>
                      <button
                        onClick={() => { setEditingKey(s.key); setEditValue(s.value) }}
                        className="px-2 py-1 text-blue-400 hover:text-blue-300 text-xs cursor-pointer"
                      >Edit</button>
                    </>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}
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
