import { system as systemApi } from '$lib/api';
import type { SystemInfo, AdminUser, MetricsResponse } from '$lib/types';

export async function load() {
	const [dataResult, metricsResult, usersResult] = await Promise.allSettled([
		systemApi.getSystemInfo(),
		systemApi.getMetrics(),
		systemApi.listUsers()
	]);

	return {
		data: dataResult.status === 'fulfilled' ? dataResult.value : null,
		metrics: metricsResult.status === 'fulfilled' ? metricsResult.value : null,
		users: usersResult.status === 'fulfilled' ? usersResult.value.data ?? [] : [],
		error:
			dataResult.status === 'rejected'
				? dataResult.reason instanceof Error
					? dataResult.reason.message
					: 'Failed to load system info'
				: ''
	};
}
