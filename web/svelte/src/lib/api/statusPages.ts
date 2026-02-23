import { api } from './client';
import type { StatusPage, PublicStatusPageData } from '$lib/types';

interface StatusPageListResponse {
	data: StatusPage[];
}

interface StatusPageCreateRequest {
	name: string;
	description?: string;
}

interface StatusPageUpdateRequest {
	name: string;
	description: string;
	is_public: boolean;
	monitor_ids: string[];
}

interface AvailableMonitor {
	id: string;
	name: string;
	type: string;
	target: string;
	status: string;
}

interface StatusPageDetailResponse {
	data: StatusPage;
	available_monitors: AvailableMonitor[];
}

export function listStatusPages(): Promise<StatusPageListResponse> {
	return api.get<StatusPageListResponse>('/api/v1/status-pages');
}

export function createStatusPage(data: StatusPageCreateRequest): Promise<{ data: StatusPage }> {
	return api.post<{ data: StatusPage }>('/api/v1/status-pages', data);
}

export function getStatusPage(id: string): Promise<StatusPageDetailResponse> {
	return api.get<StatusPageDetailResponse>(`/api/v1/status-pages/${id}`);
}

export function updateStatusPage(id: string, data: StatusPageUpdateRequest): Promise<{ data: StatusPage }> {
	return api.put<{ data: StatusPage }>(`/api/v1/status-pages/${id}`, data);
}

export function deleteStatusPage(id: string): Promise<void> {
	return api.delete<void>(`/api/v1/status-pages/${id}`);
}

export function getPublicStatusPage(username: string, slug: string): Promise<PublicStatusPageData> {
	return api.get<PublicStatusPageData>(`/api/v1/public/status/${username}/${slug}`);
}
