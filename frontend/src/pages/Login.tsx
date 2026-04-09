import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'

export function Login() {
  const login = useAuth(s => s.login)
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await login(username, password)
      navigate('/')
    } catch {
      setError('Usuario o contraseña incorrectos')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-[#0f1623]">
      <div className="w-full max-w-sm">
        {/* Logo */}
        <div className="flex items-center gap-3 justify-center mb-8">
          <div className="w-10 h-10 bg-primary rounded-xl flex items-center justify-center">
            <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24"
              stroke="currentColor" strokeWidth={2.2}>
              <path strokeLinecap="round" strokeLinejoin="round"
                d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10" />
            </svg>
          </div>
          <div>
            <div className="font-bold text-white text-lg leading-tight">BackupSMC</div>
            <div className="text-[11px] text-blue-400/70 leading-tight">Consola de administración</div>
          </div>
        </div>

        {/* Card */}
        <div className="bg-[#141b2d] rounded-2xl border border-white/5 p-8">
          <h2 className="text-white font-semibold text-lg mb-1">Iniciar sesión</h2>
          <p className="text-[#8b9ab4] text-sm mb-6">Ingresa tus credenciales para continuar</p>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-xs font-medium text-[#8b9ab4] mb-1.5">Usuario</label>
              <input
                type="text"
                value={username}
                onChange={e => setUsername(e.target.value)}
                placeholder="admin"
                required
                autoFocus
                className="w-full bg-[#0f1623] border border-white/10 rounded-lg px-3.5 py-2.5
                           text-sm text-white placeholder-white/20 outline-none
                           focus:border-primary focus:ring-1 focus:ring-primary transition-colors"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-[#8b9ab4] mb-1.5">Contraseña</label>
              <input
                type="password"
                value={password}
                onChange={e => setPassword(e.target.value)}
                placeholder="••••••••"
                required
                className="w-full bg-[#0f1623] border border-white/10 rounded-lg px-3.5 py-2.5
                           text-sm text-white placeholder-white/20 outline-none
                           focus:border-primary focus:ring-1 focus:ring-primary transition-colors"
              />
            </div>

            {error && (
              <div className="flex items-center gap-2 text-red-400 text-sm bg-red-400/10
                              border border-red-400/20 rounded-lg px-3 py-2.5">
                <svg className="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24"
                  stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round"
                    d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full bg-primary hover:bg-[#3451d1] disabled:opacity-60
                         text-white font-medium text-sm rounded-lg py-2.5 transition-colors"
            >
              {loading ? 'Ingresando...' : 'Ingresar'}
            </button>
          </form>
        </div>

        <p className="text-center text-[#8b9ab4]/40 text-xs mt-6">
          BackupSMC · SMC Soluciones
        </p>
      </div>
    </div>
  )
}
