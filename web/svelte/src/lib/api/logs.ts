import { api } from './client';
import type { LogRecord } from '$lib/types';

interface LogListResponse {
	data: LogRecord[];
}

export interface ListLogsParams {
	since?: string;
	service?: string;
	severity?: string;
	trace_id?: string;
	span_id?: string;
	before?: string; // RFC3339 keyset cursor — returns logs strictly older than this
	limit?: number;
}

export function listLogs(params: ListLogsParams = {}): Promise<LogListResponse> {
	const search = new URLSearchParams();
	if (params.since) search.set('since', params.since);
	if (params.service) search.set('service', params.service);
	if (params.severity) search.set('severity', params.severity);
	if (params.trace_id) search.set('trace_id', params.trace_id);
	if (params.span_id) search.set('span_id', params.span_id);
	if (params.before) search.set('before', params.before);
	if (params.limit !== undefined) search.set('limit', String(params.limit));
	const query = search.toString();
	return api.get<LogListResponse>(`/api/v1/logs${query ? `?${query}` : ''}`);
}
