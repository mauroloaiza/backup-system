export function History() {
  return (
    <div className="p-6">
      <div className="mb-5">
        <h1 className="text-xl font-bold text-gray-800">Historial</h1>
        <p className="text-sm text-gray-400 mt-0.5">Tendencias y estadísticas históricas</p>
      </div>
      <div className="bg-white border border-gray-200 rounded-xl p-16 text-center shadow-sm">
        <svg className="w-12 h-12 mx-auto mb-3 text-gray-200" fill="none" viewBox="0 0 24 24"
          stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round"
            d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
        </svg>
        <p className="font-semibold text-gray-600">Gráficos históricos — próximamente</p>
        <p className="text-sm text-gray-400 mt-1">Charts de tendencia por día, semana y mes. Comparativa de volumen y duración.</p>
      </div>
    </div>
  )
}
