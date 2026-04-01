export function Settings() {
  return (
    <div className="p-6 space-y-5">
      <div>
        <h1 className="text-xl font-bold text-gray-800">Configuración</h1>
        <p className="text-sm text-gray-400 mt-0.5">Ajustes del sistema</p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        {/* Server */}
        <div className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
          <h3 className="text-sm font-semibold text-gray-700 mb-4 flex items-center gap-2">
            <svg className="w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
            </svg>
            Servidor
          </h3>
          <div className="space-y-3">
            {[
              { label: 'URL del servidor', value: 'http://localhost:8000', type: 'text' },
              { label: 'Versión API', value: 'v0.1.0', type: 'text' },
            ].map(f => (
              <div key={f.label}>
                <label className="block text-xs font-medium text-gray-500 mb-1">{f.label}</label>
                <input
                  defaultValue={f.value}
                  className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm text-gray-700 bg-gray-50 focus:outline-none focus:border-primary"
                />
              </div>
            ))}
          </div>
        </div>

        {/* Notifications placeholder */}
        <div className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
          <h3 className="text-sm font-semibold text-gray-700 mb-4 flex items-center gap-2">
            <svg className="w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
            </svg>
            Alertas
          </h3>
          <div className="space-y-3">
            {['Notificar en fallo', 'Notificar en completado', 'Resumen diario'].map(label => (
              <div key={label} className="flex items-center justify-between">
                <span className="text-sm text-gray-600">{label}</span>
                <button className="relative w-10 h-5 bg-gray-200 rounded-full transition-colors focus:outline-none">
                  <span className="absolute left-0.5 top-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform" />
                </button>
              </div>
            ))}
          </div>
          <p className="text-xs text-gray-400 mt-4">Email y WhatsApp — disponible en v0.2.0</p>
        </div>
      </div>
    </div>
  )
}
