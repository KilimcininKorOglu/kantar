import { useParams } from 'react-router'

export default function PackageDetail() {
  const { registry, name } = useParams()

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="text-xs text-slate-500 mb-1">{registry}</div>
          <h2 className="text-xl font-semibold text-white">{name}</h2>
          <p className="text-sm text-slate-400 mt-1">Package details</p>
        </div>
        <div className="flex gap-2">
          <button className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white text-sm rounded transition-colors cursor-pointer">
            Approve
          </button>
          <button className="px-3 py-1.5 bg-red-600 hover:bg-red-500 text-white text-sm rounded transition-colors cursor-pointer">
            Block
          </button>
          <button className="px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-white text-sm rounded transition-colors cursor-pointer">
            Sync
          </button>
        </div>
      </div>

      {/* 3-Column Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* Versions */}
        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <h3 className="text-sm font-medium text-slate-400 mb-3">Versions</h3>
          <p className="text-sm text-slate-600">No versions synced</p>
        </div>

        {/* Info */}
        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <h3 className="text-sm font-medium text-slate-400 mb-3">Info</h3>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between"><span className="text-slate-500">License</span><span className="text-white">—</span></div>
            <div className="flex justify-between"><span className="text-slate-500">Size</span><span className="text-white">—</span></div>
            <div className="flex justify-between"><span className="text-slate-500">Dependencies</span><span className="text-white">—</span></div>
            <div className="flex justify-between"><span className="text-slate-500">Vulnerabilities</span><span className="text-white">0</span></div>
          </div>
        </div>

        {/* Stats */}
        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <h3 className="text-sm font-medium text-slate-400 mb-3">Downloads</h3>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between"><span className="text-slate-500">Today</span><span className="text-white">0</span></div>
            <div className="flex justify-between"><span className="text-slate-500">This week</span><span className="text-white">0</span></div>
            <div className="flex justify-between"><span className="text-slate-500">Total</span><span className="text-white">0</span></div>
          </div>
        </div>
      </div>

      {/* Dependencies */}
      <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
        <h3 className="text-sm font-medium text-slate-400 mb-3">Dependencies</h3>
        <p className="text-sm text-slate-600">No dependency information available</p>
      </div>

      {/* Vulnerability Report */}
      <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium text-slate-400">Vulnerability Report</h3>
          <button className="text-xs text-blue-400 hover:text-blue-300 cursor-pointer">Scan</button>
        </div>
        <p className="text-sm text-emerald-400 mt-2">No known vulnerabilities</p>
      </div>
    </div>
  )
}
