import { useState, useEffect, type FormEvent } from 'react'
import { api } from '../api/client'

interface PolicyInfo {
  id: number
  name: string
  policyType: string
  config: Record<string, any>
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export default function Policies() {
  const [policies, setPolicies] = useState<PolicyInfo[]>([])
  const [editing, setEditing] = useState<string | null>(null)
  const [editConfig, setEditConfig] = useState<Record<string, any>>({})
  const [saving, setSaving] = useState(false)

  const loadPolicies = () => {
    api.get<PolicyInfo[]>('/policies').then(setPolicies).catch(() => {})
  }

  useEffect(() => { loadPolicies() }, [])

  const handleToggle = async (name: string) => {
    await api.put(`/policies/${name}/toggle`)
    loadPolicies()
  }

  const handleSave = async (name: string, e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    try {
      await api.put(`/policies/${name}`, { config: editConfig })
      setEditing(null)
      loadPolicies()
    } catch { /* ignore */ }
    setSaving(false)
  }

  const policyDescriptions: Record<string, string> = {
    license: 'Control which software licenses are allowed or blocked',
    vulnerability: 'Block packages based on vulnerability severity',
    age: 'Require minimum package age before approval',
    size: 'Enforce maximum package size limits',
    version: 'Control pre-release and deprecated versions',
    naming: 'Block packages by scope or name prefix',
  }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-semibold text-white">Policies</h2>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {policies.map(policy => (
          <div key={policy.name} className={`bg-slate-900 border rounded-lg p-4 ${policy.enabled ? 'border-slate-800' : 'border-slate-800/50 opacity-60'}`}>
            <div className="flex items-center justify-between mb-2">
              <h3 className="font-medium text-white capitalize">{policy.name}</h3>
              <button
                onClick={() => handleToggle(policy.name)}
                className={`w-10 h-5 rounded-full transition-colors cursor-pointer relative ${policy.enabled ? 'bg-emerald-600' : 'bg-slate-700'}`}
              >
                <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${policy.enabled ? 'left-5' : 'left-0.5'}`} />
              </button>
            </div>
            <p className="text-sm text-slate-400 mb-3">{policyDescriptions[policy.policyType] || policy.policyType}</p>

            {editing === policy.name ? (
              <form onSubmit={e => handleSave(policy.name, e)} className="space-y-2">
                <PolicyForm policyType={policy.policyType} config={editConfig} onChange={setEditConfig} />
                <div className="flex gap-2 mt-2">
                  <button type="submit" disabled={saving} className="px-3 py-1 bg-blue-600 hover:bg-blue-500 text-white text-xs rounded cursor-pointer">Save</button>
                  <button type="button" onClick={() => setEditing(null)} className="px-3 py-1 bg-slate-700 hover:bg-slate-600 text-white text-xs rounded cursor-pointer">Cancel</button>
                </div>
              </form>
            ) : (
              <>
                <PolicyDisplay policyType={policy.policyType} config={policy.config} />
                <button
                  onClick={() => { setEditing(policy.name); setEditConfig({ ...policy.config }) }}
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

function PolicyDisplay({ config }: { policyType: string; config: Record<string, any> }) {
  return (
    <div className="space-y-1 text-xs text-slate-500">
      {Object.entries(config).map(([key, val]) => (
        <div key={key} className="flex justify-between">
          <span>{key}</span>
          <span className="text-slate-300 truncate ml-2 max-w-[150px]">{Array.isArray(val) ? val.join(', ') : String(val)}</span>
        </div>
      ))}
    </div>
  )
}

function PolicyForm({ policyType, config, onChange }: {
  policyType: string
  config: Record<string, any>
  onChange: (c: Record<string, any>) => void
}) {
  const updateField = (key: string, value: any) => {
    onChange({ ...config, [key]: value })
  }

  const updateArrayField = (key: string, value: string) => {
    const arr = value.split(',').map(s => s.trim()).filter(Boolean)
    onChange({ ...config, [key]: arr })
  }

  switch (policyType) {
    case 'license':
      return (
        <div className="space-y-2">
          <FieldInput label="Allowed licenses" value={(config.allowed || []).join(', ')} onChange={v => updateArrayField('allowed', v)} />
          <FieldInput label="Blocked licenses" value={(config.blocked || []).join(', ')} onChange={v => updateArrayField('blocked', v)} />
          <FieldSelect label="Action" value={config.action || 'block'} options={['block', 'warn', 'log']} onChange={v => updateField('action', v)} />
        </div>
      )
    case 'vulnerability':
      return (
        <div className="space-y-2">
          <FieldInput label="Block severity" value={(config.block_severity || []).join(', ')} onChange={v => updateArrayField('block_severity', v)} />
          <FieldInput label="Warn severity" value={(config.warn_severity || []).join(', ')} onChange={v => updateArrayField('warn_severity', v)} />
          <FieldInput label="Allow severity" value={(config.allow_severity || []).join(', ')} onChange={v => updateArrayField('allow_severity', v)} />
        </div>
      )
    case 'age':
      return (
        <div className="space-y-2">
          <FieldInput label="Min package age" value={config.min_package_age || '7d'} onChange={v => updateField('min_package_age', v)} />
        </div>
      )
    case 'size':
      return (
        <div className="space-y-2">
          <FieldInput label="Max package size" value={config.max_package_size || '500MB'} onChange={v => updateField('max_package_size', v)} />
        </div>
      )
    case 'version':
      return (
        <div className="space-y-2">
          <FieldToggle label="Allow prerelease" value={config.allow_prerelease ?? false} onChange={v => updateField('allow_prerelease', v)} />
          <FieldToggle label="Allow deprecated" value={config.allow_deprecated ?? false} onChange={v => updateField('allow_deprecated', v)} />
        </div>
      )
    case 'naming':
      return (
        <div className="space-y-2">
          <FieldInput label="Blocked scopes" value={(config.blocked_scopes || []).join(', ')} onChange={v => updateArrayField('blocked_scopes', v)} />
          <FieldInput label="Blocked prefixes" value={(config.blocked_prefixes || []).join(', ')} onChange={v => updateArrayField('blocked_prefixes', v)} />
        </div>
      )
    default:
      return <FieldInput label="Config (JSON)" value={JSON.stringify(config)} onChange={v => { try { onChange(JSON.parse(v)) } catch {} }} />
  }
}

function FieldInput({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <div>
      <label className="text-xs text-slate-500 block mb-0.5">{label}</label>
      <input
        value={value}
        onChange={e => onChange(e.target.value)}
        className="w-full bg-slate-800 border border-slate-700 rounded px-2 py-1 text-sm text-white focus:outline-none focus:border-blue-500"
      />
    </div>
  )
}

function FieldSelect({ label, value, options, onChange }: { label: string; value: string; options: string[]; onChange: (v: string) => void }) {
  return (
    <div>
      <label className="text-xs text-slate-500 block mb-0.5">{label}</label>
      <select value={value} onChange={e => onChange(e.target.value)} className="w-full bg-slate-800 border border-slate-700 rounded px-2 py-1 text-sm text-white">
        {options.map(o => <option key={o} value={o}>{o}</option>)}
      </select>
    </div>
  )
}

function FieldToggle({ label, value, onChange }: { label: string; value: boolean; onChange: (v: boolean) => void }) {
  return (
    <div className="flex items-center justify-between">
      <label className="text-xs text-slate-500">{label}</label>
      <button
        type="button"
        onClick={() => onChange(!value)}
        className={`w-10 h-5 rounded-full transition-colors cursor-pointer relative ${value ? 'bg-emerald-600' : 'bg-slate-700'}`}
      >
        <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${value ? 'left-5' : 'left-0.5'}`} />
      </button>
    </div>
  )
}
