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

export function register(email: string, password: string, confirm_password: string, website?: string): Promise<AuthResponse> {
	return api.post<AuthResponse>('/api/v1/auth/register', { email, password, confirm_password, website });
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

export interface PasswordResetMessage {
	message: string;
}

// Always returns a generic 200 message — backend doesn't reveal whether the
// email matches a real user (anti-enumeration).
export function requestPasswordReset(email: string): Promise<PasswordResetMessage> {
	return api.post<PasswordResetMessage>('/api/v1/auth/password/request', { email });
}

// Validates the reset token, updates the password, invalidates existing sessions.
// 400 means the link is no good (expired, used, or wrong) — request a new one.
export function completePasswordReset(token: string, new_password: string): Promise<PasswordResetMessage> {
	return api.post<PasswordResetMessage>('/api/v1/auth/password/reset', { token, new_password });
}
