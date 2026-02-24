export interface User {
	id: string;
	email: string;
	username: string;
	plan: string;
	is_admin: boolean;
	created_at: string;
}

export interface Agent {
	id: string;
	name: string;
	status: 'online' | 'offline';
	last_seen_at: string | null;
	created_at: string;
}

export interface Monitor {
	id: string;
	agent_id: string;
	name: string;
	type: MonitorType;
	target: string;
	status: MonitorStatus;
	enabled: boolean;
	interval_seconds: number;
	timeout_seconds: number;
	failure_threshold: number;
	metadata?: Record<string, string>;
	created_at: string;
}

export type MonitorType = 'ping' | 'http' | 'tcp' | 'dns' | 'tls' | 'docker' | 'database' | 'system';
export type MonitorStatus = 'pending' | 'up' | 'down' | 'degraded';

export interface Incident {
	id: string;
	monitor_id: string;
	status: IncidentStatus;
	started_at: string;
	resolved_at: string | null;
	acknowledged_at: string | null;
	ttr_seconds: number | null;
}

export type IncidentStatus = 'open' | 'acknowledged' | 'resolved';

export interface AlertChannel {
	id: string;
	type: AlertChannelType;
	name: string;
	config: Record<string, string>;
	enabled: boolean;
	created_at: string;
	updated_at: string;
}

export type AlertChannelType = 'discord' | 'slack' | 'email' | 'telegram' | 'pagerduty' | 'webhook';

export interface APIToken {
	id: string;
	name: string;
	prefix: string;
	scope: 'admin' | 'read_only';
	last_used_at: string | null;
	last_used_ip: string | null;
	expires_at: string | null;
	created_at: string;
}

export interface StatusPage {
	id: string;
	name: string;
	slug: string;
	description: string;
	is_public: boolean;
	monitor_ids: string[];
	created_at: string;
	updated_at: string;
}

export interface SystemInfo {
	db: {
		healthy: boolean;
		ping_ms: number;
		pool: { acquired: number; idle: number; total: number; max: number };
		size: string;
		table_sizes: { name: string; size: string }[];
		migration: { version: number; dirty: boolean };
	};
	runtime: {
		uptime_seconds: number;
		uptime_formatted: string;
		goroutines: number;
		heap_mb: number;
		stack_mb: number;
		gc_pause_ms: number;
	};
	agents_connected: number;
	heartbeats: {
		total_last_hour: number;
		per_minute: number;
		errors_last_hour: number;
	};
	audit_logs: AuditLogEntry[];
}

export interface AuditLogEntry {
	id: string;
	action: string;
	user_email: string;
	ip_address: string;
	metadata: Record<string, string>;
	created_at: string;
}

export interface DashboardStats {
	total_monitors: number;
	monitors_up: number;
	monitors_down: number;
	active_incidents: number;
	total_agents: number;
	online_agents: number;
}

export interface HeartbeatPoint {
	time: string;
	status: 'up' | 'down' | 'timeout' | 'error';
	latency_ms: number | null;
	error_message?: string;
}

export interface LatencyPoint {
	time: string;
	avg_ms: number;
	min_ms: number;
	max_ms: number;
}

export interface MonitorSummary {
	id: string;
	name: string;
	status: string;
	type: string;
	target: string;
	interval_seconds: number;
	latencies: number[];
	uptimeUp: number;
	uptimeDown: number;
	total: number;
	latest_value?: string;
}

export interface AdminUser {
	id: string;
	email: string;
	username: string;
	plan: string;
	is_admin: boolean;
	agent_count: number;
	monitor_count: number;
	created_at: string;
}

export interface APIError {
	error: string;
}

// Public status page types
export interface PublicStatusPageData {
	page: { name: string; description: string };
	monitors: PublicMonitorData[];
	incidents: PublicIncidentData[];
	overall_status: string;
	all_up: boolean;
	aggregate_uptime: number;
}

export interface PublicMonitorData {
	name: string;
	type: string;
	status: string;
	uptime_percent: number;
	latency_ms: number;
	has_latency: boolean;
	metric_value?: string;
	monitoring_since: string;
	data_days: number;
	uptime_history: { date: string; percent: number }[];
}

export interface PublicIncidentData {
	monitor_name: string;
	started_at: string;
	resolved_at: string | null;
	duration_seconds: number;
	status: string;
	is_active: boolean;
}
