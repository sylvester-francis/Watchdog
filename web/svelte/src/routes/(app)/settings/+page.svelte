<script lang="ts">
	import { onMount } from 'svelte';
	import {
		AtSign,
		Bell,
		BellOff,
		Key,
		Plus,
		Trash2,
		RefreshCw,
		Copy,
		Check,
		Loader2,
		AlertCircle,
		MessageCircle,
		Hash,
		Mail,
		Send,
		PhoneCall,
		Webhook
	} from 'lucide-svelte';
	import { settings as settingsApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast.svelte';
	import { getAuth } from '$lib/stores/auth.svelte';
	import type { APIToken, AlertChannel, AlertChannelType } from '$lib/types';
	import CreateChannelModal from '$lib/components/settings/CreateChannelModal.svelte';
	import CreateTokenModal from '$lib/components/settings/CreateTokenModal.svelte';

	const toast = getToasts();
	const auth = getAuth();

	// Data
	let tokens = $state<APIToken[]>([]);
	let channels = $state<AlertChannel[]>([]);
	let loading = $state(true);

	// Username form
	let username = $state('');
	let usernameLoading = $state(false);
	let usernameSuccess = $state('');
	let usernameError = $state('');

	// Modals
	let showChannelModal = $state(false);
	let showTokenModal = $state(false);

	// Token plaintext display (after create or regenerate)
	let plaintextToken = $state('');
	let plaintextTokenName = $state('');
	let copiedToken = $state(false);

	// Per-channel action states
	let testingChannelId = $state<string | null>(null);
	let channelTestResult = $state<Record<string, { ok: boolean; message: string }>>({});
	let togglingChannelId = $state<string | null>(null);
	let confirmDeleteChannelId = $state<string | null>(null);
	let deletingChannelId = $state<string | null>(null);

	// Per-token action states
	let confirmDeleteTokenId = $state<string | null>(null);
	let deletingTokenId = $state<string | null>(null);
	let confirmRegenTokenId = $state<string | null>(null);
	let regenTokenId = $state<string | null>(null);

	const inputClass = 'w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background';
	const labelClass = 'block text-xs font-medium text-muted-foreground mb-1.5';

	const usernameRegex = /^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$/;

	// Channel type config
	const channelTypeConfig: Record<AlertChannelType, {
		icon: typeof MessageCircle;
		iconBg: string;
		iconColor: string;
		badgeBg: string;
		badgeText: string;
		label: string;
	}> = {
		discord: { icon: MessageCircle, iconBg: 'bg-indigo-500/10', iconColor: 'text-indigo-400', badgeBg: 'bg-indigo-500/15', badgeText: 'text-indigo-400', label: 'Discord' },
		slack: { icon: Hash, iconBg: 'bg-green-500/10', iconColor: 'text-green-400', badgeBg: 'bg-green-500/15', badgeText: 'text-green-400', label: 'Slack' },
		email: { icon: Mail, iconBg: 'bg-blue-500/10', iconColor: 'text-blue-400', badgeBg: 'bg-blue-500/15', badgeText: 'text-blue-400', label: 'Email' },
		telegram: { icon: Send, iconBg: 'bg-cyan-500/10', iconColor: 'text-cyan-400', badgeBg: 'bg-cyan-500/15', badgeText: 'text-cyan-400', label: 'Telegram' },
		pagerduty: { icon: PhoneCall, iconBg: 'bg-emerald-500/10', iconColor: 'text-emerald-400', badgeBg: 'bg-emerald-500/15', badgeText: 'text-emerald-400', label: 'PagerDuty' },
		webhook: { icon: Webhook, iconBg: 'bg-orange-500/10', iconColor: 'text-orange-400', badgeBg: 'bg-orange-500/15', badgeText: 'text-orange-400', label: 'Webhook' }
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

	// Username
	async function handleUsernameSubmit(e: Event) {
		e.preventDefault();
		usernameSuccess = '';
		usernameError = '';

		const trimmed = username.trim();
		if (!trimmed) {
			usernameError = 'Username is required.';
			return;
		}
		if (!usernameRegex.test(trimmed)) {
			usernameError = 'Must be 3-50 characters. Lowercase letters, numbers, and hyphens only. Cannot start or end with a hyphen.';
			return;
		}

		usernameLoading = true;
		try {
			const res = await settingsApi.updateProfile({ username: trimmed });
			username = res.data.username;
			usernameSuccess = 'Username updated successfully.';
			setTimeout(() => { usernameSuccess = ''; }, 3000);
		} catch (err) {
			usernameError = err instanceof Error ? err.message : 'Failed to update username.';
		} finally {
			usernameLoading = false;
		}
	}

	// Channel actions
	async function handleTestChannel(id: string) {
		testingChannelId = id;
		// Clear previous result for this channel
		const updated = { ...channelTestResult };
		delete updated[id];
		channelTestResult = updated;

		try {
			await settingsApi.testChannel(id);
			channelTestResult = { ...channelTestResult, [id]: { ok: true, message: 'Test notification sent.' } };
		} catch (err) {
			channelTestResult = { ...channelTestResult, [id]: { ok: false, message: err instanceof Error ? err.message : 'Test failed.' } };
		} finally {
			testingChannelId = null;
			// Clear feedback after 4s
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
			channels = channels.map((c) => c.id === id ? res.data : c);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to toggle channel.');
		} finally {
			togglingChannelId = null;
		}
	}

	async function handleDeleteChannel(id: string) {
		if (confirmDeleteChannelId !== id) {
			confirmDeleteChannelId = id;
			return;
		}
		deletingChannelId = id;
		try {
			await settingsApi.deleteChannel(id);
			channels = channels.filter((c) => c.id !== id);
			confirmDeleteChannelId = null;
			toast.success('Channel deleted.');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to delete channel.');
		} finally {
			deletingChannelId = null;
		}
	}

	// Token actions
	async function handleDeleteToken(id: string) {
		if (confirmDeleteTokenId !== id) {
			confirmDeleteTokenId = id;
			return;
		}
		deletingTokenId = id;
		try {
			await settingsApi.deleteToken(id);
			tokens = tokens.filter((t) => t.id !== id);
			confirmDeleteTokenId = null;
			toast.success('Token deleted.');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to delete token.');
		} finally {
			deletingTokenId = null;
		}
	}

	async function handleRegenerateToken(id: string) {
		if (confirmRegenTokenId !== id) {
			confirmRegenTokenId = id;
			return;
		}
		regenTokenId = id;
		try {
			const res = await settingsApi.regenerateToken(id);
			tokens = tokens.map((t) => t.id === id ? res.data : t);
			plaintextToken = res.plaintext;
			plaintextTokenName = res.data.name;
			confirmRegenTokenId = null;
			toast.success('Token regenerated.');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to regenerate token.');
		} finally {
			regenTokenId = null;
		}
	}

	async function copyTokenToClipboard() {
		await navigator.clipboard.writeText(plaintextToken);
		copiedToken = true;
		setTimeout(() => { copiedToken = false; }, 2000);
	}

	function dismissPlaintext() {
		plaintextToken = '';
		plaintextTokenName = '';
		copiedToken = false;
	}

	function handleTokenCreated(plaintext: string) {
		// Reload tokens and show plaintext
		loadTokens();
		plaintextToken = plaintext;
		plaintextTokenName = 'New token';
	}

	function handleChannelCreated() {
		loadChannels();
		toast.success('Channel created.');
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
			const [tokenRes, channelRes] = await Promise.all([
				settingsApi.listTokens(),
				settingsApi.listChannels()
			]);
			tokens = tokenRes.data ?? [];
			channels = channelRes.data ?? [];
		} catch {
			// Keep defaults on error
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		// Prefill username from auth store
		if (auth.user?.username) {
			username = auth.user.username;
		}
		loadData();
	});
</script>

<svelte:head>
	<title>Settings - WatchDog</title>
</svelte:head>

{#if loading}
	<!-- Skeleton loading state -->
	<div class="animate-fade-in-up space-y-6">
		<!-- Header skeleton -->
		<div>
			<div class="h-7 w-28 bg-muted/50 rounded animate-pulse"></div>
			<div class="h-4 w-72 bg-muted/30 rounded animate-pulse mt-1.5"></div>
		</div>
		<!-- Card skeletons -->
		{#each Array(3) as _}
			<div class="bg-card border border-border rounded-lg p-5">
				<div class="flex items-center space-x-3 mb-4">
					<div class="w-9 h-9 bg-muted/50 rounded-lg animate-pulse"></div>
					<div class="space-y-1.5">
						<div class="h-4 w-32 bg-muted/50 rounded animate-pulse"></div>
						<div class="h-3 w-56 bg-muted/30 rounded animate-pulse"></div>
					</div>
				</div>
				<div class="h-10 w-full bg-muted/30 rounded-md animate-pulse"></div>
			</div>
		{/each}
	</div>
{:else}
	<div class="animate-fade-in-up space-y-6">
		<!-- Page header -->
		<div>
			<h1 class="text-lg font-semibold text-foreground">Settings</h1>
			<p class="text-xs text-muted-foreground mt-0.5">Manage your account, notifications, and API access.</p>
		</div>

		<!-- Plaintext token banner (shown after create or regenerate) -->
		{#if plaintextToken}
			<div class="bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-4">
				<div class="flex items-start justify-between mb-2">
					<div class="flex items-center space-x-2">
						<Key class="w-4 h-4 text-yellow-400" />
						<span class="text-sm font-medium text-foreground">Token Created</span>
					</div>
					<button
						onclick={dismissPlaintext}
						class="text-muted-foreground hover:text-foreground transition-colors text-xs"
					>
						Dismiss
					</button>
				</div>
				<p class="text-xs text-muted-foreground mb-2">Copy this token now. You won't be able to see it again.</p>
				<div class="flex items-center space-x-2">
					<code class="flex-1 text-xs font-mono bg-card border border-border rounded px-3 py-2 text-foreground break-all select-all">{plaintextToken}</code>
					<button
						onclick={copyTokenToClipboard}
						class="p-2 rounded-md hover:bg-muted/50 text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
						aria-label="Copy token"
					>
						{#if copiedToken}
							<Check class="w-4 h-4 text-emerald-400" />
						{:else}
							<Copy class="w-4 h-4" />
						{/if}
					</button>
				</div>
			</div>
		{/if}

		<!-- ==================== USERNAME SECTION ==================== -->
		<div class="bg-card border border-border rounded-lg">
			<div class="p-5">
				<div class="flex items-center space-x-3 mb-4">
					<div class="w-9 h-9 bg-blue-500/10 rounded-lg flex items-center justify-center">
						<AtSign class="w-4.5 h-4.5 text-blue-400" />
					</div>
					<div>
						<h2 class="text-sm font-medium text-foreground">Username</h2>
						<p class="text-xs text-muted-foreground">Your public identifier for status page URLs.</p>
					</div>
				</div>

				<form onsubmit={handleUsernameSubmit} class="space-y-3">
					<div>
						<label for="settings-username" class={labelClass}>Username</label>
						<div class="flex items-center">
							<span class="px-3 py-2 bg-muted/50 border border-border border-r-0 rounded-l-md text-xs text-muted-foreground font-mono whitespace-nowrap">usewatchdog.dev/status/@</span>
							<input
								id="settings-username"
								type="text"
								bind:value={username}
								placeholder="your-name"
								class="flex-1 px-3 py-2 bg-card-elevated border border-border rounded-r-md text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background"
							/>
						</div>
						<p class="text-[10px] text-muted-foreground/70 mt-1.5">3-50 characters. Lowercase letters, numbers, and hyphens only.</p>
					</div>

					{#if usernameError}
						<div class="bg-destructive/10 border border-destructive/20 rounded-md px-3 py-2 flex items-center space-x-2" role="alert">
							<AlertCircle class="w-3.5 h-3.5 text-destructive flex-shrink-0" />
							<span class="text-xs text-destructive">{usernameError}</span>
						</div>
					{/if}

					{#if usernameSuccess}
						<div class="bg-emerald-500/10 border border-emerald-500/20 rounded-md px-3 py-2 flex items-center space-x-2">
							<Check class="w-3.5 h-3.5 text-emerald-400 flex-shrink-0" />
							<span class="text-xs text-emerald-400">{usernameSuccess}</span>
						</div>
					{/if}

					<div class="flex justify-end">
						<button
							type="submit"
							disabled={usernameLoading}
							class="px-4 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors disabled:opacity-50"
						>
							{usernameLoading ? 'Saving...' : 'Save'}
						</button>
					</div>
				</form>
			</div>
		</div>

		<!-- ==================== ALERT CHANNELS SECTION ==================== -->
		<div class="bg-card border border-border rounded-lg">
			<!-- Header -->
			<div class="px-5 py-3.5 border-b border-border flex items-center justify-between">
				<div class="flex items-center space-x-3">
					<div class="w-9 h-9 bg-accent/10 rounded-lg flex items-center justify-center">
						<Bell class="w-4.5 h-4.5 text-accent" />
					</div>
					<div>
						<div class="flex items-center space-x-2">
							<h2 class="text-sm font-medium text-foreground">Alert Channels</h2>
							{#if channels.length > 0}
								<span class="text-[10px] font-mono text-muted-foreground bg-muted/50 px-1.5 py-0.5 rounded">{channels.length}</span>
							{/if}
						</div>
						<p class="text-xs text-muted-foreground">Where notifications are sent when incidents occur.</p>
					</div>
				</div>
				<button
					onclick={() => { showChannelModal = true; }}
					class="flex items-center space-x-1.5 px-3 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors"
				>
					<Plus class="w-3.5 h-3.5" />
					<span>Add Channel</span>
				</button>
			</div>

			<!-- Channel list -->
			{#if channels.length === 0}
				<div class="p-8 text-center">
					<div class="w-12 h-12 bg-muted/50 rounded-lg flex items-center justify-center mx-auto mb-4">
						<BellOff class="w-6 h-6 text-muted-foreground/40" />
					</div>
					<p class="text-sm font-medium text-foreground mb-1">No alert channels</p>
					<p class="text-xs text-muted-foreground mb-4">
						Add a channel to receive notifications when your monitors go down.
					</p>
					<button
						onclick={() => { showChannelModal = true; }}
						class="inline-flex items-center space-x-1.5 px-4 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors"
					>
						<Plus class="w-3.5 h-3.5" />
						<span>Add Channel</span>
					</button>
				</div>
			{:else}
				<div class="divide-y divide-border/20">
					{#each channels as channel (channel.id)}
						{@const typeConf = channelTypeConfig[channel.type]}
						<div class="px-5 py-3.5 flex items-center justify-between gap-3">
							<div class="flex items-center space-x-3 min-w-0">
								<div class="w-8 h-8 {typeConf.iconBg} rounded-lg flex items-center justify-center flex-shrink-0">
									<svelte:component this={typeConf.icon} class="w-4 h-4 {typeConf.iconColor}" />
								</div>
								<div class="min-w-0">
									<div class="flex items-center space-x-2">
										<span class="text-sm text-foreground truncate">{channel.name}</span>
										<span class="text-[9px] font-medium uppercase px-1.5 py-0.5 rounded {typeConf.badgeBg} {typeConf.badgeText}">{typeConf.label}</span>
									</div>
									<div class="flex items-center space-x-2 mt-0.5">
										{#if channel.enabled}
											<span class="flex items-center space-x-1">
												<span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
												<span class="text-[10px] text-emerald-400">Active</span>
											</span>
										{:else}
											<span class="flex items-center space-x-1">
												<span class="w-1.5 h-1.5 rounded-full bg-muted-foreground/50"></span>
												<span class="text-[10px] text-muted-foreground">Paused</span>
											</span>
										{/if}
									</div>
								</div>
							</div>

							<div class="flex items-center space-x-1.5 flex-shrink-0">
								<!-- Test feedback -->
								{#if channelTestResult[channel.id]}
									{@const result = channelTestResult[channel.id]}
									<span class="text-[10px] {result.ok ? 'text-emerald-400' : 'text-red-400'} mr-1">{result.message}</span>
								{/if}

								<!-- Test button -->
								<button
									onclick={() => handleTestChannel(channel.id)}
									disabled={testingChannelId === channel.id}
									class="px-2.5 py-1.5 text-[10px] font-medium text-muted-foreground hover:text-foreground bg-muted/50 hover:bg-muted rounded-md transition-colors disabled:opacity-50 flex items-center space-x-1"
								>
									{#if testingChannelId === channel.id}
										<Loader2 class="w-3 h-3 animate-spin" />
										<span>Testing</span>
									{:else}
										<span>Test</span>
									{/if}
								</button>

								<!-- Toggle button -->
								<button
									onclick={() => handleToggleChannel(channel.id)}
									disabled={togglingChannelId === channel.id}
									class="px-2.5 py-1.5 text-[10px] font-medium rounded-md transition-colors disabled:opacity-50 {channel.enabled
										? 'text-muted-foreground hover:text-foreground bg-muted/50 hover:bg-muted'
										: 'text-emerald-400 hover:text-emerald-300 bg-emerald-500/10 hover:bg-emerald-500/15'}"
								>
									{channel.enabled ? 'Disable' : 'Enable'}
								</button>

								<!-- Delete button -->
								{#if confirmDeleteChannelId === channel.id}
									<button
										onclick={() => handleDeleteChannel(channel.id)}
										disabled={deletingChannelId === channel.id}
										class="px-2.5 py-1.5 text-[10px] font-medium bg-red-500/20 text-red-400 hover:bg-red-500/30 rounded-md transition-colors disabled:opacity-50"
									>
										{deletingChannelId === channel.id ? 'Deleting...' : 'Confirm'}
									</button>
									<button
										onclick={() => { confirmDeleteChannelId = null; }}
										class="px-2 py-1.5 text-[10px] text-muted-foreground hover:text-foreground transition-colors"
									>
										Cancel
									</button>
								{:else}
									<button
										onclick={() => handleDeleteChannel(channel.id)}
										class="p-1.5 text-muted-foreground/40 hover:text-red-400 rounded-md hover:bg-red-500/10 transition-colors"
										aria-label="Delete channel"
									>
										<Trash2 class="w-3.5 h-3.5" />
									</button>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			{/if}

			<!-- Footer: integrations row -->
			<div class="px-5 py-3 border-t border-border/50">
				<div class="flex items-center flex-wrap gap-2">
					<span class="text-[10px] text-muted-foreground/70 font-medium">Integrations:</span>
					{#each Object.entries(channelTypeConfig) as [key, conf]}
						<span class="text-[9px] font-medium uppercase px-1.5 py-0.5 rounded {conf.badgeBg} {conf.badgeText}">{conf.label}</span>
					{/each}
				</div>
			</div>
		</div>

		<!-- ==================== API TOKENS SECTION ==================== -->
		<div class="bg-card border border-border rounded-lg">
			<!-- Header -->
			<div class="px-5 py-3.5 border-b border-border flex items-center justify-between">
				<div class="flex items-center space-x-3">
					<div class="w-9 h-9 bg-yellow-500/10 rounded-lg flex items-center justify-center">
						<Key class="w-4.5 h-4.5 text-yellow-400" />
					</div>
					<div>
						<div class="flex items-center space-x-2">
							<h2 class="text-sm font-medium text-foreground">API Tokens</h2>
							{#if tokens.length > 0}
								<span class="text-[10px] font-mono text-muted-foreground bg-muted/50 px-1.5 py-0.5 rounded">{tokens.length}</span>
							{/if}
						</div>
						<p class="text-xs text-muted-foreground">Manage tokens for API access and integrations.</p>
					</div>
				</div>
				<button
					onclick={() => { showTokenModal = true; }}
					class="flex items-center space-x-1.5 px-3 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors"
				>
					<Plus class="w-3.5 h-3.5" />
					<span>New Token</span>
				</button>
			</div>

			<!-- Token list -->
			{#if tokens.length === 0}
				<div class="p-8 text-center">
					<div class="w-12 h-12 bg-muted/50 rounded-lg flex items-center justify-center mx-auto mb-4">
						<Key class="w-6 h-6 text-muted-foreground/40" />
					</div>
					<p class="text-sm font-medium text-foreground mb-1">No API tokens</p>
					<p class="text-xs text-muted-foreground mb-4">
						Create a token to access the WatchDog API programmatically.
					</p>
					<button
						onclick={() => { showTokenModal = true; }}
						class="inline-flex items-center space-x-1.5 px-4 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors"
					>
						<Plus class="w-3.5 h-3.5" />
						<span>New Token</span>
					</button>
				</div>
			{:else}
				<div class="divide-y divide-border/20">
					{#each tokens as token (token.id)}
						<div class="px-5 py-3.5 flex items-center justify-between gap-3">
							<div class="flex items-center space-x-3 min-w-0">
								<div class="w-8 h-8 bg-yellow-500/10 rounded-lg flex items-center justify-center flex-shrink-0">
									<Key class="w-4 h-4 text-yellow-400" />
								</div>
								<div class="min-w-0">
									<div class="flex items-center space-x-2">
										<span class="text-sm text-foreground truncate">{token.name}</span>
										<code class="text-[10px] font-mono text-muted-foreground bg-muted/50 px-1.5 py-0.5 rounded">{token.prefix}...</code>
										{#if token.scope === 'admin'}
											<span class="text-[9px] font-medium uppercase px-1.5 py-0.5 rounded bg-yellow-500/15 text-yellow-400">Admin</span>
										{:else}
											<span class="text-[9px] font-medium uppercase px-1.5 py-0.5 rounded bg-blue-500/15 text-blue-400">Read Only</span>
										{/if}
									</div>
									<div class="flex items-center space-x-3 mt-0.5 text-[10px] text-muted-foreground">
										<span>Created {timeAgo(token.created_at)}</span>
										{#if token.expires_at}
											<span>Expires {timeAgo(token.expires_at)}</span>
										{:else}
											<span>No expiry</span>
										{/if}
										{#if token.last_used_at}
											<span>Last used {timeAgo(token.last_used_at)}{token.last_used_ip ? ` from ${token.last_used_ip}` : ''}</span>
										{:else}
											<span>Never used</span>
										{/if}
									</div>
								</div>
							</div>

							<div class="flex items-center space-x-1.5 flex-shrink-0">
								<!-- Regenerate button -->
								{#if confirmRegenTokenId === token.id}
									<button
										onclick={() => handleRegenerateToken(token.id)}
										disabled={regenTokenId === token.id}
										class="px-2.5 py-1.5 text-[10px] font-medium bg-yellow-500/20 text-yellow-400 hover:bg-yellow-500/30 rounded-md transition-colors disabled:opacity-50"
									>
										{regenTokenId === token.id ? 'Regenerating...' : 'Confirm Regen'}
									</button>
									<button
										onclick={() => { confirmRegenTokenId = null; }}
										class="px-2 py-1.5 text-[10px] text-muted-foreground hover:text-foreground transition-colors"
									>
										Cancel
									</button>
								{:else}
									<button
										onclick={() => handleRegenerateToken(token.id)}
										class="p-1.5 text-muted-foreground/40 hover:text-yellow-400 rounded-md hover:bg-yellow-500/10 transition-colors"
										aria-label="Regenerate token"
									>
										<RefreshCw class="w-3.5 h-3.5" />
									</button>
								{/if}

								<!-- Delete button -->
								{#if confirmDeleteTokenId === token.id}
									<button
										onclick={() => handleDeleteToken(token.id)}
										disabled={deletingTokenId === token.id}
										class="px-2.5 py-1.5 text-[10px] font-medium bg-red-500/20 text-red-400 hover:bg-red-500/30 rounded-md transition-colors disabled:opacity-50"
									>
										{deletingTokenId === token.id ? 'Deleting...' : 'Confirm'}
									</button>
									<button
										onclick={() => { confirmDeleteTokenId = null; }}
										class="px-2 py-1.5 text-[10px] text-muted-foreground hover:text-foreground transition-colors"
									>
										Cancel
									</button>
								{:else}
									<button
										onclick={() => handleDeleteToken(token.id)}
										class="p-1.5 text-muted-foreground/40 hover:text-red-400 rounded-md hover:bg-red-500/10 transition-colors"
										aria-label="Delete token"
									>
										<Trash2 class="w-3.5 h-3.5" />
									</button>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</div>

	<CreateChannelModal
		bind:open={showChannelModal}
		onClose={() => { showChannelModal = false; }}
		onCreated={handleChannelCreated}
	/>

	<CreateTokenModal
		bind:open={showTokenModal}
		onClose={() => { showTokenModal = false; plaintextToken = ''; }}
		onCreated={handleTokenCreated}
	/>
{/if}
