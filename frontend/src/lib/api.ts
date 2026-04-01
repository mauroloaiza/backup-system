import axios from 'axios'

export const api = axios.create({ baseURL: '/api/v1' })

// ── Types ─────────────────────────────────────────────────────────────────────

export interface Node {
  id: string
  name: string
  hostname: string
  os: string
  agent_version: string
  status: string
  last_seen: string
  created_at: string
}

export interface JobRun {
  id: number
  job_id: string
  node_id: string | null
  status: string
  backup_type: string
  files_total: number
  files_done: number
  bytes_total: number
  bytes_done: number
  current_file: string | null
  error_message: string | null
  manifest_path: string | null
  progress_pct: number
  duration_seconds: number | null
  started_at: string
  updated_at: string
  finished_at: string | null
}

export interface DashboardStats {
  total_nodes: number
  nodes_online: number
  nodes_offline: number
  total_runs: number
  runs_today: number
  runs_completed: number
  runs_failed: number
  runs_running: number
  bytes_backed_up_today: number
  bytes_backed_up_total: number
  recent_runs: JobRun[]
  running_now: JobRun[]
}

// ── API calls ─────────────────────────────────────────────────────────────────

export const fetchDashboard = () =>
  api.get<DashboardStats>('/dashboard').then(r => r.data)

export const fetchNodes = () =>
  api.get<Node[]>('/nodes').then(r => r.data)

export const fetchJobs = (params?: { status?: string; node_id?: string; limit?: number }) =>
  api.get<JobRun[]>('/jobs', { params }).then(r => r.data)

export const fetchJob = (jobId: string) =>
  api.get<JobRun>(`/jobs/${jobId}`).then(r => r.data)
