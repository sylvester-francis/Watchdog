import { api } from './client';
import type { Span, TraceSummary } from '$lib/types';

interface TraceListResponse {
	data: TraceSummary[];
}

interface TraceDetailResponse {
	data: Span[];
}

export interface ListTracesParams {
	since?: string; // RFC3339 timestamp
	service?: string;
	limit?: number;
}

export function listTraces(params: ListTracesParams = {}): Promise<TraceListResponse> {
	const search = new URLSearchParams();
	if (params.since) search.set('since', params.since);
	if (params.service) search.set('service', params.service);
	if (params.limit !== undefined) search.set('limit', String(params.limit));
	const query = search.toString();
	return api.get<TraceListResponse>(`/api/v1/traces${query ? `?${query}` : ''}`);
}

export function getTrace(traceId: string): Promise<TraceDetailResponse> {
	return api.get<TraceDetailResponse>(`/api/v1/traces/${traceId}`);
}
