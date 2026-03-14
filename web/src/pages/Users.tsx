export default function Users() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-white">Users</h2>
        <button className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 text-white text-sm rounded transition-colors cursor-pointer">
          Create User
        </button>
      </div>

      <div className="bg-slate-900 border border-slate-800 rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-800 text-slate-500">
              <th className="text-left px-4 py-3 font-medium">Username</th>
              <th className="text-left px-4 py-3 font-medium">Email</th>
              <th className="text-left px-4 py-3 font-medium">Role</th>
              <th className="text-left px-4 py-3 font-medium">Status</th>
              <th className="text-right px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr className="border-b border-slate-800/50">
              <td className="px-4 py-3 text-white font-medium">admin</td>
              <td className="px-4 py-3 text-slate-400">—</td>
              <td className="px-4 py-3">
                <span className="px-2 py-0.5 bg-purple-900/40 text-purple-300 text-xs rounded">super_admin</span>
              </td>
              <td className="px-4 py-3">
                <span className="px-2 py-0.5 bg-emerald-900/40 text-emerald-300 text-xs rounded">active</span>
              </td>
              <td className="px-4 py-3 text-right">
                <button className="text-blue-400 hover:text-blue-300 text-xs cursor-pointer">Edit</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  )
}
