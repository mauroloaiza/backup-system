import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { api } from '@/lib/api'

interface AuthState {
  token: string | null
  username: string | null
  login: (username: string, password: string) => Promise<void>
  logout: () => void
}

export const useAuth = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      username: null,

      login: async (username, password) => {
        const res = await api.post<{ access_token: string }>('/auth/login', { username, password })
        const token = res.data.access_token
        api.defaults.headers.common['Authorization'] = `Bearer ${token}`
        set({ token, username })
      },

      logout: () => {
        delete api.defaults.headers.common['Authorization']
        set({ token: null, username: null })
      },
    }),
    {
      name: 'backupsmc-auth',
      onRehydrateStorage: () => (state) => {
        // Restore axios header after page reload
        if (state?.token) {
          api.defaults.headers.common['Authorization'] = `Bearer ${state.token}`
        }
      },
    }
  )
)
