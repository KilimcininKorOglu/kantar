export interface SystemStatus {
  status: string
  version: string
  uptime: string
  goVersion: string
  numCpu: number
  goroutines: number
  memory: {
    allocBytes: number
    totalAllocBytes: number
    sysBytes: number
    numGc: number
  }
}

export interface Package {
  id: number
  registryType: string
  name: string
  description: string
  license: string
  homepage: string
  repository: string
  status: 'pending' | 'approved' | 'blocked'
  requestedBy: string
  approvedBy: string
  blockedReason: string
  createdAt: string
  updatedAt: string
}

export interface PackageVersion {
  id: number
  packageId: number
  version: string
  size: number
  checksumSha256: string
  storagePath: string
  deprecated: boolean
  yanked: boolean
  syncedAt: string
  createdAt: string
}

export interface User {
  id: number
  username: string
  email: string
  role: string
  active: boolean
  timezone: string
  locale: string
  createdAt: string
  updatedAt: string
}

export interface AuditLogEntry {
  id: number
  timestamp: string
  event: string
  actorUsername: string
  actorIp: string
  actorUserAgent: string
  resourceRegistry: string
  resourcePackage: string
  resourceVersion: string
  result: string
  metadataJson: string
  prevHash: string
  hash: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  expiresAt: string
  user: User
}

export interface RegistryInfo {
  ecosystem: string
  mode: string
  upstream: string
  enabled: boolean
  packageCount: number
  storageSize: number
  lastSync: string
  status: 'healthy' | 'degraded' | 'offline'
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  perPage: number
}
