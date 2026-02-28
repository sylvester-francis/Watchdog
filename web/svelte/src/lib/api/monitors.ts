import { api } from './client';
import type { Monitor, MonitorSummary, HeartbeatPoint, LatencyPoint, DashboardStats, CertDetails, SLAResponse } from '$lib/types';

interface MonitorListResponse {
	data: Monitor[];
}

interface MonitorDetailResponse {
	data: Monitor;
	heartbeats: {
		latencies: number[];
		uptime_up: number;
		uptime_down: number;
		total: number;
	};
}

interface MonitorCreateRequest {
	agent_id: string;
	name: string;
	type: string;
	target: string;
	interval_seconds?: number;
	timeout_seconds?: number;
	failure_threshold?: number;
	metadata?: Record<string, string>;
}

interface MonitorUpdateRequest {
	name?: string;
	target?: string;
	interval_seconds?: number;
	timeout_seconds?: number;
	failure_threshold?: number;
	enabled?: boolean;
	sla_target_percent?: number;
	agent_id?: string;
}

export function listMonitors(): Promise<MonitorListResponse> {
	return api.get<MonitorListResponse>('/api/v1/monitors');
}

export function getMonitor(id: string): Promise<MonitorDetailResponse> {
	return api.get<MonitorDetailResponse>(`/api/v1/monitors/${id}`);
}

export function createMonitor(data: MonitorCreateRequest): Promise<{ data: Monitor }> {
	return api.post<{ data: Monitor }>('/api/v1/monitors', data);
}

export function updateMonitor(id: string, data: MonitorUpdateRequest): Promise<{ data: Monitor }> {
	return api.put<{ data: Monitor }>(`/api/v1/monitors/${id}`, data);
}

export function deleteMonitor(id: string): Promise<void> {
	return api.delete<void>(`/api/v1/monitors/${id}`);
}

export function getHeartbeats(monitorId: string, period?: string): Promise<HeartbeatPoint[]> {
	const query = period ? `?period=${period}` : '';
	return api.get<HeartbeatPoint[]>(`/api/v1/monitors/${monitorId}/heartbeats${query}`);
}

export function getLatencyHistory(monitorId: string, period?: string): Promise<LatencyPoint[]> {
	const query = period ? `?period=${period}` : '';
	return api.get<LatencyPoint[]>(`/api/v1/monitors/${monitorId}/latency${query}`);
}

export function getDashboardStats(): Promise<DashboardStats> {
	return api.get<DashboardStats>('/api/v1/dashboard/stats');
}

export function getMonitorsSummary(): Promise<MonitorSummary[]> {
	return api.get<MonitorSummary[]>('/api/v1/monitors/summary');
}

export function getCertDetails(monitorId: string): Promise<{ data: CertDetails }> {
	return api.get<{ data: CertDetails }>(`/api/v1/monitors/${monitorId}/certificate`);
}

export function getExpiringCertificates(days = 30): Promise<{ data: CertDetails[] }> {
	return api.get<{ data: CertDetails[] }>(`/api/v1/certificates/expiring?days=${days}`);
}

export function getMonitorSLA(monitorId: string, period = '30d'): Promise<{ data: SLAResponse }> {
	return api.get<{ data: SLAResponse }>(`/api/v1/monitors/${monitorId}/sla?period=${period}`);
}
