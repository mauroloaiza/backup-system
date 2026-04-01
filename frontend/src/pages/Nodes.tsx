import { useQuery } from '@tanstack/react-query'
import { fetchNodes } from '@/lib/api'
import { Badge } from '@/components/Badge'
import { timeAgo } from '@/lib/utils'

export function Nodes() {
  const { data = [], isLoading } = useQuery({
    queryKey: ['nodes'],
    queryFn: fetchNodes,
    refetchInterval: 15_000,
  })

  return (
    <div className="p-6 space-y-5">
      <div>
        <h1 className="text-xl font-bold text-gray-800">Nodos</h1>
        <p className="text-sm text-gray-400 mt-0.5">{data.length} nodos registrados</p>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-2 gap-4">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="bg-white border border-gray-200 rounded-xl p-5 h-36 animate-pulse">
              <div className="h-4 bg-gray-100 rounded w-1/3 mb-3" />
              <div className="h-3 bg-gray-100 rounded w-1/2" />
            </div>
          ))}
        </div>
      ) : data.length === 0 ? (
        <div className="bg-white border border-gray-200 rounded-xl p-12 text-center">
          <div className="w-12 h-12 bg-gray-50 rounded-xl flex items-center justify-center mx-auto mb-3">
            <svg className="w-6 h-6 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
            </svg>
          </div>
          <p className="text-sm text-gray-500 font-medium">Sin nodos registrados</p>
          <p className="text-xs text-gray-400 mt-1">El agente se registra automáticamente al correr el primer backup.</p>
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-4">
          {data.map(node => (
            <div key={node.id} className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm hover:shadow-md transition-shadow">
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 bg-orange-100 rounded-xl flex items-center justify-center flex-shrink-0">
                    <svg className="w-5 h-5 text-orange-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
                    </svg>
                  </div>
                  <div>
                    <p className="font-semibold text-gray-800 text-sm">{node.hostname}</p>
                    <p className="text-xs text-gray-400 font-mono mt-0.5">{node.id}</p>
                  </div>
                </div>
                <Badge status={node.status} />
              </div>

              <div className="grid grid-cols-3 gap-3 mt-4">
                <div className="text-center p-2 bg-gray-50 rounded-lg">
                  <p className="text-[10px] text-gray-400 uppercase tracking-wide">OS</p>
                  <p className="text-xs font-semibold text-gray-700 mt-0.5 capitalize">{node.os}</p>
                </div>
                <div className="text-center p-2 bg-gray-50 rounded-lg">
                  <p className="text-[10px] text-gray-400 uppercase tracking-wide">Versión</p>
                  <p className="text-xs font-semibold text-gray-700 mt-0.5">v{node.agent_version}</p>
                </div>
                <div className="text-center p-2 bg-gray-50 rounded-lg">
                  <p className="text-[10px] text-gray-400 uppercase tracking-wide">Visto</p>
                  <p className="text-xs font-semibold text-gray-700 mt-0.5">{timeAgo(node.last_seen)}</p>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
