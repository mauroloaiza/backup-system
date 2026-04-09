import { Routes, Route, Navigate } from 'react-router-dom'
import Sidebar from './components/Sidebar'
import Dashboard from './pages/Dashboard'
import Sources from './pages/Sources'
import Destinations from './pages/Destinations'
import ServerPage from './pages/Server'
import Logs from './pages/Logs'

export default function App() {
  return (
    <div className="flex h-screen w-screen overflow-hidden bg-navy-900">
      <Sidebar />
      <main className="flex-1 overflow-y-auto">
        <Routes>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/sources" element={<Sources />} />
          <Route path="/destinations" element={<Destinations />} />
          <Route path="/server" element={<ServerPage />} />
          <Route path="/logs" element={<Logs />} />
        </Routes>
      </main>
    </div>
  )
}
