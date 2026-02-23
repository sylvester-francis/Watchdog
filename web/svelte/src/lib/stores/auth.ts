import { auth as authApi } from '$lib/api';
import type { User } from '$lib/types';

let user = $state<User | null>(null);
let loading = $state(true);
let checked = $state(false);

export function getAuth() {
	function isAuthenticated(): boolean {
		return user !== null;
	}

	function isAdmin(): boolean {
		return user?.is_admin ?? false;
	}

	async function check(): Promise<User | null> {
		if (checked) return user;
		loading = true;
		try {
			const res = await authApi.me();
			user = res.user;
		} catch {
			user = null;
		} finally {
			loading = false;
			checked = true;
		}
		return user;
	}

	async function login(email: string, password: string): Promise<User> {
		const res = await authApi.login(email, password);
		user = res.user;
		checked = true;
		return res.user;
	}

	async function register(email: string, password: string, confirmPassword: string): Promise<User> {
		const res = await authApi.register(email, password, confirmPassword);
		user = res.user;
		checked = true;
		return res.user;
	}

	async function setupAdmin(email: string, password: string, confirmPassword: string): Promise<User> {
		const res = await authApi.setup(email, password, confirmPassword);
		user = res.user;
		checked = true;
		return res.user;
	}

	async function logout(): Promise<void> {
		try {
			await authApi.logout();
		} finally {
			user = null;
			checked = false;
		}
	}

	return {
		get user() { return user; },
		get loading() { return loading; },
		get isAuthenticated() { return isAuthenticated(); },
		get isAdmin() { return isAdmin(); },
		check,
		login,
		register,
		setupAdmin,
		logout
	};
}
