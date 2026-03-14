export default function AuditLog() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-white">Audit Log</h2>
        <div className="flex gap-2">
          <button className="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs rounded transition-colors cursor-pointer">
            Export CSV
          </button>
          <button className="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs rounded transition-colors cursor-pointer">
            Export JSON
          </button>
          <button className="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs rounded transition-colors cursor-pointer">
            Verify Chain
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex gap-3">
        <select className="bg-slate-800 border border-slate-700 rounded px-3 py-1.5 text-sm text-slate-300">
          <option value="">All Events</option>
          <option value="package.download">package.download</option>
          <option value="package.approve">package.approve</option>
          <option value="package.block">package.block</option>
          <option value="user.login">user.login</option>
          <option value="registry.sync">registry.sync</option>
          <option value="policy.violation">policy.violation</option>
        </select>
        <input
          type="text"
          placeholder="Filter by actor..."
          className="bg-slate-800 border border-slate-700 rounded px-3 py-1.5 text-sm text-white focus:outline-none focus:border-blue-500"
        />
        <input
          type="date"
          className="bg-slate-800 border border-slate-700 rounded px-3 py-1.5 text-sm text-slate-300"
        />
      </div>

      {/* Audit Table */}
      <div className="bg-slate-900 border border-slate-800 rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-800 text-slate-500">
              <th className="text-left px-4 py-3 font-medium">Timestamp</th>
              <th className="text-left px-4 py-3 font-medium">Event</th>
              <th className="text-left px-4 py-3 font-medium">Actor</th>
              <th className="text-left px-4 py-3 font-medium">Resource</th>
              <th className="text-left px-4 py-3 font-medium">Result</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td colSpan={5} className="px-4 py-12 text-center text-slate-600">
                No audit log entries
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  )
}
