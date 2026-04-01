export function Destinations() {
  const destinations = [
    { name: 'S3 · backupsmc', type: 'S3', icon: '☁️', color: 'bg-blue-100 text-blue-500', used: 187, total: 500, unit: 'GB' },
    { name: 'SFTP · srv-backup', type: 'SFTP', icon: '🖧', color: 'bg-green-100 text-green-600', used: 94, total: 200, unit: 'GB' },
    { name: 'Local · D:\\Backups', type: 'Local', icon: '💾', color: 'bg-amber-100 text-amber-600', used: 42, total: 120, unit: 'GB' },
  ]

  return (
    <div className="p-6 space-y-5">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-xl font-bold text-gray-800">Destinos</h1>
          <p className="text-sm text-gray-400 mt-0.5">Almacenamiento configurado</p>
        </div>
        <button className="flex items-center gap-1.5 text-sm bg-primary text-white px-4 py-2 rounded-lg hover:bg-primary-dark transition-colors">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
          </svg>
          Agregar destino
        </button>
      </div>

      <div className="grid grid-cols-3 gap-4">
        {destinations.map(d => {
          const pct = Math.round((d.used / d.total) * 100)
          const barColor = pct > 80 ? 'bg-red-500' : pct > 60 ? 'bg-amber-500' : 'bg-primary'
          return (
            <div key={d.name} className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className={`w-10 h-10 rounded-xl flex items-center justify-center text-lg ${d.color.split(' ')[0]}`}>
                    <svg className={`w-5 h-5 ${d.color.split(' ')[1]}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      {d.type === 'S3' && <path strokeLinecap="round" strokeLinejoin="round" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z" />}
                      {d.type === 'SFTP' && <path strokeLinecap="round" strokeLinejoin="round" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />}
                      {d.type === 'Local' && <path strokeLinecap="round" strokeLinejoin="round" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />}
                    </svg>
                  </div>
                  <div>
                    <p className="font-semibold text-gray-800 text-sm">{d.name}</p>
                    <span className="text-[10px] bg-gray-100 text-gray-500 px-1.5 py-0.5 rounded font-medium">{d.type}</span>
                  </div>
                </div>
                <span className="w-2 h-2 rounded-full bg-emerald-400 mt-1" />
              </div>

              <div className="space-y-1.5">
                <div className="flex justify-between text-xs text-gray-500">
                  <span>Usado</span>
                  <span className="font-medium text-gray-700">{d.used} {d.unit} / {d.total} {d.unit}</span>
                </div>
                <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
                  <div className={`h-full ${barColor} rounded-full bar-fill`} style={{ width: `${pct}%` }} />
                </div>
                <p className="text-right text-[10px] text-gray-400">{pct}% usado</p>
              </div>
            </div>
          )
        })}
      </div>

      <div className="bg-white border border-gray-200 rounded-xl p-8 text-center shadow-sm">
        <p className="text-sm text-gray-500">La configuración de destinos desde la UI estará disponible en v0.2.0.</p>
        <p className="text-xs text-gray-400 mt-1">Por ahora, configura los destinos en el archivo <code className="bg-gray-100 px-1.5 py-0.5 rounded text-xs">agent.yaml</code></p>
      </div>
    </div>
  )
}
