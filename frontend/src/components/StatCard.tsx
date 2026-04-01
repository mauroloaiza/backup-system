import { ReactNode } from 'react'

interface StatCardProps {
  label: string
  value: string | number
  sub?: string
  subColor?: string
  icon: ReactNode
  iconBg: string
}

export function StatCard({ label, value, sub, subColor, icon, iconBg }: StatCardProps) {
  return (
    <div className="bg-white border border-gray-200 rounded-xl p-4 flex items-center gap-3.5 shadow-sm">
      <div className={`w-11 h-11 rounded-[10px] flex items-center justify-center flex-shrink-0 ${iconBg}`}>
        {icon}
      </div>
      <div>
        <div className="text-2xl font-bold text-gray-800 leading-tight">{value}</div>
        <div className="text-xs text-gray-400 mt-0.5">{label}</div>
        {sub && (
          <div className={`text-[10px] mt-0.5 ${subColor ?? 'text-gray-400'}`}>{sub}</div>
        )}
      </div>
    </div>
  )
}
