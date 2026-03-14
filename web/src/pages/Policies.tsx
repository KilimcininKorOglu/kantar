export default function Policies() {
  const policies = [
    { name: 'License', description: 'Allowed/blocked license types', action: 'block', rules: 'Allowed: MIT, Apache-2.0, BSD. Blocked: GPL-3.0, AGPL-3.0' },
    { name: 'Vulnerability', description: 'Severity-based blocking', action: 'block', rules: 'Block: critical, high. Warn: medium. Allow: low' },
    { name: 'Age', description: 'Minimum package age requirement', action: 'warn', rules: 'Min age: 7 days, Min maintainers: 2' },
    { name: 'Size', description: 'Maximum package size limits', action: 'block', rules: 'Max size: 500MB, Max layers: 20' },
    { name: 'Version', description: 'Version constraints', action: 'block', rules: 'Pre-release: blocked, Deprecated: blocked' },
    { name: 'Naming', description: 'Package naming rules', action: 'warn', rules: 'Blocked scopes and prefixes' },
  ]

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-semibold text-white">Policies</h2>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {policies.map((policy) => (
          <div key={policy.name} className="bg-slate-900 border border-slate-800 rounded-lg p-4">
            <div className="flex items-center justify-between mb-2">
              <h3 className="font-medium text-white">{policy.name}</h3>
              <span className={`px-2 py-0.5 text-xs rounded ${
                policy.action === 'block'
                  ? 'bg-red-900/40 text-red-300'
                  : 'bg-yellow-900/40 text-yellow-300'
              }`}>
                {policy.action}
              </span>
            </div>
            <p className="text-sm text-slate-400 mb-3">{policy.description}</p>
            <p className="text-xs text-slate-500">{policy.rules}</p>
          </div>
        ))}
      </div>
    </div>
  )
}
