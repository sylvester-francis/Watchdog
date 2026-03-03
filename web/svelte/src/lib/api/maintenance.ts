import { api } from './client';
import type { MaintenanceWindow } from '$lib/types';

interface MaintenanceListResponse {
	data: MaintenanceWindow[];
}

interface MaintenanceCreateRequest {
	agent_id: string;
	name: string;
	starts_at: string;
	ends_at: string;
}

interface MaintenanceUpdateRequest {
	name?: string;
	starts_at?: string;
	ends_at?: string;
}

export function listWindows(): Promise<MaintenanceListResponse> {
	return api.get<MaintenanceListResponse>('/api/v1/maintenance-windows');
}

export function createWindow(data: MaintenanceCreateRequest): Promise<{ data: MaintenanceWindow }> {
	return api.post<{ data: MaintenanceWindow }>('/api/v1/maintenance-windows', data);
}

export function updateWindow(id: string, data: MaintenanceUpdateRequest): Promise<{ data: MaintenanceWindow }> {
	return api.request<{ data: MaintenanceWindow }>(`/api/v1/maintenance-windows/${id}`, {
		method: 'PUT',
		body: JSON.stringify(data)
	});
}

export function deleteWindow(id: string): Promise<void> {
	return api.delete<void>(`/api/v1/maintenance-windows/${id}`);
}
