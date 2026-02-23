import { api } from './client';
import type { User } from '$lib/types';

export interface AuthResponse {
	user: User;
	must_change_password?: boolean;
}

interface NeedsSetupResponse {
	needs_setup: boolean;
}

export function login(email: string, password: string): Promise<AuthResponse> {
	return api.post<AuthResponse>('/api/v1/auth/login', { email, password });
}

export function register(email: string, password: string, confirm_password: string): Promise<AuthResponse> {
	return api.post<AuthResponse>('/api/v1/auth/register', { email, password, confirm_password });
}

export function logout(): Promise<void> {
	return api.post<void>('/api/v1/auth/logout');
}

export function me(): Promise<AuthResponse> {
	return api.get<AuthResponse>('/api/v1/auth/me');
}

export function setup(email: string, password: string, confirm_password: string): Promise<AuthResponse> {
	return api.post<AuthResponse>('/api/v1/auth/setup', { email, password, confirm_password });
}

export function needsSetup(): Promise<NeedsSetupResponse> {
	return api.get<NeedsSetupResponse>('/api/v1/auth/needs-setup');
}
