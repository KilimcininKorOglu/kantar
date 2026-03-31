import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api/client'
import type { SystemStatus } from '../api/types'
import { useAuth } from '../hooks/useAuth'
import { getTimezone, setTimezone as setTz, getTimezoneList } from '../utils/date'
import { setLocale, getLocale, supportedLocales } from '../i18n'
import { Cpu, HardDrive, Activity, Clock, Globe, Languages, Pencil, Check, X } from 'lucide-react'

interface Setting {
  key: string; value: string; category: string; description: string; updatedAt: string
}

export default function Settings() {
  const { t } = useTranslation()
  const { user } = useAuth()
  const [status, setStatus] = useState<SystemStatus | null>(null)
  const [settings, setSettings] = useState<Setting[]>([])
  const [editingKey, setEditingKey] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const [saving, setSaving] = useState(false)
  const [timezone, setTimezone] = useState(getTimezone())
  const [locale, setLang] = useState(getLocale())

  useEffect(() => {
    api.get<SystemStatus>('/system/status').then(setStatus).catch(() => {})
    api.get<Setting[]>('/settings').then(setSettings).catch(() => {})
  }, [])

  const categories = [...new Set(settings.map(s => s.category))]

  const handleSave = async (key: string) => {
    setSaving(true)
    try {
      await api.put(`/settings/${key}`, { value: editValue })
      setSettings(prev => prev.map(s => s.key === key ? { ...s, value: editValue } : s))
      setEditingKey(null)
    } catch {}
    setSaving(false)
  }

  const handleTimezoneChange = async (tz: string) => {
    setTimezone(tz); setTz(tz)
    if (user) { try { await api.put(`/users/${user.id}`, { timezone: tz }) } catch {} }
  }

  const handleLocaleChange = async (lng: string) => {
    setLang(lng); setLocale(lng)
    if (user) { try { await api.put(`/users/${user.id}`, { locale: lng }) } catch {} }
  }

  return (
    <div className="space-y-5">
      {/* System Info */}
      <div className="bg-surface border border-border rounded p-4">
        <h3 className="text-xs font-semibold text-text-dim uppercase tracking-wider mb-4">{t('settings.systemInfo')}</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <SysInfo icon={Activity} label="Status" value={status?.status || '—'} />
          <SysInfo icon={Clock} label="Uptime" value={status?.uptime || '—'} />
          <SysInfo icon={Cpu} label="Go" value={status?.goVersion || '—'} mono />
          <SysInfo icon={HardDrive} label="Memory" value={status ? `${(status.memory.allocBytes / 1048576).toFixed(1)} MB` : '—'} mono />
        </div>
      </div>

      {/* Display Preferences */}
      <div className="bg-surface border border-border rounded p-4">
        <h3 className="text-xs font-semibold text-text-dim uppercase tracking-wider mb-3">{t('settings.display')}</h3>
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Globe className="w-3.5 h-3.5 text-text-dim" />
              <div>
                <div className="text-xs text-text">{t('settings.timezone')}</div>
                <div className="text-[10px] text-text-dim">{t('settings.timezoneDesc')}</div>
              </div>
            </div>
            <select value={timezone} onChange={e => handleTimezoneChange(e.target.value)}
              className="bg-surface-2 border border-border rounded px-3 py-1.5 text-xs text-text w-56">
              {getTimezoneList().map(tz => <option key={tz} value={tz}>{tz}</option>)}
            </select>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Languages className="w-3.5 h-3.5 text-text-dim" />
              <div>
                <div className="text-xs text-text">{t('settings.language')}</div>
                <div className="text-[10px] text-text-dim">{t('settings.languageDesc')}</div>
              </div>
            </div>
            <select value={locale} onChange={e => handleLocaleChange(e.target.value)}
              className="bg-surface-2 border border-border rounded px-3 py-1.5 text-xs text-text w-56">
              {supportedLocales.map(l => <option key={l.code} value={l.code}>{l.label}</option>)}
            </select>
          </div>
        </div>
      </div>

      {/* Dynamic Settings */}
      {categories.map(cat => (
        <div key={cat} className="bg-surface border border-border rounded p-4">
          <h3 className="text-xs font-semibold text-text-dim uppercase tracking-wider mb-3 capitalize">{cat}</h3>
          <div className="space-y-1">
            {settings.filter(s => s.category === cat).map(s => (
              <div key={s.key} className="flex items-center justify-between py-2 border-b border-border last:border-0">
                <div className="flex-1 min-w-0">
                  <div className="text-xs text-text font-mono">{s.key}</div>
                  {s.description && <div className="text-[10px] text-text-dim truncate">{s.description}</div>}
                </div>
                <div className="flex items-center gap-2 ml-4">
                  {editingKey === s.key ? (
                    <>
                      <input value={editValue} onChange={e => setEditValue(e.target.value)}
                        onKeyDown={e => e.key === 'Enter' && handleSave(s.key)}
                        className="bg-surface-2 border border-border rounded px-2 py-1 text-xs text-text w-44 focus:outline-none focus:border-accent font-mono" autoFocus />
                      <button onClick={() => handleSave(s.key)} disabled={saving} className="text-accent cursor-pointer"><Check className="w-3.5 h-3.5" /></button>
                      <button onClick={() => setEditingKey(null)} className="text-text-dim cursor-pointer"><X className="w-3.5 h-3.5" /></button>
                    </>
                  ) : (
                    <>
                      <span className="text-xs text-text-muted font-mono">{s.value}</span>
                      <button onClick={() => { setEditingKey(s.key); setEditValue(s.value) }} className="text-text-dim hover:text-accent cursor-pointer"><Pencil className="w-3 h-3" /></button>
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

function SysInfo({ icon: Icon, label, value, mono }: { icon: typeof Cpu; label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-center gap-2">
      <Icon className="w-3.5 h-3.5 text-text-dim" />
      <div>
        <div className="text-[10px] text-text-dim uppercase">{label}</div>
        <div className={`text-xs text-text ${mono ? 'font-mono' : ''}`}>{value}</div>
      </div>
    </div>
  )
}
