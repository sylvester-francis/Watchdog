<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Plus,
		Copy,
		Check,
		Loader2,
		AlertCircle,
		MessageCircle,
		Hash,
		Mail,
		Send,
		PhoneCall,
		Webhook,
		AlertTriangle,
		X
	} from 'lucide-svelte';
	import { Alert, Skeleton } from '@sylvester-francis/watchdog-ui';
	import { settings as settingsApi, maintenance as maintenanceApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast.svelte';
	import { getAuth } from '$lib/stores/auth.svelte';
	import { page } from '$app/state';
	import type { APIToken, AlertChannel, AlertChannelType, MaintenanceWindow } from '$lib/types';
	import CreateChannelModal from '$lib/components/settings/CreateChannelModal.svelte';
	import CreateTokenModal from '$lib/components/settings/CreateTokenModal.svelte';
	import CreateMaintenanceModal from '$lib/components/settings/CreateMaintenanceModal.svelte';
	import ConfirmModal from '$lib/components/ConfirmModal.svelte';

	const toast = getToasts();
	const auth = getAuth();

	// Force change password banner
	const forceChange = $derived(page.url.searchParams.get('change_password') === '1');
	let showForceChangeBanner = $state(false);

	// Data
	let tokens = $state<APIToken[]>([]);
	let channels = $state<AlertChannel[]>([]);
	let maintenanceWindows = $state<MaintenanceWindow[]>([]);
	let loading = $state(true);

	// Username form
	let username = $state('');
	let usernameLoading = $state(false);
	let usernameSuccess = $state('');
	let usernameError = $state('');
	let editingUsername = $state(false);
	let usernameDraft = $state('');

	// Change password form
	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmNewPassword = $state('');
	let passwordLoading = $state(false);
	let passwordSuccess = $state('');
	let passwordError = $state('');
	let editingPassword = $state(false);

	// Modals
	let showChannelModal = $state(false);
	let showTokenModal = $state(false);
	let showMaintenanceModal = $state(false);

	// Token plaintext display (after create or regenerate)
	let plaintextToken = $state('');
	let plaintextTokenName = $state('');
	let copiedToken = $state(false);

	// Per-channel action states
	let testingChannelId = $state<string | null>(null);
	let channelTestResult = $state<Record<string, { ok: boolean; message: string }>>({});
	let togglingChannelId = $state<string | null>(null);

	// Confirm modal state
	let confirmModal = $state<{
		open: boolean;
		title: string;
		message: string;
		confirmLabel: string;
		variant: 'danger' | 'warning';
		loading: boolean;
		action: (() => Promise<void>) | null;
	}>({
		open: false,
		title: '',
		message: '',
		confirmLabel: 'Confirm',
		variant: 'danger',
		loading: false,
		action: null
	});

	const inlineInputClass =
		'w-full px-2.5 py-1.5 bg-background border border-border rounded text-sm text-foreground placeholder:text-muted-foreground/50 focus:outline-none focus:border-foreground/30 focus:ring-0';

	const usernameRegex = /^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$/;

	// Channel type config (label + icon only — colors removed for grayscale chrome)
	const channelTypeConfig: Record<AlertChannelType, {
		icon: typeof MessageCircle;
		label: string;
	}> = {
		discord: { icon: MessageCircle, label: 'Discord' },
		slack: { icon: Hash, label: 'Slack' },
		email: { icon: Mail, label: 'Email' },
		telegram: { icon: Send, label: 'Telegram' },
		pagerduty: { icon: PhoneCall, label: 'PagerDuty' },
		webhook: { icon: Webhook, label: 'Webhook' }
	};

	function timeAgo(dateStr: string | null): string {
		if (!dateStr) return 'Never';
		const now = Date.now();
		const then = new Date(dateStr).getTime();
		const diff = Math.floor((now - then) / 1000);
		if (diff < 60) return 'just now';
		if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
		if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
		return `${Math.floor(diff / 86400)}d ago`;
	}

	function formatRange(starts: string, ends: string): string {
		const s = new Date(starts);
		const e = new Date(ends);
		const sameDay = s.toDateString() === e.toDateString();
		const dateOpts: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric' };
		const timeOpts: Intl.DateTimeFormatOptions = { hour: '2-digit', minute: '2-digit' };
		if (sameDay) {
			return `${s.toLocaleDateString(undefined, dateOpts)} ${s.toLocaleTimeString(undefined, timeOpts)}–${e.toLocaleTimeString(undefined, timeOpts)}`;
		}
		return `${s.toLocaleDateString(undefined, dateOpts)} ${s.toLocaleTimeString(undefined, timeOpts)} → ${e.toLocaleDateString(undefined, dateOpts)} ${e.toLocaleTimeString(undefined, timeOpts)}`;
	}

	// Username
	function startEditUsername() {
		usernameDraft = username;
		usernameError = '';
		usernameSuccess = '';
		editingUsername = true;
	}

	function cancelEditUsername() {
		editingUsername = false;
		usernameError = '';
	}

	async function handleUsernameSubmit(e: Event) {
		e.preventDefault();
		usernameSuccess = '';
		usernameError = '';

		const trimmed = usernameDraft.trim();
		if (!trimmed) {
			usernameError = 'Username is required.';
			return;
		}
		if (!usernameRegex.test(trimmed)) {
			usernameError =
				'Must be 3-50 characters. Lowercase letters, numbers, and hyphens only. Cannot start or end with a hyphen.';
			return;
		}

		usernameLoading = true;
		try {
			const res = await settingsApi.updateProfile({ username: trimmed });
			username = res.data.username;
			editingUsername = false;
			usernameSuccess = 'Username updated.';
			setTimeout(() => {
				usernameSuccess = '';
			}, 3000);
		} catch (err) {
			usernameError = err instanceof Error ? err.message : 'Failed to update username.';
		} finally {
			usernameLoading = false;
		}
	}

	// Password
	function startEditPassword() {
		editingPassword = true;
		passwordError = '';
		passwordSuccess = '';
		currentPassword = '';
		newPassword = '';
		confirmNewPassword = '';
	}

	function cancelEditPassword() {
		editingPassword = false;
		currentPassword = '';
		newPassword = '';
		confirmNewPassword = '';
		passwordError = '';
	}

	// Channel actions
	async function handleTestChannel(id: string) {
		testingChannelId = id;
		const updated = { ...channelTestResult };
		delete updated[id];
		channelTestResult = updated;

		try {
			await settingsApi.testChannel(id);
			channelTestResult = {
				...channelTestResult,
				[id]: { ok: true, message: 'Test sent.' }
			};
		} catch (err) {
			channelTestResult = {
				...channelTestResult,
				[id]: {
					ok: false,
					message: err instanceof Error ? err.message : 'Test failed.'
				}
			};
		} finally {
			testingChannelId = null;
			setTimeout(() => {
				const cleaned = { ...channelTestResult };
				delete cleaned[id];
				channelTestResult = cleaned;
			}, 4000);
		}
	}

	async function handleToggleChannel(id: string) {
		togglingChannelId = id;
		try {
			const res = await settingsApi.toggleChannel(id);
			channels = channels.map((c) => (c.id === id ? res.data : c));
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to toggle channel.');
		} finally {
			togglingChannelId = null;
		}
	}

	function handleDeleteChannel(id: string) {
		const channel = channels.find((c) => c.id === id);
		confirmModal = {
			open: true,
			title: 'Delete Channel',
			message: `Are you sure you want to delete "${channel?.name ?? 'this channel'}"? This cannot be undone.`,
			confirmLabel: 'Delete',
			variant: 'danger',
			loading: false,
			action: async () => {
				confirmModal.loading = true;
				try {
					await settingsApi.deleteChannel(id);
					channels = channels.filter((c) => c.id !== id);
					closeConfirmModal();
					toast.success('Channel deleted.');
				} catch (err) {
					toast.error(err instanceof Error ? err.message : 'Failed to delete channel.');
					confirmModal.loading = false;
				}
			}
		};
	}

	// Token actions
	function handleDeleteToken(id: string) {
		const token = tokens.find((t) => t.id === id);
		confirmModal = {
			open: true,
			title: 'Delete Token',
			message: `Are you sure you want to delete "${token?.name ?? 'this token'}"? Any integrations using this token will stop working.`,
			confirmLabel: 'Delete',
			variant: 'danger',
			loading: false,
			action: async () => {
				confirmModal.loading = true;
				try {
					await settingsApi.deleteToken(id);
					tokens = tokens.filter((t) => t.id !== id);
					closeConfirmModal();
					toast.success('Token deleted.');
				} catch (err) {
					toast.error(err instanceof Error ? err.message : 'Failed to delete token.');
					confirmModal.loading = false;
				}
			}
		};
	}

	function handleRegenerateToken(id: string) {
		const token = tokens.find((t) => t.id === id);
		confirmModal = {
			open: true,
			title: 'Regenerate Token',
			message: `Are you sure you want to regenerate "${token?.name ?? 'this token'}"? The current token will be invalidated immediately.`,
			confirmLabel: 'Regenerate',
			variant: 'warning',
			loading: false,
			action: async () => {
				confirmModal.loading = true;
				try {
					const res = await settingsApi.regenerateToken(id);
					tokens = tokens.map((t) => (t.id === id ? res.data : t));
					plaintextToken = res.plaintext;
					plaintextTokenName = res.data.name;
					closeConfirmModal();
					toast.success('Token regenerated.');
				} catch (err) {
					toast.error(err instanceof Error ? err.message : 'Failed to regenerate token.');
					confirmModal.loading = false;
				}
			}
		};
	}

	function closeConfirmModal() {
		confirmModal = {
			open: false,
			title: '',
			message: '',
			confirmLabel: 'Confirm',
			variant: 'danger',
			loading: false,
			action: null
		};
	}

	async function executeConfirm() {
		if (confirmModal.action) await confirmModal.action();
	}

	async function copyTokenToClipboard() {
		await navigator.clipboard.writeText(plaintextToken);
		copiedToken = true;
		setTimeout(() => {
			copiedToken = false;
		}, 2000);
	}

	function dismissPlaintext() {
		plaintextToken = '';
		plaintextTokenName = '';
		copiedToken = false;
	}

	function handleTokenCreated(plaintext: string) {
		loadTokens();
		plaintextToken = plaintext;
		plaintextTokenName = 'New token';
	}

	function handleChannelCreated() {
		loadChannels();
		toast.success('Channel created.');
	}

	// Maintenance window actions
	function handleDeleteMaintenance(id: string) {
		const mw = maintenanceWindows.find((w) => w.id === id);
		confirmModal = {
			open: true,
			title: 'Delete Maintenance Window',
			message: `Are you sure you want to delete "${mw?.name ?? 'this window'}"? This cannot be undone.`,
			confirmLabel: 'Delete',
			variant: 'danger',
			loading: false,
			action: async () => {
				confirmModal.loading = true;
				try {
					await maintenanceApi.deleteWindow(id);
					maintenanceWindows = maintenanceWindows.filter((w) => w.id !== id);
					closeConfirmModal();
					toast.success('Maintenance window deleted.');
				} catch (err) {
					toast.error(
						err instanceof Error ? err.message : 'Failed to delete maintenance window.'
					);
					confirmModal.loading = false;
				}
			}
		};
	}

	function handleMaintenanceCreated() {
		loadMaintenance();
		toast.success('Maintenance window scheduled.');
	}

	async function loadMaintenance() {
		try {
			const res = await maintenanceApi.listWindows();
			maintenanceWindows = res.data ?? [];
		} catch {
			// silent
		}
	}

	async function loadTokens() {
		try {
			const res = await settingsApi.listTokens();
			tokens = res.data ?? [];
		} catch {
			// silent
		}
	}

	async function loadChannels() {
		try {
			const res = await settingsApi.listChannels();
			channels = res.data ?? [];
		} catch {
			// silent
		}
	}

	async function loadData() {
		try {
			const [tokenRes, channelRes, mwRes] = await Promise.all([
				settingsApi.listTokens(),
				settingsApi.listChannels(),
				maintenanceApi.listWindows().catch(() => ({ data: [] as MaintenanceWindow[] }))
			]);
			tokens = tokenRes.data ?? [];
			channels = channelRes.data ?? [];
			maintenanceWindows = mwRes.data ?? [];
		} catch {
			// Keep defaults on error
		} finally {
			loading = false;
		}
	}

	async function handlePasswordSubmit(e: Event) {
		e.preventDefault();
		passwordSuccess = '';
		passwordError = '';

		if (!currentPassword || !newPassword) {
			passwordError = 'All fields are required.';
			return;
		}
		if (newPassword.length < 8) {
			passwordError = 'New password must be at least 8 characters.';
			return;
		}
		if (newPassword !== confirmNewPassword) {
			passwordError = 'New passwords do not match.';
			return;
		}
		if (newPassword === currentPassword) {
			passwordError = 'New password must be different from current password.';
			return;
		}

		passwordLoading = true;
		try {
			await settingsApi.changePassword({
				current_password: currentPassword,
				new_password: newPassword,
				confirm_password: confirmNewPassword
			});
			passwordSuccess = 'Password changed.';
			currentPassword = '';
			newPassword = '';
			confirmNewPassword = '';
			showForceChangeBanner = false;
			editingPassword = false;
			auth.clearMustChangePassword();
			setTimeout(() => {
				passwordSuccess = '';
			}, 5000);
		} catch (err) {
			passwordError = err instanceof Error ? err.message : 'Failed to change password.';
		} finally {
			passwordLoading = false;
		}
	}

	onMount(() => {
		if (auth.user?.username) {
			username = auth.user.username;
		}
		if (forceChange) {
			showForceChangeBanner = true;
			editingPassword = true;
		}
		loadData();
	});
</script>

<svelte:head>
	<title>Settings - WatchDog</title>
</svelte:head>

{#if loading}
	<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-6 sm:px-6 sm:py-10">
		<div class="space-y-2">
			<Skeleton emphasis="secondary" width="8rem" height="2rem" />
			<Skeleton emphasis="tertiary" width="22rem" height="1rem" />
		</div>
		<div class="mt-10 space-y-10">
			{#each Array(4) as _}
				<div class="space-y-3 border-t border-border pt-8">
					<Skeleton emphasis="secondary" width="10rem" height="1.25rem" />
					<Skeleton emphasis="tertiary" width="100%" height="3rem" />
					<Skeleton emphasis="tertiary" width="100%" height="3rem" />
				</div>
			{/each}
		</div>
	</div>
{:else}
	<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-6 sm:px-6 sm:py-12 lg:py-16">
		<!-- Page header -->
		<header class="mb-2 flex flex-col items-start gap-2 sm:flex-row sm:items-baseline sm:justify-between sm:gap-4">
			<div>
				<h1 class="text-xl font-medium tracking-tight text-foreground sm:text-3xl lg:text-[32px]">Settings</h1>
				<p class="mt-1.5 text-sm text-muted-foreground sm:text-base">
					Account, notifications, and API access.
				</p>
			</div>
			{#if auth.user?.email}
				<span class="font-mono tabular-nums text-xs text-muted-foreground sm:text-sm">{auth.user.email}</span>
			{/if}
		</header>

		<!-- Plaintext token banner (shown after create or regenerate) -->
		{#if plaintextToken}
			<div class="mt-8 border border-warning/40 bg-warning/[0.04] p-4">
				<div class="mb-2 flex items-start justify-between gap-3">
					<div>
						<div class="text-sm font-medium text-foreground">
							{plaintextTokenName} created
						</div>
						<p class="mt-0.5 text-xs text-muted-foreground">
							Copy this token now. You won't be able to see it again.
						</p>
					</div>
					<button
						onclick={dismissPlaintext}
						class="text-muted-foreground transition-colors hover:text-foreground"
						aria-label="Dismiss"
					>
						<X class="h-4 w-4" />
					</button>
				</div>
				<div class="flex items-center gap-2">
					<code
						class="flex-1 select-all break-all border border-border bg-background px-3 py-2 font-mono text-xs text-foreground"
					>
						{plaintextToken}
					</code>
					<button
						onclick={copyTokenToClipboard}
						class="flex shrink-0 items-center gap-1.5 border border-border px-3 py-2 text-xs font-medium text-foreground transition-colors hover:bg-muted/40"
						aria-label="Copy token"
					>
						{#if copiedToken}
							<Check class="h-3.5 w-3.5 text-success" />
							<span>Copied</span>
						{:else}
							<Copy class="h-3.5 w-3.5" />
							<span>Copy</span>
						{/if}
					</button>
				</div>
			</div>
		{/if}

		<!-- ==================== ACCOUNT ==================== -->
		<section class="mt-8 border-t border-border pt-8">
			<div class="mb-6 flex flex-col items-start gap-3 sm:mb-8 sm:flex-row sm:items-baseline sm:justify-between sm:gap-4">
				<div>
					<h2 class="text-xl font-medium text-foreground sm:text-2xl">Account</h2>
					<p class="mt-1 text-sm text-muted-foreground">
						Identity and credentials for this workspace.
					</p>
				</div>
			</div>

			<div class="divide-y divide-border">
				<!-- Username row -->
				<div class="py-4">
					{#if !editingUsername}
						<div class="flex items-center justify-between gap-4">
							<div class="min-w-0">
								<div class="text-sm text-muted-foreground">Username</div>
								<div class="mt-1 flex items-baseline gap-2">
									<span
										class="font-mono tabular-nums text-sm text-muted-foreground/60"
									>usewatchdog.dev/status/@</span>
									<span class="font-mono tabular-nums text-sm text-foreground">{username || '—'}</span>
								</div>
							</div>
							<button
								onclick={startEditUsername}
								class="shrink-0 text-sm text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline"
							>
								Edit
							</button>
						</div>
						{#if usernameSuccess}
							<p class="mt-2 font-mono tabular-nums text-xs text-success">
								<span aria-hidden="true">●</span> {usernameSuccess}
							</p>
						{/if}
					{:else}
						<form onsubmit={handleUsernameSubmit} class="space-y-3">
							<div class="text-sm text-muted-foreground">Username</div>
							<div class="flex items-center gap-2">
								<span
									class="hidden font-mono tabular-nums text-sm text-muted-foreground/60 sm:inline"
								>usewatchdog.dev/status/@</span>
								<input
									id="settings-username"
									type="text"
									bind:value={usernameDraft}
									placeholder="my-username"
									class="{inlineInputClass} flex-1 font-mono tabular-nums"
								/>
								<button
									type="submit"
									disabled={usernameLoading}
									class="shrink-0 bg-accent px-3 py-1.5 text-sm font-medium text-background transition-opacity hover:opacity-90 disabled:opacity-50"
								>
									{usernameLoading ? 'Saving…' : 'Save'}
								</button>
								<button
									type="button"
									onclick={cancelEditUsername}
									disabled={usernameLoading}
									class="shrink-0 px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground disabled:opacity-50"
								>
									Cancel
								</button>
							</div>
							<p class="text-xs text-muted-foreground">
								3–50 characters. Lowercase letters, numbers, and hyphens only.
							</p>
							{#if usernameError}
								<Alert tone="down">
									{#snippet icon()}<AlertCircle class="h-3.5 w-3.5" />{/snippet}
									{usernameError}
								</Alert>
							{/if}
						</form>
					{/if}
				</div>

				<!-- Password row -->
				<div class="py-4">
					{#if showForceChangeBanner}
						<div class="mb-4">
							<Alert tone="warn">
								{#snippet icon()}<AlertTriangle class="h-3.5 w-3.5" />{/snippet}
								Your password was reset by an administrator. Please set a new password.
							</Alert>
						</div>
					{/if}

					{#if !editingPassword}
						<div class="flex items-center justify-between gap-4">
							<div class="min-w-0">
								<div class="text-sm text-muted-foreground">Password</div>
								<div class="mt-1 font-mono tabular-nums text-sm text-foreground">
									••••••••••••
								</div>
							</div>
							<button
								onclick={startEditPassword}
								class="shrink-0 text-sm text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline"
							>
								Change
							</button>
						</div>
						{#if passwordSuccess}
							<p class="mt-2 font-mono tabular-nums text-xs text-success">
								<span aria-hidden="true">●</span> {passwordSuccess}
							</p>
						{/if}
					{:else}
						<form onsubmit={handlePasswordSubmit} class="space-y-3">
							<div class="text-sm text-muted-foreground">Change password</div>
							<div class="grid gap-2 sm:max-w-sm">
								<input
									type="password"
									bind:value={currentPassword}
									autocomplete="current-password"
									placeholder="Current password"
									class={inlineInputClass}
								/>
								<input
									type="password"
									bind:value={newPassword}
									autocomplete="new-password"
									placeholder="New password (min 8 characters)"
									class={inlineInputClass}
								/>
								<input
									type="password"
									bind:value={confirmNewPassword}
									autocomplete="new-password"
									placeholder="Confirm new password"
									class={inlineInputClass}
								/>
							</div>

							{#if passwordError}
								<Alert tone="down">
									{#snippet icon()}<AlertCircle class="h-3.5 w-3.5" />{/snippet}
									{passwordError}
								</Alert>
							{/if}

							<div class="flex items-center gap-2">
								<button
									type="submit"
									disabled={passwordLoading}
									class="bg-accent px-3 py-1.5 text-sm font-medium text-background transition-opacity hover:opacity-90 disabled:opacity-50"
								>
									{passwordLoading ? 'Changing…' : 'Change password'}
								</button>
								<button
									type="button"
									onclick={cancelEditPassword}
									disabled={passwordLoading}
									class="px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground disabled:opacity-50"
								>
									Cancel
								</button>
							</div>
						</form>
					{/if}
				</div>
			</div>
		</section>

		<!-- ==================== NOTIFICATIONS ==================== -->
		<section class="mt-8 border-t border-border pt-8">
			<div class="mb-6 flex flex-col items-start gap-3 sm:mb-8 sm:flex-row sm:items-baseline sm:justify-between sm:gap-4">
				<div>
					<h2 class="flex items-baseline gap-2 text-xl font-medium text-foreground sm:text-2xl">
						<span>Notifications</span>
						{#if channels.length > 0}
							<span class="font-mono tabular-nums text-xs text-muted-foreground">
								{channels.length}
							</span>
						{/if}
					</h2>
					<p class="mt-1 text-sm text-muted-foreground">
						Where incidents are delivered when monitors fail or recover.
					</p>
				</div>
				<button
					onclick={() => {
						showChannelModal = true;
					}}
					class="inline-flex shrink-0 items-center gap-1.5 bg-accent px-3 py-1.5 text-sm font-medium text-background transition-opacity hover:opacity-90"
				>
					<Plus class="h-3.5 w-3.5" strokeWidth={2.5} />
					<span>Add channel</span>
				</button>
			</div>

			{#if channels.length === 0}
				<div class="border border-dashed border-border px-6 py-10 text-center">
					<p class="text-sm text-foreground">No alert channels configured.</p>
					<p class="mt-1 text-xs text-muted-foreground">
						Add Discord, Slack, Email, Telegram, PagerDuty, or a webhook to receive incidents.
					</p>
				</div>
			{:else}
				<div class="divide-y divide-border border-y border-border">
					{#each channels as channel (channel.id)}
						{@const typeConf = channelTypeConfig[channel.type]}
						{@const ChannelIcon = typeConf.icon}
						{@const result = channelTestResult[channel.id]}
						<div class="group flex flex-wrap items-center gap-x-4 gap-y-2 px-2 py-3 transition-colors hover:bg-muted/[0.15]">
							<!-- Status pip -->
							<span
								class="inline-block h-1.5 w-1.5 shrink-0 rounded-full {channel.enabled
									? 'bg-success'
									: 'bg-muted-foreground/40'}"
								aria-label={channel.enabled ? 'Active' : 'Paused'}
							></span>

							<!-- Type icon -->
							<ChannelIcon class="h-4 w-4 shrink-0 text-muted-foreground" />

							<!-- Name + type -->
							<div class="flex min-w-0 flex-1 items-baseline gap-3">
								<span class="truncate text-sm text-foreground">{channel.name}</span>
								<span class="font-mono tabular-nums text-xs text-muted-foreground">
									{typeConf.label.toLowerCase()}
								</span>
							</div>

							<!-- Test result inline -->
							{#if result}
								<span
									class="font-mono tabular-nums text-xs {result.ok
										? 'text-success'
										: 'text-destructive'}"
								>
									<span aria-hidden="true">●</span> {result.message}
								</span>
							{/if}

							<!-- Actions -->
							<div class="ml-auto flex shrink-0 items-center gap-3">
								<button
									onclick={() => handleTestChannel(channel.id)}
									disabled={testingChannelId === channel.id}
									class="text-sm text-muted-foreground transition-colors hover:text-foreground disabled:opacity-50"
								>
									{#if testingChannelId === channel.id}
										<Loader2 class="h-3.5 w-3.5 animate-spin" />
									{:else}
										Test
									{/if}
								</button>
								<button
									onclick={() => handleToggleChannel(channel.id)}
									disabled={togglingChannelId === channel.id}
									class="text-sm text-muted-foreground transition-colors hover:text-foreground disabled:opacity-50"
								>
									{channel.enabled ? 'Disable' : 'Enable'}
								</button>
								<button
									onclick={() => handleDeleteChannel(channel.id)}
									class="text-sm text-muted-foreground transition-colors hover:text-destructive"
								>
									Delete
								</button>
							</div>
						</div>
					{/each}
				</div>
			{/if}

			<!-- Supported integrations footer -->
			<div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground/70">
				<span>Supported:</span>
				{#each Object.entries(channelTypeConfig) as [key, conf]}
					{@const IntegrationIcon = conf.icon}
					<span class="inline-flex items-center gap-1">
						<IntegrationIcon class="h-3 w-3" />
						<span>{conf.label}</span>
					</span>
				{/each}
			</div>
		</section>

		<!-- ==================== MAINTENANCE WINDOWS ==================== -->
		<section class="mt-8 border-t border-border pt-8">
			<div class="mb-6 flex flex-col items-start gap-3 sm:mb-8 sm:flex-row sm:items-baseline sm:justify-between sm:gap-4">
				<div>
					<h2 class="flex items-baseline gap-2 text-xl font-medium text-foreground sm:text-2xl">
						<span>Maintenance windows</span>
						{#if maintenanceWindows.length > 0}
							<span class="font-mono tabular-nums text-xs text-muted-foreground">
								{maintenanceWindows.length}
							</span>
						{/if}
					</h2>
					<p class="mt-1 text-sm text-muted-foreground">
						Suppress alerts during planned downtime.
					</p>
				</div>
				<button
					onclick={() => {
						showMaintenanceModal = true;
					}}
					class="inline-flex shrink-0 items-center gap-1.5 bg-accent px-3 py-1.5 text-sm font-medium text-background transition-opacity hover:opacity-90"
				>
					<Plus class="h-3.5 w-3.5" strokeWidth={2.5} />
					<span>Schedule</span>
				</button>
			</div>

			{#if maintenanceWindows.length === 0}
				<div class="border border-dashed border-border px-6 py-10 text-center">
					<p class="text-sm text-foreground">No maintenance windows scheduled.</p>
					<p class="mt-1 text-xs text-muted-foreground">
						Suppress alerts ahead of planned downtime, deploys, or upgrades.
					</p>
				</div>
			{:else}
				<div class="divide-y divide-border border-y border-border">
					{#each maintenanceWindows as mw (mw.id)}
						<div class="group flex flex-wrap items-center gap-x-4 gap-y-2 px-2 py-3 transition-colors hover:bg-muted/[0.15]">
							<!-- Status pip -->
							<span
								class="inline-block h-1.5 w-1.5 shrink-0 rounded-full {mw.status === 'active'
									? 'bg-success'
									: mw.status === 'expired'
										? 'bg-muted-foreground/40'
										: 'bg-warning'}"
								aria-label={mw.status}
							></span>

							<!-- Name + meta -->
							<div class="flex min-w-0 flex-1 flex-col gap-0.5">
								<div class="flex items-baseline gap-3">
									<span class="truncate text-sm text-foreground">{mw.name}</span>
									<span class="font-mono tabular-nums text-xs text-muted-foreground">
										{mw.status}
									</span>
									{#if mw.recurrence && mw.recurrence !== 'once'}
										<span class="font-mono tabular-nums text-xs text-muted-foreground/70">
											↻ {mw.recurrence}
										</span>
									{/if}
								</div>
								<div class="font-mono tabular-nums text-xs text-muted-foreground">
									<span>{mw.agent_name}</span>
									<span class="px-1.5 text-muted-foreground/40">·</span>
									<span>{formatRange(mw.starts_at, mw.ends_at)}</span>
								</div>
							</div>

							<!-- Actions -->
							<button
								onclick={() => handleDeleteMaintenance(mw.id)}
								class="ml-auto shrink-0 text-sm text-muted-foreground transition-colors hover:text-destructive"
							>
								Delete
							</button>
						</div>
					{/each}
				</div>
			{/if}
		</section>

		<!-- ==================== API TOKENS ==================== -->
		<section class="mt-8 border-t border-border pt-8">
			<div class="mb-6 flex flex-col items-start gap-3 sm:mb-8 sm:flex-row sm:items-baseline sm:justify-between sm:gap-4">
				<div>
					<h2 class="flex items-baseline gap-2 text-xl font-medium text-foreground sm:text-2xl">
						<span>API tokens</span>
						{#if tokens.length > 0}
							<span class="font-mono tabular-nums text-xs text-muted-foreground">
								{tokens.length}
							</span>
						{/if}
					</h2>
					<p class="mt-1 text-sm text-muted-foreground">
						Programmatic access to the WatchDog API for CI, Grafana, or custom scripts.
					</p>
				</div>
				<button
					onclick={() => {
						showTokenModal = true;
					}}
					class="inline-flex shrink-0 items-center gap-1.5 bg-accent px-3 py-1.5 text-sm font-medium text-background transition-opacity hover:opacity-90"
				>
					<Plus class="h-3.5 w-3.5" strokeWidth={2.5} />
					<span>New token</span>
				</button>
			</div>

			{#if tokens.length === 0}
				<div class="border border-dashed border-border px-6 py-10 text-center">
					<p class="text-sm text-foreground">No API tokens issued.</p>
					<p class="mt-1 text-xs text-muted-foreground">
						Create a token to integrate with CI/CD pipelines, Grafana, or custom scripts.
					</p>
				</div>
			{:else}
				<div class="divide-y divide-border border-y border-border">
					{#each tokens as token (token.id)}
						<div class="group flex flex-wrap items-center gap-x-4 gap-y-2 px-2 py-3 transition-colors hover:bg-muted/[0.15]">
							<!-- Scope pip -->
							<span
								class="inline-block h-1.5 w-1.5 shrink-0 rounded-full {token.scope === 'admin'
									? 'bg-warning'
									: token.scope === 'telemetry_ingest'
										? 'bg-success'
										: 'bg-muted-foreground/40'}"
								aria-label={token.scope}
							></span>

							<!-- Name + prefix + meta -->
							<div class="flex min-w-0 flex-1 flex-col gap-0.5">
								<div class="flex items-baseline gap-3">
									<span class="truncate text-sm text-foreground">{token.name}</span>
									<code class="font-mono tabular-nums text-xs text-muted-foreground">
										{token.prefix}…
									</code>
									<span class="font-mono tabular-nums text-xs text-muted-foreground">
										{token.scope === 'telemetry_ingest' ? 'telemetry' : token.scope}
									</span>
								</div>
								<div class="flex flex-wrap items-baseline gap-x-3 font-mono tabular-nums text-xs text-muted-foreground">
									<span>created {timeAgo(token.created_at)}</span>
									{#if token.expires_at}
										<span class="text-warning/80">expires {timeAgo(token.expires_at)}</span>
									{:else}
										<span class="text-muted-foreground/60">no expiry</span>
									{/if}
									{#if token.last_used_at}
										<span class="hidden sm:inline">last used {timeAgo(token.last_used_at)}</span>
										{#if token.last_used_ip}
											<span class="hidden text-muted-foreground/60 sm:inline">
												from {token.last_used_ip}
											</span>
										{/if}
									{:else}
										<span class="hidden text-muted-foreground/60 sm:inline">never used</span>
									{/if}
								</div>
							</div>

							<!-- Actions -->
							<div class="ml-auto flex shrink-0 items-center gap-3">
								<button
									onclick={() => handleRegenerateToken(token.id)}
									class="text-sm text-muted-foreground transition-colors hover:text-foreground"
								>
									Regenerate
								</button>
								<button
									onclick={() => handleDeleteToken(token.id)}
									class="text-sm text-muted-foreground transition-colors hover:text-destructive"
								>
									Delete
								</button>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</section>
	</div>

	<CreateChannelModal
		bind:open={showChannelModal}
		onClose={() => {
			showChannelModal = false;
		}}
		onCreated={handleChannelCreated}
	/>

	<CreateTokenModal
		bind:open={showTokenModal}
		onClose={() => {
			showTokenModal = false;
			plaintextToken = '';
		}}
		onCreated={handleTokenCreated}
	/>

	<CreateMaintenanceModal
		bind:open={showMaintenanceModal}
		onClose={() => {
			showMaintenanceModal = false;
		}}
		onCreated={handleMaintenanceCreated}
	/>

	<ConfirmModal
		open={confirmModal.open}
		title={confirmModal.title}
		message={confirmModal.message}
		confirmLabel={confirmModal.confirmLabel}
		variant={confirmModal.variant}
		loading={confirmModal.loading}
		onConfirm={executeConfirm}
		onCancel={closeConfirmModal}
	/>
{/if}
