import { auth as authApi } from '$lib/api';
import type { User } from '$lib/types';

let user = $state<User | null>(null);
let loading = $state(true);
let checked = $state(false);
let mustChangePassword = $state(false);
let lastCheckAt = 0;

// Cross-tab logout coordination
const channel = typeof BroadcastChannel !== 'undefined' ? new BroadcastChannel('watchdog-auth') : null;
channel?.addEventListener('message', (e) => {
	if (e.data === 'logout') {
		user = null;
		checked = false;
	}
});

// Revalidate session when tab regains focus (stale auth cache fix)
const REVALIDATE_INTERVAL_MS = 5 * 60 * 1000; // 5 minutes
if (typeof document !== 'undefined') {
	document.addEventListener('visibilitychange', () => {
		if (document.visibilityState === 'visible' && checked && Date.now() - lastCheckAt > REVALIDATE_INTERVAL_MS) {
			revalidate();
		}
	});
}

async function revalidate(): Promise<void> {
	try {
		const res = await authApi.me();
		user = res.user;
		mustChangePassword = res.must_change_password === true;
		lastCheckAt = Date.now();
	} catch {
		user = null;
		checked = false;
	}
}

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
			mustChangePassword = res.must_change_password === true;
			lastCheckAt = Date.now();
		} catch {
			user = null;
		} finally {
			loading = false;
			checked = true;
		}
		return user;
	}

	async function login(email: string, password: string): Promise<{ user: User; must_change_password?: boolean }> {
		const res = await authApi.login(email, password);
		user = res.user;
		mustChangePassword = res.must_change_password === true;
		checked = true;
		lastCheckAt = Date.now();
		return res;
	}

	async function register(email: string, password: string, confirmPassword: string, website?: string): Promise<User> {
		const res = await authApi.register(email, password, confirmPassword, website);
		user = res.user;
		checked = true;
		lastCheckAt = Date.now();
		return res.user;
	}

	async function setupAdmin(email: string, password: string, confirmPassword: string): Promise<User> {
		const res = await authApi.setup(email, password, confirmPassword);
		user = res.user;
		checked = true;
		lastCheckAt = Date.now();
		return res.user;
	}

	async function logout(): Promise<void> {
		try {
			await authApi.logout();
		} finally {
			user = null;
			checked = false;
			mustChangePassword = false;
			channel?.postMessage('logout');
		}
	}

	function clearMustChangePassword(): void {
		mustChangePassword = false;
	}

	return {
		get user() { return user; },
		get loading() { return loading; },
		get isAuthenticated() { return isAuthenticated(); },
		get isAdmin() { return isAdmin(); },
		get mustChangePassword() { return mustChangePassword; },
		check,
		login,
		register,
		setupAdmin,
		logout,
		clearMustChangePassword
	};
}
