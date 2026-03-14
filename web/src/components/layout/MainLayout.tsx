import { Outlet } from 'react-router'
import Sidebar from './Sidebar'
import Header from './Header'

export default function MainLayout() {
  return (
    <div className="flex min-h-screen bg-slate-950">
      <Sidebar />
      <div className="flex-1 flex flex-col">
        <Header />
        <main className="flex-1 p-6 overflow-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
