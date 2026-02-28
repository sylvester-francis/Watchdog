import { api } from './client';
import type { SystemInfo, AdminUser, SecurityEvent } from '$lib/types';

export function getSystemInfo(): Promise<SystemInfo> {
	return api.get<SystemInfo>('/api/v1/system');
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
