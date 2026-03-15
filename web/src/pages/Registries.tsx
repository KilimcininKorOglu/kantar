const registries = [
  { name: 'Docker', key: 'docker', upstream: 'registry-1.docker.io', mode: 'allowlist' },
  { name: 'npm', key: 'npm', upstream: 'registry.npmjs.org', mode: 'allowlist' },
  { name: 'PyPI', key: 'pypi', upstream: 'pypi.org', mode: 'allowlist' },
  { name: 'Go Modules', key: 'gomod', upstream: 'proxy.golang.org', mode: 'allowlist' },
  { name: 'Cargo', key: 'cargo', upstream: 'crates.io', mode: 'allowlist' },
  { name: 'Maven', key: 'maven', upstream: 'repo1.maven.org', mode: 'allowlist' },
  { name: 'NuGet', key: 'nuget', upstream: 'api.nuget.org', mode: 'allowlist' },
  { name: 'Helm', key: 'helm', upstream: '—', mode: 'allowlist' },
]

export default function Registries() {
  return (
    <div className="space-y-4">
      <h2 className="text-xl font-semibold text-white">Registries</h2>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {registries.map((reg) => (
          <div key={reg.key} className="bg-slate-900 border border-slate-800 rounded-lg p-4">
            <div className="flex items-center justify-between mb-3">
              <h3 className="font-medium text-white">{reg.name}</h3>
              <span className="w-2 h-2 rounded-full bg-emerald-500" />
            </div>
            <div className="space-y-1.5 text-sm">
              <div className="flex justify-between">
                <span className="text-slate-500">Mode</span>
                <span className="text-slate-300">{reg.mode}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Upstream</span>
                <span className="text-slate-300 text-xs">{reg.upstream}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Packages</span>
                <span className="text-white">0</span>
              </div>
            </div>
            <button className="w-full mt-3 px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs rounded transition-colors cursor-pointer">
              Sync Now
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
