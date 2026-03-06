import { api } from './client';
import type { SystemInfo, AdminUser, SecurityEvent, AuditLogParams, PaginatedAuditResponse, MetricsResponse } from '$lib/types';

export function getSystemInfo(): Promise<SystemInfo> {
	return api.get<SystemInfo>('/api/v1/system');
}

export function getAuditLogs(params: AuditLogParams): Promise<PaginatedAuditResponse> {
	const searchParams = new URLSearchParams();
	searchParams.set('page', String(params.page));
	searchParams.set('per_page', String(params.per_page));
	if (params.action) searchParams.set('action', params.action);
	if (params.from) searchParams.set('from', params.from);
	if (params.to) searchParams.set('to', params.to);
	return api.get<PaginatedAuditResponse>(`/api/v1/audit-logs?${searchParams.toString()}`);
}

export function listUsers(): Promise<{ data: AdminUser[] }> {
	return api.get<{ data: AdminUser[] }>('/api/v1/admin/users');
}

export function resetUserPassword(userId: string): Promise<{ password: string }> {
	return api.post<{ password: string }>(`/api/v1/admin/users/${userId}/reset-password`);
}

export function deleteUser(userId: string): Promise<{ status: string }> {
	return api.delete<{ status: string }>(`/api/v1/admin/users/${userId}`);
}

export function getSecurityEvents(): Promise<{ data: SecurityEvent[] }> {
	return api.get<{ data: SecurityEvent[] }>('/api/v1/admin/security-events');
}

export function getMetrics(): Promise<MetricsResponse> {
	return api.get<MetricsResponse>('/api/v1/system/metrics');
}
