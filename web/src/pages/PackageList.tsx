import { useState } from 'react'

const registries = ['docker', 'npm', 'pypi', 'gomod', 'cargo', 'maven', 'nuget', 'helm']

export default function PackageList() {
  const [activeRegistry, setActiveRegistry] = useState('npm')
  const [search, setSearch] = useState('')

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-white">Packages</h2>
        <input
          type="text"
          placeholder="Search packages..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="bg-slate-800 border border-slate-700 rounded px-3 py-1.5 text-sm text-white w-64 focus:outline-none focus:border-blue-500"
        />
      </div>

      {/* Registry Tabs */}
      <div className="flex gap-1 bg-slate-900 border border-slate-800 rounded-lg p-1">
        {registries.map((reg) => (
          <button
            key={reg}
            onClick={() => setActiveRegistry(reg)}
            className={`px-3 py-1.5 text-xs font-medium rounded transition-colors cursor-pointer ${
              activeRegistry === reg
                ? 'bg-blue-600 text-white'
                : 'text-slate-400 hover:text-white hover:bg-slate-800'
            }`}
          >
            {reg}
          </button>
        ))}
      </div>

      {/* Package Table */}
      <div className="bg-slate-900 border border-slate-800 rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-800 text-slate-500">
              <th className="text-left px-4 py-3 font-medium">Name</th>
              <th className="text-left px-4 py-3 font-medium">Version</th>
              <th className="text-left px-4 py-3 font-medium">Status</th>
              <th className="text-left px-4 py-3 font-medium">License</th>
              <th className="text-right px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td colSpan={5} className="px-4 py-12 text-center text-slate-600">
                No packages in <span className="text-slate-400">{activeRegistry}</span> registry
                {search && <> matching "<span className="text-slate-400">{search}</span>"</>}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  )
}
