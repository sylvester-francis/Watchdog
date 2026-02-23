import { api } from './client';
import type { SystemInfo } from '$lib/types';

export function getSystemInfo(): Promise<SystemInfo> {
	return api.get<SystemInfo>('/api/v1/system');
}
