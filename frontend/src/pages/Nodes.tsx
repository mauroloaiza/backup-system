import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchNodes, updateNodeSourcePaths, type Node } from '@/lib/api'
import { Badge } from '@/components/Badge'
import { timeAgo } from '@/lib/utils'

export function Nodes() {
  const qc = useQueryClient()
  const { data = [], isLoading } = useQuery({
    queryKey: ['nodes'],
    queryFn: fetchNodes,
    refetchInterval: 15_000,
  })

  return (
    <div className="p-6 space-y-5">
      <div>
        <h1 className="text-xl font-bold text-gray-800">Nodos</h1>
        <p className="text-sm text-gray-400 mt-0.5">
          {data.length} nodo{data.length !== 1 ? 's' : ''} registrado{data.length !== 1 ? 's' : ''}
        </p>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {[...Array(2)].map((_, i) => (
            <div key={i} className="bg-white border border-gray-200 rounded-xl p-5 h-48 animate-pulse">
              <div className="h-4 bg-gray-100 rounded w-1/3 mb-3" />
              <div className="h-3 bg-gray-100 rounded w-1/2" />
            </div>
          ))}
        </div>
      ) : data.length === 0 ? (
        <EmptyState />
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {data.map(node => (
            <NodeCard
              key={node.id}
              node={node}
              onUpdate={() => qc.invalidateQueries({ queryKey: ['nodes'] })}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function NodeCard({ node, onUpdate }: { node: Node; onUpdate: () => void }) {
  const [showAdd, setShowAdd] = useState(false)
  const [newPath, setNewPath] = useState('')
  const [saving, setSaving] = useState(false)

  const addPath = async () => {
    const p = newPath.trim()
    if (!p || node.source_paths.includes(p)) { setNewPath(''); setShowAdd(false); return }
    setSaving(true)
    try {
      await updateNodeSourcePaths(node.id, [...node.source_paths, p])
      setNewPath('')
      setShowAdd(false)
      onUpdate()
    } finally {
      setSaving(false)
    }
  }

  const removePath = async (path: string) => {
    await updateNodeSourcePaths(node.id, node.source_paths.filter(p => p !== path))
    onUpdate()
  }

  return (
    <div className="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
      {/* Header */}
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-orange-100 rounded-xl flex items-center justify-center flex-shrink-0">
            <svg className="w-5 h-5 text-orange-500" fill="none" viewBox="0 0 24 24"
              stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round"
                d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
            </svg>
          </div>
          <div>
            <p className="font-semibold text-gray-800 text-sm">{node.hostname}</p>
            <p className="text-xs text-gray-400 font-mono mt-0.5">{node.id}</p>
          </div>
        </div>
        <Badge status={node.status} />
      </div>

      {/* Info row */}
      <div className="grid grid-cols-3 gap-2 mb-4">
        <InfoCell label="OS" value={node.os} />
        <InfoCell label="Versión" value={`v${node.agent_version}`} />
        <InfoCell label="Visto" value={timeAgo(node.last_seen)} />
      </div>

      {/* Source paths */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <p className="text-[10px] font-semibold text-gray-400 uppercase tracking-widest">
            Carpetas respaldadas
          </p>
          <button
            onClick={() => setShowAdd(v => !v)}
            className="flex items-center gap-1 text-xs text-primary hover:text-[#3451d1] font-medium transition-colors"
          >
            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24"
              stroke="currentColor" strokeWidth={2.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
            </svg>
            Agregar
          </button>
        </div>

        {node.source_paths.length === 0 && !showAdd ? (
          <p className="text-xs text-gray-400 italic py-1">
            Sin carpetas — el agente las sincroniza al conectarse.
          </p>
        ) : (
          <ul className="space-y-1">
            {node.source_paths.map(path => (
              <li key={path}
                className="group flex items-center gap-2 text-xs bg-gray-50 rounded-lg px-3 py-2">
                <svg className="w-3.5 h-3.5 text-gray-400 shrink-0" fill="none"
                  viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round"
                    d="M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V7z" />
                </svg>
                <span className="flex-1 font-mono text-gray-700 truncate">{path}</span>
                <button
                  onClick={() => removePath(path)}
                  title="Quitar"
                  className="opacity-0 group-hover:opacity-100 text-gray-400
                             hover:text-red-500 transition-all"
                >
                  <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24"
                    stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </li>
            ))}
          </ul>
        )}

        {showAdd && (
          <div className="flex gap-2 mt-2">
            <input
              type="text"
              value={newPath}
              onChange={e => setNewPath(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && addPath()}
              placeholder="C:\Users\Datos"
              autoFocus
              className="flex-1 text-xs bg-gray-50 border border-gray-200 rounded-lg px-3 py-2
                         font-mono outline-none focus:border-primary focus:ring-1 focus:ring-primary
                         transition-colors"
            />
            <button
              onClick={addPath}
              disabled={saving || !newPath.trim()}
              className="text-xs bg-primary hover:bg-[#3451d1] disabled:opacity-50 text-white
                         font-medium px-3 py-2 rounded-lg transition-colors"
            >
              {saving ? '...' : 'OK'}
            </button>
            <button
              onClick={() => { setShowAdd(false); setNewPath('') }}
              className="text-xs text-gray-400 hover:text-gray-600 px-2 transition-colors"
            >
              ✕
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

function InfoCell({ label, value }: { label: string; value: string }) {
  return (
    <div className="text-center p-2 bg-gray-50 rounded-lg">
      <p className="text-[10px] text-gray-400 uppercase tracking-wide">{label}</p>
      <p className="text-xs font-semibold text-gray-700 mt-0.5 capitalize">{value}</p>
    </div>
  )
}

function EmptyState() {
  return (
    <div className="bg-white border border-gray-200 rounded-xl p-12 text-center">
      <div className="w-12 h-12 bg-gray-50 rounded-xl flex items-center justify-center mx-auto mb-3">
        <svg className="w-6 h-6 text-gray-300" fill="none" viewBox="0 0 24 24"
          stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round"
            d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
        </svg>
      </div>
      <p className="text-sm text-gray-500 font-medium">Sin nodos registrados</p>
      <p className="text-xs text-gray-400 mt-1">
        El agente se registra automáticamente al arrancar el servicio.
      </p>
    </div>
  )
}
