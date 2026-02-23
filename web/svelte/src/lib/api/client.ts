import type { APIError } from '$lib/types';

type UnauthorizedCallback = () => void;
let onUnauthorized: UnauthorizedCallback | null = null;

/** Register a callback for 401 responses (called from app layout). */
export function setOnUnauthorized(cb: UnauthorizedCallback): void {
	onUnauthorized = cb;
}

class APIClient {
	private baseURL = '';

	async request<T>(path: string, options: RequestInit = {}): Promise<T> {
		const url = `${this.baseURL}${path}`;
		const response = await fetch(url, {
			credentials: 'include',
			headers: {
				'Content-Type': 'application/json',
				...options.headers
			},
			...options
		});

		if (response.status === 401) {
			onUnauthorized?.();
			throw new Error('Unauthorized');
		}

		if (!response.ok) {
			const body = await response.json().catch(() => ({ error: 'Request failed' })) as APIError;
			throw new Error(body.error || `HTTP ${response.status}`);
		}

		if (response.status === 204) {
			return undefined as T;
		}

		return response.json() as Promise<T>;
	}

	get<T>(path: string): Promise<T> {
		return this.request<T>(path, { method: 'GET' });
	}

	post<T>(path: string, body?: unknown): Promise<T> {
		return this.request<T>(path, {
			method: 'POST',
			body: body ? JSON.stringify(body) : undefined
		});
	}

	put<T>(path: string, body?: unknown): Promise<T> {
		return this.request<T>(path, {
			method: 'PUT',
			body: body ? JSON.stringify(body) : undefined
		});
	}

	delete<T>(path: string): Promise<T> {
		return this.request<T>(path, { method: 'DELETE' });
	}
}

export const api = new APIClient();
