import { api } from './client';
import type { APIToken, AlertChannel } from '$lib/types';

interface TokenListResponse {
	data: APIToken[];
}

interface TokenCreateRequest {
	name: string;
	scope?: 'admin' | 'read_only';
	expires?: '30d' | '90d' | '';
}

interface TokenCreateResponse {
	data: APIToken;
	plaintext: string;
}

interface ChannelListResponse {
	data: AlertChannel[];
}

interface ChannelCreateRequest {
	type: string;
	name: string;
	config: Record<string, string>;
}

interface ProfileUpdateRequest {
	username: string;
}

export function listTokens(): Promise<TokenListResponse> {
	return api.get<TokenListResponse>('/api/v1/tokens');
}

export function createToken(data: TokenCreateRequest): Promise<TokenCreateResponse> {
	return api.post<TokenCreateResponse>('/api/v1/tokens', data);
}

export function deleteToken(id: string): Promise<void> {
	return api.delete<void>(`/api/v1/tokens/${id}`);
}

export function regenerateToken(id: string): Promise<TokenCreateResponse> {
	return api.post<TokenCreateResponse>(`/api/v1/tokens/${id}/regenerate`);
}

export function listChannels(): Promise<ChannelListResponse> {
	return api.get<ChannelListResponse>('/api/v1/alert-channels');
}

export function createChannel(data: ChannelCreateRequest): Promise<{ data: AlertChannel }> {
	return api.post<{ data: AlertChannel }>('/api/v1/alert-channels', data);
}

export function deleteChannel(id: string): Promise<void> {
	return api.delete<void>(`/api/v1/alert-channels/${id}`);
}

export function toggleChannel(id: string): Promise<{ data: AlertChannel }> {
	return api.post<{ data: AlertChannel }>(`/api/v1/alert-channels/${id}/toggle`);
}

export function testChannel(id: string): Promise<{ status: string }> {
	return api.post<{ status: string }>(`/api/v1/alert-channels/${id}/test`);
}

export function updateProfile(data: ProfileUpdateRequest): Promise<{ data: { username: string } }> {
	return api.request<{ data: { username: string } }>('/api/v1/users/me', {
		method: 'PATCH',
		body: JSON.stringify(data)
	});
}
