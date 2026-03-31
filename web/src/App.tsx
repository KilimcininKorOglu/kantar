import { Routes, Route, Navigate } from 'react-router'
import { useAuth } from './hooks/useAuth'
import MainLayout from './components/layout/MainLayout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import PackageList from './pages/PackageList'
import PackageDetail from './pages/PackageDetail'
import Registries from './pages/Registries'
import Users from './pages/Users'
import AuditLog from './pages/AuditLog'
import Policies from './pages/Policies'
import Settings from './pages/Settings'
import Profile from './pages/Profile'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth()
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <MainLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="packages" element={<PackageList />} />
        <Route path="packages/:registry/:name" element={<PackageDetail />} />
        <Route path="registries" element={<Registries />} />
        <Route path="users" element={<Users />} />
        <Route path="audit" element={<AuditLog />} />
        <Route path="policies" element={<Policies />} />
        <Route path="settings" element={<Settings />} />
        <Route path="profile" element={<Profile />} />
      </Route>
    </Routes>
  )
}
