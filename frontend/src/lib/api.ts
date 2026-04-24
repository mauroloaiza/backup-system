import axios from 'axios'

export const api = axios.create({ baseURL: '/api/v1' })

// Redirect to login on 401
api.interceptors.response.use(
  r => r,
  err => {
    if (err.response?.status === 401 && !window.location.pathname.includes('login')) {
      // Clear stored token and redirect
      localStorage.removeItem('backupsmc-auth')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

// ── Types ─────────────────────────────────────────────────────────────────────

export interface Node {
  id: string
  name: string
  hostname: string
  os: string
  agent_version: string
  status: string
  source_paths: string[]
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

export const updateNodeSourcePaths = (nodeId: string, source_paths: string[]) =>
  api.put<Node>(`/nodes/${nodeId}/source-paths`, { source_paths }).then(r => r.data)

// ── Remote agent config editor ────────────────────────────────────────────────

export interface NodeConfigPayload {
  source_paths: string[]
  exclude_patterns: string[]
  schedule_interval_minutes: number
  use_vss: boolean
  incremental: boolean
  verify_after_backup: boolean
  throttle_mbps: number
  pre_script: string | null
  post_script: string | null
  retention_days: number
  gfs_enabled: boolean
  gfs_keep_daily: number
  gfs_keep_weekly: number
  gfs_keep_monthly: number
  retry_max_attempts: number
  retry_initial_delay_seconds: number
  log_level: 'debug' | 'info' | 'warn' | 'error'
  email: {
    enabled: boolean
    on_failure: boolean
    on_success: boolean
    to: string[]
  }
}

export interface NodeConfig {
  node_id: string
  version: number
  payload: NodeConfigPayload
  updated_by: string | null
  updated_at: string | null
  last_pulled_version: number
  last_pulled_at: string | null
  in_sync: boolean
}

export const fetchNodeConfig = (nodeId: string) =>
  api.get<NodeConfig>(`/nodes/${nodeId}/config`).then(r => r.data)

export const updateNodeConfig = (nodeId: string, payload: NodeConfigPayload) =>
  api.put<NodeConfig>(`/nodes/${nodeId}/config`, { payload }).then(r => r.data)

// ── Destinations ──────────────────────────────────────────────────────────────

export interface DestinationAggregate {
  node_id: string
  node_name: string
  hostname: string
  node_status: 'online' | 'stale' | 'offline'
  type: 'local' | 's3' | 'sftp'
  target: string
  details: Record<string, string> | null
  bytes_backed_up: number
  runs_count: number
  last_backup_at: string | null
  last_status: string | null
}

export const fetchDestinations = () =>
  api.get<DestinationAggregate[]>('/destinations').then(r => r.data)

export interface DailyStatPoint {
  date: string
  total: number
  completed: number
  failed: number
  bytes: number
  duration_avg: number
}

export interface HistoryStats {
  points: DailyStatPoint[]
  success_rate: number
  avg_duration: number
  total_bytes: number
  total_runs: number
}

export const fetchHistory = (days = 30) =>
  api.get<HistoryStats>('/stats/history', { params: { days } }).then(r => r.data)

// ── Tokens ────────────────────────────────────────────────────────────────────

export interface AgentToken {
  id: number
  name: string
  token: string
  is_active: boolean
  last_used_at: string | null
  created_at: string
}

export const fetchTokens = () =>
  api.get<AgentToken[]>('/tokens').then(r => r.data)

export const createToken = (name: string) =>
  api.post<AgentToken>('/tokens', { name }).then(r => r.data)

export const revokeToken = (id: number) =>
  api.delete(`/tokens/${id}`).then(r => r.data)

// ── Settings ──────────────────────────────────────────────────────────────────

export interface Settings {
  server_version: string
  agent_token: string | null
  notify_email_enabled: boolean
  notify_email_to: string
  notify_on_failure: boolean
  notify_on_success: boolean
  notify_daily_summary: boolean
}

export const fetchSettings = () =>
  api.get<Settings>('/settings').then(r => r.data)

export const updateSettings = (data: Partial<Omit<Settings, 'server_version' | 'agent_token'>>) =>
  api.put<Settings>('/settings', data).then(r => r.data)

// ── Mail (SMTP) Settings ──────────────────────────────────────────────────────

export interface EmailSettings {
  outbound_provider: 'smtp' | 'office365'
  from_name: string
  from_email: string | null
  smtp_host: string | null
  smtp_port: number
  smtp_secure: boolean
  smtp_user: string | null
  smtp_pass: string | null   // masked ("••••••••") when set
  configured: boolean
}

export const fetchMailSettings = () =>
  api.get<EmailSettings>('/mail-settings').then(r => r.data)

export const updateMailSettings = (data: Partial<EmailSettings>) =>
  api.put<EmailSettings>('/mail-settings', data).then(r => r.data)

export const testMailSettings = (to?: string) =>
  api.post<{ ok: boolean; sent_to: string }>('/mail-settings/test', { to }).then(r => r.data)

export const sendDailySummaryTest = (to?: string) =>
  api.post<{ ok: boolean; sent: boolean; sent_to?: string; summary?: any }>(
    '/mail-settings/test-daily',
    { to },
  ).then(r => r.data)

export interface DailySummaryPreview {
  window_hours: number
  generated_at: string
  totals: {
    runs: number
    completed: number
    failed: number
    warning: number
    bytes: number
    nodes_active: number
    nodes_total: number
    nodes_offline: number
    success_rate: number
  }
  per_node: Array<{
    node_id: string
    node_name: string
    hostname: string
    runs: number
    completed: number
    failed: number
    warning: number
    running: number
    bytes: number
    last_at: string | null
  }>
  offline_nodes: Array<{ name: string; hostname: string; last_seen: string | null }>
}

export const fetchDailySummaryPreview = () =>
  api.get<DailySummaryPreview>('/mail-settings/daily-preview').then(r => r.data)


// ── Notifications ─────────────────────────────────────────────────────────────

export interface AppNotification {
  id: number
  type: string
  title: string
  body: string | null
  entity_type: string | null
  entity_id: string | null
  read: boolean
  created_at: string
}

export interface NotificationList {
  notifications: AppNotification[]
  unread: number
}

export const fetchNotifications = (unread = false) =>
  api.get<NotificationList>('/notifications', { params: { unread } }).then(r => r.data)

export const markNotificationRead = (id: number) =>
  api.patch(`/notifications/${id}/read`).then(r => r.data)

export const markAllNotificationsRead = () =>
  api.post('/notifications/mark-all-read').then(r => r.data)

// ── Restore Requests ──────────────────────────────────────────────────────────

export interface RestoreRequest {
  id: number
  node_id: string
  source_job_id: string
  target_path: string
  filter_pattern: string | null
  dry_run: boolean
  status: 'queued' | 'running' | 'completed' | 'failed' | 'cancelled'
  message: string | null
  files_restored: number
  bytes_restored: number
  requested_by: string | null
  created_at: string
  started_at: string | null
  finished_at: string | null
  updated_at: string
}

export interface RestoreRequestCreate {
  source_job_id: string
  target_path: string
  filter_pattern?: string
  dry_run?: boolean
}

export const createRestoreRequest = (nodeId: string, data: RestoreRequestCreate) =>
  api.post<RestoreRequest>(`/nodes/${nodeId}/restore`, data).then(r => r.data)

export const fetchRestoreRequests = (params?: { node_id?: string; status?: string; limit?: number }) =>
  api.get<RestoreRequest[]>('/restore', { params }).then(r => r.data)

export const fetchRestoreRequest = (id: number) =>
  api.get<RestoreRequest>(`/restore/${id}`).then(r => r.data)

export const cancelRestoreRequest = (id: number) =>
  api.post<RestoreRequest>(`/restore/${id}/cancel`).then(r => r.data)
