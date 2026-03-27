import { useState, useEffect } from 'react'
import { api } from '../api/client'

interface RegistryInfo {
  id: number
  ecosystem: string
  mode: string
  upstream: string
  autoSync: boolean
  autoSyncInterval: string
  maxVersions: number
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export default function Registries() {
  const [registries, setRegistries] = useState<RegistryInfo[]>([])
  const [editing, setEditing] = useState<string | null>(null)
  const [editData, setEditData] = useState<Partial<RegistryInfo>>({})
  const [saving, setSaving] = useState(false)

  const loadRegistries = () => {
    api.get<RegistryInfo[]>('/registries').then(setRegistries).catch(() => {})
  }

  useEffect(() => { loadRegistries() }, [])

  const handleSave = async (ecosystem: string) => {
    setSaving(true)
    try {
      await api.put(`/registries/${ecosystem}`, editData)
      setEditing(null)
      loadRegistries()
    } catch { /* ignore */ }
    setSaving(false)
  }

  const handleToggle = async (ecosystem: string, currentEnabled: boolean) => {
    await api.put(`/registries/${ecosystem}`, { enabled: !currentEnabled })
    loadRegistries()
  }

  const ecoNames: Record<string, string> = {
    docker: 'Docker', npm: 'npm', pypi: 'PyPI', gomod: 'Go Modules',
    cargo: 'Cargo', maven: 'Maven', nuget: 'NuGet', helm: 'Helm',
  }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-semibold text-white">Registries</h2>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {registries.map(reg => (
          <div key={reg.ecosystem} className={`bg-slate-900 border rounded-lg p-4 ${reg.enabled ? 'border-slate-800' : 'border-slate-800/50 opacity-60'}`}>
            <div className="flex items-center justify-between mb-3">
              <h3 className="font-medium text-white">{ecoNames[reg.ecosystem] || reg.ecosystem}</h3>
              <button
                onClick={() => handleToggle(reg.ecosystem, reg.enabled)}
                className={`w-10 h-5 rounded-full transition-colors cursor-pointer relative ${reg.enabled ? 'bg-emerald-600' : 'bg-slate-700'}`}
              >
                <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${reg.enabled ? 'left-5' : 'left-0.5'}`} />
              </button>
            </div>

            {editing === reg.ecosystem ? (
              <div className="space-y-2">
                <div>
                  <label className="text-xs text-slate-500">Mode</label>
                  <select
                    value={editData.mode || reg.mode}
                    onChange={e => setEditData({ ...editData, mode: e.target.value })}
                    className="w-full bg-slate-800 border border-slate-700 rounded px-2 py-1 text-sm text-white"
                  >
                    <option value="allowlist">allowlist</option>
                    <option value="mirror">mirror</option>
                  </select>
                </div>
                <div>
                  <label className="text-xs text-slate-500">Upstream</label>
                  <input
                    value={editData.upstream ?? reg.upstream}
                    onChange={e => setEditData({ ...editData, upstream: e.target.value })}
                    className="w-full bg-slate-800 border border-slate-700 rounded px-2 py-1 text-sm text-white focus:outline-none focus:border-blue-500"
                  />
                </div>
                <div className="flex gap-2 mt-2">
                  <button
                    onClick={() => handleSave(reg.ecosystem)}
                    disabled={saving}
                    className="px-3 py-1 bg-blue-600 hover:bg-blue-500 text-white text-xs rounded cursor-pointer"
                  >Save</button>
                  <button
                    onClick={() => setEditing(null)}
                    className="px-3 py-1 bg-slate-700 hover:bg-slate-600 text-white text-xs rounded cursor-pointer"
                  >Cancel</button>
                </div>
              </div>
            ) : (
              <>
                <div className="space-y-1.5 text-sm">
                  <div className="flex justify-between">
                    <span className="text-slate-500">Mode</span>
                    <span className="text-slate-300">{reg.mode}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-500">Upstream</span>
                    <span className="text-slate-300 text-xs truncate ml-2">{reg.upstream || '—'}</span>
                  </div>
                </div>
                <button
                  onClick={() => { setEditing(reg.ecosystem); setEditData({}) }}
                  className="w-full mt-3 px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs rounded transition-colors cursor-pointer"
                >Edit</button>
              </>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
