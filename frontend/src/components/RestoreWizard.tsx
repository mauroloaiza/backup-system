import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { createRestoreRequest, type JobRun } from '@/lib/api'

interface Props {
  run: JobRun
  onClose: () => void
  onCreated?: () => void
}

export function RestoreWizard({ run, onClose, onCreated }: Props) {
  const qc = useQueryClient()
  const [targetPath, setTargetPath] = useState('')
  const [filterPattern, setFilterPattern] = useState('')
  const [dryRun, setDryRun] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const mutation = useMutation({
    mutationFn: () => {
      if (!run.node_id) throw new Error('Este job no tiene nodo asociado')
      return createRestoreRequest(run.node_id, {
        source_job_id: run.job_id,
        target_path: targetPath.trim(),
        filter_pattern: filterPattern.trim() || undefined,
        dry_run: dryRun,
      })
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['restore-requests'] })
      onCreated?.()
      onClose()
    },
    onError: (err: any) => {
      setError(err?.response?.data?.detail ?? err?.message ?? 'Error desconocido')
    },
  })

  const canSubmit = targetPath.trim().length > 0 && !mutation.isPending

  return (
    <>
      <div className="fixed inset-0 bg-black/40 backdrop-blur-[2px] z-[60]" onClick={onClose} />
      <div className="fixed inset-0 flex items-center justify-center z-[70] p-4">
        <div className="bg-white rounded-xl shadow-2xl w-full max-w-md overflow-hidden" onClick={e => e.stopPropagation()}>
          <div className="flex items-center justify-between px-5 py-4 border-b border-gray-100">
            <div>
              <h2 className="text-sm font-bold text-gray-800">Restaurar job</h2>
              <p className="text-xs font-mono text-gray-400 mt-0.5">{run.job_id.slice(0, 12)}…</p>
            </div>
            <button onClick={onClose} className="text-gray-400 hover:text-gray-600 p-1 rounded-lg hover:bg-gray-100">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <div className="px-5 py-5 space-y-4">
            <div>
              <label className="block text-xs font-semibold text-gray-600 mb-1.5">
                Ruta destino <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                placeholder="C:\Restaurado"
                value={targetPath}
                onChange={e => setTargetPath(e.target.value)}
                className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm font-mono
                  focus:outline-none focus:border-primary bg-white"
                autoFocus
              />
              <p className="text-[10px] text-gray-400 mt-1">
                Ruta absoluta en el nodo <span className="font-semibold">{run.node_id}</span>. Se creará si no existe.
              </p>
            </div>

            <div>
              <label className="block text-xs font-semibold text-gray-600 mb-1.5">
                Filtro (opcional)
              </label>
              <input
                type="text"
                placeholder="Documents/**, *.pdf"
                value={filterPattern}
                onChange={e => setFilterPattern(e.target.value)}
                className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm font-mono
                  focus:outline-none focus:border-primary bg-white"
              />
              <p className="text-[10px] text-gray-400 mt-1">
                Glob relativo al origen. Si se omite, se restaura todo.
              </p>
            </div>

            <label className="flex items-start gap-2 cursor-pointer select-none">
              <input
                type="checkbox"
                checked={dryRun}
                onChange={e => setDryRun(e.target.checked)}
                className="mt-0.5 w-4 h-4 rounded border-gray-300 text-primary focus:ring-primary"
              />
              <div>
                <p className="text-xs font-semibold text-gray-700">Dry-run</p>
                <p className="text-[10px] text-gray-400">
                  Lista archivos que se restaurarían, sin escribir nada en disco.
                </p>
              </div>
            </label>

            {error && (
              <div className="bg-red-50 border border-red-100 rounded-lg px-3 py-2">
                <p className="text-xs text-red-600">{error}</p>
              </div>
            )}

            <div className="bg-amber-50 border border-amber-100 rounded-lg px-3 py-2">
              <p className="text-[11px] text-amber-700">
                La restauración se encolará en el servidor. El agente la recogerá en su próximo polling
                (hasta 60s) y reportará progreso al servidor.
              </p>
            </div>
          </div>

          <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-gray-100 bg-gray-50">
            <button
              onClick={onClose}
              disabled={mutation.isPending}
              className="px-3 py-1.5 text-xs font-semibold text-gray-600 hover:text-gray-800 disabled:opacity-50"
            >
              Cancelar
            </button>
            <button
              onClick={() => { setError(null); mutation.mutate() }}
              disabled={!canSubmit}
              className="px-4 py-1.5 text-xs font-semibold text-white bg-primary rounded-lg
                hover:bg-[#3451d1] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {mutation.isPending ? 'Encolando…' : 'Restaurar'}
            </button>
          </div>
        </div>
      </div>
    </>
  )
}
