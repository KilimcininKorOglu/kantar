import { useState, useEffect } from 'react'
import { api } from '../api/client'
import { Pencil, X } from 'lucide-react'

interface RegistryInfo {
  id: number; ecosystem: string; mode: string; upstream: string
  autoSync: boolean; autoSyncInterval: string; maxVersions: number
  enabled: boolean; createdAt: string; updatedAt: string
}

const ecoNames: Record<string, string> = {
  docker: 'Docker', npm: 'npm', pypi: 'PyPI', gomod: 'Go Modules',
  cargo: 'Cargo', maven: 'Maven', nuget: 'NuGet', helm: 'Helm',
}

export default function Registries() {
  const [registries, setRegistries] = useState<RegistryInfo[]>([])
  const [editing, setEditing] = useState<string | null>(null)
  const [editData, setEditData] = useState<Partial<RegistryInfo>>({})
  const [saving, setSaving] = useState(false)

  const load = () => { api.get<RegistryInfo[]>('/registries').then(setRegistries).catch(() => {}) }
  useEffect(() => { load() }, [])

  const handleSave = async (eco: string) => {
    setSaving(true)
    try { await api.put(`/registries/${eco}`, editData); setEditing(null); load() } catch {}
    setSaving(false)
  }

  const handleToggle = async (eco: string, current: boolean) => {
    await api.put(`/registries/${eco}`, { enabled: !current }); load()
  }

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
        {registries.map(reg => (
          <div key={reg.ecosystem} className={`bg-surface border border-border rounded p-4 ${!reg.enabled ? 'opacity-50' : ''}`}>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold text-text">{ecoNames[reg.ecosystem] || reg.ecosystem}</h3>
              <button onClick={() => handleToggle(reg.ecosystem, reg.enabled)}
                className={`w-8 h-4 rounded-full cursor-pointer relative ${reg.enabled ? 'bg-accent' : 'bg-border'}`}>
                <span className={`absolute top-0.5 w-3 h-3 rounded-full bg-white ${reg.enabled ? 'left-4' : 'left-0.5'}`} />
              </button>
            </div>

            {editing === reg.ecosystem ? (
              <div className="space-y-2">
                <div>
                  <label className="text-[10px] text-text-dim uppercase">Mode</label>
                  <select value={editData.mode || reg.mode} onChange={e => setEditData({ ...editData, mode: e.target.value })}
                    className="w-full bg-surface-2 border border-border rounded px-2 py-1 text-xs text-text">
                    <option value="allowlist">allowlist</option><option value="mirror">mirror</option>
                  </select>
                </div>
                <div>
                  <label className="text-[10px] text-text-dim uppercase">Upstream</label>
                  <input value={editData.upstream ?? reg.upstream} onChange={e => setEditData({ ...editData, upstream: e.target.value })}
                    className="w-full bg-surface-2 border border-border rounded px-2 py-1 text-xs text-text focus:outline-none focus:border-accent" />
                </div>
                <div className="flex gap-2">
                  <button onClick={() => handleSave(reg.ecosystem)} disabled={saving} className="px-3 py-1 bg-accent hover:bg-accent-hover text-white text-[11px] rounded cursor-pointer">Save</button>
                  <button onClick={() => setEditing(null)} className="px-3 py-1 bg-surface-2 text-text-muted text-[11px] rounded cursor-pointer"><X className="w-3 h-3 inline" /></button>
                </div>
              </div>
            ) : (
              <>
                <div className="space-y-1.5 text-xs">
                  <div className="flex justify-between"><span className="text-text-dim">Mode</span><span className="text-text font-mono">{reg.mode}</span></div>
                  <div className="flex justify-between"><span className="text-text-dim">Upstream</span><span className="text-text-muted font-mono text-[11px] truncate ml-2 max-w-[140px]">{reg.upstream || '—'}</span></div>
                </div>
                <button onClick={() => { setEditing(reg.ecosystem); setEditData({}) }}
                  className="flex items-center gap-1 mt-3 w-full justify-center py-1.5 bg-surface-2 border border-border text-text-muted text-[11px] rounded cursor-pointer">
                  <Pencil className="w-3 h-3" /> Edit
                </button>
              </>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
