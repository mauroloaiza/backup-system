import { useQuery } from '@tanstack/react-query'
import { fetchRestoreRequests, cancelRestoreRequest, type RestoreRequest } from '@/lib/api'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { fmtBytes, fmtDate, fmtShortId } from '@/lib/utils'

const STATUS_COLORS: Record<RestoreRequest['status'], string> = {
  queued:    'bg-slate-100 text-slate-600',
  running:   'bg-blue-100 text-blue-700',
  completed: 'bg-green-100 text-green-700',
  failed:    'bg-red-100 text-red-700',
  cancelled: 'bg-gray-100 text-gray-500',
}

const STATUS_LABEL: Record<RestoreRequest['status'], string> = {
  queued: 'En cola',
  running: 'En curso',
  completed: 'Completada',
  failed: 'Falló',
  cancelled: 'Cancelada',
}

export function Restore() {
  const { data = [], isLoading } = useQuery({
    queryKey: ['restore-requests'],
    queryFn: () => fetchRestoreRequests({ limit: 100 }),
    refetchInterval: 10_000,
  })

  const qc = useQueryClient()
  const cancelMut = useMutation({
    mutationFn: (id: number) => cancelRestoreRequest(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['restore-requests'] }),
  })

  return (
    <div className="p-6 space-y-5">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-xl font-bold text-gray-800">Restauraciones</h1>
          <p className="text-sm text-gray-400 mt-0.5">{data.length} solicitudes</p>
        </div>
      </div>

      <div className="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-[11px] text-gray-400 uppercase tracking-wide">
            <tr>
              <th className="text-left px-5 py-3 font-medium">ID</th>
              <th className="text-left px-5 py-3 font-medium">Nodo</th>
              <th className="text-left px-5 py-3 font-medium">Job origen</th>
              <th className="text-left px-5 py-3 font-medium">Destino</th>
              <th className="text-left px-5 py-3 font-medium">Estado</th>
              <th className="text-right px-5 py-3 font-medium">Archivos</th>
              <th className="text-right px-5 py-3 font-medium">Bytes</th>
              <th className="text-left px-5 py-3 font-medium">Solicitado</th>
              <th className="text-right px-5 py-3 font-medium">Acción</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {isLoading ? (
              <tr><td colSpan={9} className="px-5 py-12 text-center text-gray-400 text-sm">Cargando…</td></tr>
            ) : data.length === 0 ? (
              <tr><td colSpan={9} className="px-5 py-12 text-center text-gray-400 text-sm">
                Sin solicitudes de restauración. Inicia una desde el detalle de un job completado.
              </td></tr>
            ) : data.map(rr => (
              <tr key={rr.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-5 py-3 text-xs font-mono text-gray-600">#{rr.id}</td>
                <td className="px-5 py-3 text-xs text-gray-700">{rr.node_id}</td>
                <td className="px-5 py-3 text-xs font-mono text-gray-500">{fmtShortId(rr.source_job_id)}</td>
                <td className="px-5 py-3 text-xs font-mono text-gray-700 truncate max-w-[240px]" title={rr.target_path}>
                  {rr.target_path}
                  {rr.dry_run && <span className="ml-1 text-[10px] text-amber-600">[dry-run]</span>}
                </td>
                <td className="px-5 py-3">
                  <span className={`inline-flex items-center px-2 py-0.5 rounded text-[10px] font-semibold ${STATUS_COLORS[rr.status]}`}>
                    {STATUS_LABEL[rr.status]}
                  </span>
                </td>
                <td className="px-5 py-3 text-right text-xs text-gray-600">{rr.files_restored.toLocaleString()}</td>
                <td className="px-5 py-3 text-right text-xs text-gray-600">{fmtBytes(rr.bytes_restored)}</td>
                <td className="px-5 py-3 text-xs text-gray-400">
                  {fmtDate(rr.created_at)}
                  {rr.requested_by && <span className="block text-[10px]">por {rr.requested_by}</span>}
                </td>
                <td className="px-5 py-3 text-right">
                  {rr.status === 'queued' ? (
                    <button
                      onClick={() => cancelMut.mutate(rr.id)}
                      disabled={cancelMut.isPending}
                      className="text-[11px] text-red-600 hover:text-red-800 font-semibold"
                    >
                      Cancelar
                    </button>
                  ) : rr.message ? (
                    <span className="text-[10px] text-gray-400" title={rr.message}>
                      {rr.message.slice(0, 30)}{rr.message.length > 30 ? '…' : ''}
                    </span>
                  ) : '—'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
