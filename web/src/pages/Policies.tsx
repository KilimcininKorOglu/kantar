import { useState, useEffect, type FormEvent } from 'react'
import { api } from '../api/client'
import { Pencil, X } from 'lucide-react'

interface PolicyInfo {
  id: number; name: string; policyType: string
  config: Record<string, any>; enabled: boolean
  createdAt: string; updatedAt: string
}

const policyDescriptions: Record<string, string> = {
  license: 'Control which software licenses are allowed or blocked',
  vulnerability: 'Block packages based on vulnerability severity',
  age: 'Require minimum package age before approval',
  size: 'Enforce maximum package size limits',
  version: 'Control pre-release and deprecated versions',
  naming: 'Block packages by scope or name prefix',
}

export default function Policies() {
  const [policies, setPolicies] = useState<PolicyInfo[]>([])
  const [editing, setEditing] = useState<string | null>(null)
  const [editConfig, setEditConfig] = useState<Record<string, any>>({})
  const [saving, setSaving] = useState(false)

  const load = () => { api.get<PolicyInfo[]>('/policies').then(setPolicies).catch(() => {}) }
  useEffect(() => { load() }, [])

  const handleToggle = async (name: string) => { await api.put(`/policies/${name}/toggle`); load() }

  const handleSave = async (name: string, e: FormEvent) => {
    e.preventDefault(); setSaving(true)
    try { await api.put(`/policies/${name}`, { config: editConfig }); setEditing(null); load() } catch {}
    setSaving(false)
  }

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
        {policies.map(policy => (
          <div key={policy.name} className={`bg-surface border border-border rounded p-4 ${!policy.enabled ? 'opacity-50' : ''}`}>
            <div className="flex items-center justify-between mb-1.5">
              <h3 className="text-sm font-semibold text-text capitalize">{policy.name}</h3>
              <button onClick={() => handleToggle(policy.name)}
                className={`w-8 h-4 rounded-full cursor-pointer relative ${policy.enabled ? 'bg-accent' : 'bg-border'}`}>
                <span className={`absolute top-0.5 w-3 h-3 rounded-full bg-white ${policy.enabled ? 'left-4' : 'left-0.5'}`} />
              </button>
            </div>
            <p className="text-[11px] text-text-dim mb-3">{policyDescriptions[policy.policyType] || policy.policyType}</p>

            {editing === policy.name ? (
              <form onSubmit={e => handleSave(policy.name, e)} className="space-y-2">
                <PolicyForm policyType={policy.policyType} config={editConfig} onChange={setEditConfig} />
                <div className="flex gap-2 pt-1">
                  <button type="submit" disabled={saving} className="px-3 py-1 bg-accent hover:bg-accent-hover text-white text-[11px] rounded cursor-pointer">Save</button>
                  <button type="button" onClick={() => setEditing(null)} className="px-3 py-1 bg-surface-2 text-text-muted text-[11px] rounded cursor-pointer"><X className="w-3 h-3 inline" /></button>
                </div>
              </form>
            ) : (
              <>
                <div className="space-y-1">
                  {Object.entries(policy.config).map(([key, val]) => (
                    <div key={key} className="flex justify-between text-[11px]">
                      <span className="text-text-dim">{key}</span>
                      <span className="text-text-muted font-mono truncate ml-2 max-w-[140px]">{Array.isArray(val) ? val.join(', ') : String(val)}</span>
                    </div>
                  ))}
                </div>
                <button onClick={() => { setEditing(policy.name); setEditConfig({ ...policy.config }) }}
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

function PolicyForm({ policyType, config, onChange }: { policyType: string; config: Record<string, any>; onChange: (c: Record<string, any>) => void }) {
  const updateField = (key: string, value: any) => onChange({ ...config, [key]: value })
  const updateArray = (key: string, value: string) => onChange({ ...config, [key]: value.split(',').map(s => s.trim()).filter(Boolean) })
  const inputCls = "w-full bg-surface-2 border border-border rounded px-2 py-1 text-[11px] text-text focus:outline-none focus:border-accent"
  const labelCls = "text-[10px] text-text-dim uppercase block mb-0.5"

  switch (policyType) {
    case 'license': return (<>
      <div><label className={labelCls}>Allowed</label><input value={(config.allowed || []).join(', ')} onChange={e => updateArray('allowed', e.target.value)} className={inputCls} /></div>
      <div><label className={labelCls}>Blocked</label><input value={(config.blocked || []).join(', ')} onChange={e => updateArray('blocked', e.target.value)} className={inputCls} /></div>
      <div><label className={labelCls}>Action</label>
        <select value={config.action || 'block'} onChange={e => updateField('action', e.target.value)} className={inputCls}>
          <option value="block">block</option><option value="warn">warn</option><option value="log">log</option>
        </select></div>
    </>)
    case 'vulnerability': return (<>
      <div><label className={labelCls}>Block severity</label><input value={(config.block_severity || []).join(', ')} onChange={e => updateArray('block_severity', e.target.value)} className={inputCls} /></div>
      <div><label className={labelCls}>Warn severity</label><input value={(config.warn_severity || []).join(', ')} onChange={e => updateArray('warn_severity', e.target.value)} className={inputCls} /></div>
    </>)
    case 'age': return (<div><label className={labelCls}>Min package age</label><input value={config.min_package_age || '7d'} onChange={e => updateField('min_package_age', e.target.value)} className={inputCls} /></div>)
    case 'size': return (<div><label className={labelCls}>Max package size</label><input value={config.max_package_size || '500MB'} onChange={e => updateField('max_package_size', e.target.value)} className={inputCls} /></div>)
    case 'version': return (<>
      <ToggleRow label="Allow prerelease" value={config.allow_prerelease ?? false} onChange={v => updateField('allow_prerelease', v)} />
      <ToggleRow label="Allow deprecated" value={config.allow_deprecated ?? false} onChange={v => updateField('allow_deprecated', v)} />
    </>)
    case 'naming': return (<>
      <div><label className={labelCls}>Blocked scopes</label><input value={(config.blocked_scopes || []).join(', ')} onChange={e => updateArray('blocked_scopes', e.target.value)} className={inputCls} /></div>
      <div><label className={labelCls}>Blocked prefixes</label><input value={(config.blocked_prefixes || []).join(', ')} onChange={e => updateArray('blocked_prefixes', e.target.value)} className={inputCls} /></div>
    </>)
    default: return <div><label className={labelCls}>Config (JSON)</label><input value={JSON.stringify(config)} onChange={e => { try { onChange(JSON.parse(e.target.value)) } catch {} }} className={inputCls} /></div>
  }
}

function ToggleRow({ label, value, onChange }: { label: string; value: boolean; onChange: (v: boolean) => void }) {
  return (
    <div className="flex items-center justify-between py-0.5">
      <span className="text-[10px] text-text-dim uppercase">{label}</span>
      <button type="button" onClick={() => onChange(!value)}
        className={`w-8 h-4 rounded-full cursor-pointer relative ${value ? 'bg-accent' : 'bg-border'}`}>
        <span className={`absolute top-0.5 w-3 h-3 rounded-full bg-white ${value ? 'left-4' : 'left-0.5'}`} />
      </button>
    </div>
  )
}
