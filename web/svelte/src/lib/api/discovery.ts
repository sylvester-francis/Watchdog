import { api } from './client';
import type { DiscoveryScan, DiscoveredDevice } from '$lib/types';

interface ScanListResponse {
	data: Array<DiscoveryScan>;
}

interface ScanDetailResponse {
	data: {
		scan: DiscoveryScan;
		devices: DiscoveredDevice[];
	};
}

interface StartScanRequest {
	agent_id: string;
	subnet: string;
	community?: string;
	snmp_version?: string;
}

export function startScan(data: StartScanRequest): Promise<{ data: { id: string; subnet: string; status: string } }> {
	return api.post('/api/v1/discovery', data);
}

export function listScans(): Promise<ScanListResponse> {
	return api.get<ScanListResponse>('/api/v1/discovery');
}

export function getScan(id: string): Promise<ScanDetailResponse> {
	return api.get<ScanDetailResponse>(`/api/v1/discovery/${id}`);
}
