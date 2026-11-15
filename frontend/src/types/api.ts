// Wire types for the action and docker-logs endpoints (contract section 3).

export interface ActionRequestBody {
  action: string
  target: string
  params?: Record<string, unknown>
}

export interface ActionResponse {
  status: 'success'
  action: string
  target: string
  output?: string
}

export interface ApiError {
  error: string
}

export interface DockerLogLine {
  stream: 'stdout' | 'stderr'
  text: string
}

export interface DockerLogs {
  container: string
  lines: DockerLogLine[]
}
