import { useState, useEffect, type FormEvent } from 'react'
import { api } from '../api/client'
import type { User } from '../api/types'
import { Plus, Pencil, Trash2, X } from 'lucide-react'

export default function Users() {
  const [users, setUsers] = useState<User[]>([])
  const [showCreate, setShowCreate] = useState(false)
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [error, setError] = useState('')

  const loadUsers = () => {
    api.get<User[]>('/users').then(setUsers).catch(() => setError('Failed to load users'))
  }

  useEffect(() => { loadUsers() }, [])

  const handleDelete = async (id: number) => {
    if (!confirm('Are you sure you want to delete this user?')) return
    try {
      await api.delete(`/users/${id}`)
      loadUsers()
    } catch { setError('Failed to delete user') }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div />
        <button
          onClick={() => { setShowCreate(true); setError('') }}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-accent hover:bg-accent-hover text-white text-xs font-medium rounded cursor-pointer"
        >
          <Plus className="w-3.5 h-3.5" /> Create User
        </button>
      </div>

      {error && <div className="bg-danger/10 border border-danger/20 text-danger text-xs rounded px-3 py-2">{error}</div>}

      {showCreate && <CreateUserForm onClose={() => setShowCreate(false)} onCreated={() => { setShowCreate(false); loadUsers() }} />}
      {editingUser && <EditUserForm user={editingUser} onClose={() => setEditingUser(null)} onUpdated={() => { setEditingUser(null); loadUsers() }} />}

      <div className="bg-surface border border-border rounded overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-text-dim">
              <th className="text-left px-4 py-2.5 text-xs font-medium">Username</th>
              <th className="text-left px-4 py-2.5 text-xs font-medium">Email</th>
              <th className="text-left px-4 py-2.5 text-xs font-medium">Role</th>
              <th className="text-left px-4 py-2.5 text-xs font-medium">Status</th>
              <th className="text-right px-4 py-2.5 text-xs font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {users.length === 0 ? (
              <tr><td colSpan={5} className="px-4 py-10 text-center text-text-dim text-xs">No users found</td></tr>
            ) : users.map((u) => (
              <tr key={u.id} className="border-b border-border last:border-0">
                <td className="px-4 py-2.5 text-text font-medium text-xs">{u.username}</td>
                <td className="px-4 py-2.5 text-text-muted text-xs font-mono">{u.email || '—'}</td>
                <td className="px-4 py-2.5"><RoleBadge role={u.role} /></td>
                <td className="px-4 py-2.5">
                  <span className={`text-[11px] font-medium ${u.active ? 'text-success' : 'text-danger'}`}>
                    {u.active ? 'active' : 'disabled'}
                  </span>
                </td>
                <td className="px-4 py-2.5 text-right">
                  <div className="flex items-center justify-end gap-2">
                    <button onClick={() => setEditingUser(u)} className="text-text-dim hover:text-accent cursor-pointer"><Pencil className="w-3.5 h-3.5" /></button>
                    <button onClick={() => handleDelete(u.id)} className="text-text-dim hover:text-danger cursor-pointer"><Trash2 className="w-3.5 h-3.5" /></button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function RoleBadge({ role }: { role: string }) {
  const cls = role === 'super_admin' ? 'text-accent bg-accent/10' :
    role === 'registry_admin' ? 'text-blue-300 bg-blue-400/10' :
    'text-text-muted bg-surface-2'
  return <span className={`text-[11px] font-medium px-1.5 py-0.5 rounded ${cls}`}>{role}</span>
}

function CreateUserForm({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setLoading(true); setError('')
    try {
      await api.post('/auth/register', { username, email, password })
      onCreated()
    } catch { setError('Failed to create user') }
    setLoading(false)
  }

  return (
    <div className="bg-surface border border-border rounded p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-xs font-semibold text-text">Create User</h3>
        <button onClick={onClose} className="text-text-dim hover:text-text cursor-pointer"><X className="w-4 h-4" /></button>
      </div>
      {error && <div className="bg-danger/10 border border-danger/20 text-danger text-xs rounded px-3 py-2 mb-3">{error}</div>}
      <form onSubmit={handleSubmit} className="grid grid-cols-1 md:grid-cols-4 gap-3">
        <input placeholder="Username" value={username} onChange={(e) => setUsername(e.target.value)} className="bg-surface-2 border border-border rounded px-3 py-1.5 text-xs text-text focus:outline-none focus:border-accent" required />
        <input placeholder="Email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} className="bg-surface-2 border border-border rounded px-3 py-1.5 text-xs text-text focus:outline-none focus:border-accent" />
        <input placeholder="Password (min 8)" type="password" value={password} onChange={(e) => setPassword(e.target.value)} className="bg-surface-2 border border-border rounded px-3 py-1.5 text-xs text-text focus:outline-none focus:border-accent" required minLength={8} />
        <button type="submit" disabled={loading} className="bg-accent hover:bg-accent-hover disabled:opacity-50 text-white text-xs font-medium rounded py-1.5 cursor-pointer">{loading ? 'Creating...' : 'Create'}</button>
      </form>
    </div>
  )
}

function EditUserForm({ user, onClose, onUpdated }: { user: User; onClose: () => void; onUpdated: () => void }) {
  const [role, setRole] = useState(user.role)
  const [active, setActive] = useState(user.active)
  const [email, setEmail] = useState(user.email || '')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setLoading(true); setError('')
    try {
      await api.put(`/users/${user.id}`, { email, role, active })
      onUpdated()
    } catch { setError('Failed to update user') }
    setLoading(false)
  }

  return (
    <div className="bg-surface border border-border rounded p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-xs font-semibold text-text">Edit: {user.username}</h3>
        <button onClick={onClose} className="text-text-dim hover:text-text cursor-pointer"><X className="w-4 h-4" /></button>
      </div>
      {error && <div className="bg-danger/10 border border-danger/20 text-danger text-xs rounded px-3 py-2 mb-3">{error}</div>}
      <form onSubmit={handleSubmit} className="grid grid-cols-1 md:grid-cols-4 gap-3 items-end">
        <input placeholder="Email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} className="bg-surface-2 border border-border rounded px-3 py-1.5 text-xs text-text focus:outline-none focus:border-accent" />
        <select value={role} onChange={(e) => setRole(e.target.value)} className="bg-surface-2 border border-border rounded px-3 py-1.5 text-xs text-text">
          {['viewer', 'consumer', 'publisher', 'registry_admin', 'super_admin'].map(r => <option key={r} value={r}>{r}</option>)}
        </select>
        <label className="flex items-center gap-2 text-xs text-text-muted">
          <input type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} className="rounded" /> Active
        </label>
        <button type="submit" disabled={loading} className="bg-accent hover:bg-accent-hover disabled:opacity-50 text-white text-xs font-medium rounded py-1.5 cursor-pointer">{loading ? 'Saving...' : 'Save'}</button>
      </form>
    </div>
  )
}
