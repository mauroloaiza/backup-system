import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Layout } from '@/components/Layout'
import { Dashboard } from '@/pages/Dashboard'
import { Jobs } from '@/pages/Jobs'
import { Nodes } from '@/pages/Nodes'
import { History } from '@/pages/History'
import { Destinations } from '@/pages/Destinations'
import { Settings } from '@/pages/Settings'
import { Login } from '@/pages/Login'
import { useAuth } from '@/store/auth'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 5_000 },
  },
})

function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = useAuth(s => s.token)
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route
            element={
              <RequireAuth>
                <Layout />
              </RequireAuth>
            }
          >
            <Route index element={<Dashboard />} />
            <Route path="jobs" element={<Jobs />} />
            <Route path="nodes" element={<Nodes />} />
            <Route path="history" element={<History />} />
            <Route path="destinations" element={<Destinations />} />
            <Route path="settings" element={<Settings />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
