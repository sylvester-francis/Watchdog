import { api } from './client';
import type { Agent } from '$lib/types';

interface CreateAgentResponse {
	data: {
		id: string;
		name: string;
		api_key: string;
	};
}

export function listAgents(): Promise<{ data: Agent[] }> {
	return api.get<{ data: Agent[] }>('/api/v1/agents');
}

export function createAgent(name: string): Promise<CreateAgentResponse> {
	return api.post<CreateAgentResponse>('/api/v1/agents', { name });
}

export function deleteAgent(id: string): Promise<void> {
	return api.delete<void>(`/api/v1/agents/${id}`);
}
