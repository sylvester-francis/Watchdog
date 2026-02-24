import { api } from './client';
import type { Incident } from '$lib/types';

export function listIncidents(status?: string): Promise<{ data: Incident[] }> {
	const params = status ? `?status=${encodeURIComponent(status)}` : '';
	return api.get<{ data: Incident[] }>(`/api/v1/incidents${params}`);
}

export function acknowledgeIncident(id: string): Promise<{ status: string }> {
	return api.post<{ status: string }>(`/api/v1/incidents/${id}/acknowledge`);
}

export function resolveIncident(id: string): Promise<{ status: string }> {
	return api.post<{ status: string }>(`/api/v1/incidents/${id}/resolve`);
}
