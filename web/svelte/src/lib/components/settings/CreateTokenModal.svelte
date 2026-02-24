<script lang="ts">
	import { X, AlertCircle, Copy, Check, Key, ShieldCheck, Eye } from 'lucide-svelte';
	import { settings as settingsApi } from '$lib/api';

	interface Props {
		open: boolean;
		onClose: () => void;
		onCreated: (plaintext: string) => void;
	}

	let { open = $bindable(), onClose, onCreated }: Props = $props();

	// Form state
	let name = $state('');
	let scope = $state<'admin' | 'read_only'>('read_only');
	let expires = $state<'' | '30d' | '90d'>('');
	let loading = $state(false);
	let error = $state('');

	// Created state
	let view = $state<'form' | 'created'>('form');
	let plaintext = $state('');
	let copied = $state(false);

	const inputClass = 'w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background';
	const labelClass = 'block text-xs font-medium text-muted-foreground mb-1.5';

	function resetForm() {
		name = '';
		scope = 'read_only';
		expires = '';
		error = '';
		loading = false;
		view = 'form';
		plaintext = '';
		copied = false;
	}

	function handleClose() {
		resetForm();
		onClose();
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();

		if (!name.trim()) {
			error = 'Name is required.';
			return;
		}

		loading = true;
		error = '';

		try {
			const res = await settingsApi.createToken({
				name: name.trim(),
				scope,
				expires
			});
			plaintext = res.plaintext;
			view = 'created';
			onCreated(res.plaintext);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create token.';
		} finally {
			loading = false;
		}
	}

	async function copyToClipboard() {
		await navigator.clipboard.writeText(plaintext);
		copied = true;
		setTimeout(() => { copied = false; }, 2000);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') handleClose();
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm animate-fade-in"
		onkeydown={handleKeydown}
	>
		<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
		<div
			class="bg-card border border-border rounded-lg shadow-lg w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto animate-fade-in-up"
			onclick={(e) => e.stopPropagation()}
			role="dialog"
			aria-label="Create API token"
		>
			<!-- Header -->
			<div class="px-5 py-3.5 border-b border-border flex items-center justify-between">
				<h3 class="text-sm font-medium text-foreground">
					{view === 'form' ? 'New API Token' : 'Token Created'}
				</h3>
				<button
					onclick={handleClose}
					class="text-muted-foreground hover:text-foreground transition-colors"
					aria-label="Close"
				>
					<X class="w-4 h-4" />
				</button>
			</div>

			{#if view === 'form'}
				<!-- Form view -->
				<form onsubmit={handleSubmit}>
					<div class="p-5 space-y-4">
						{#if error}
							<div class="bg-destructive/10 border border-destructive/20 rounded-md px-3 py-2 flex items-center space-x-2" role="alert">
								<AlertCircle class="w-3.5 h-3.5 text-destructive flex-shrink-0" />
								<span class="text-xs text-destructive">{error}</span>
							</div>
						{/if}

						<!-- Name -->
						<div>
							<label for="token-name" class={labelClass}>Name</label>
							<input
								id="token-name"
								type="text"
								bind:value={name}
								required
								placeholder="e.g. CI Pipeline, Grafana Integration"
								class={inputClass}
							/>
						</div>

						<!-- Scope: radio cards -->
						<div>
							<span class={labelClass}>Scope</span>
							<div class="grid grid-cols-2 gap-3 mt-1">
								<button
									type="button"
									onclick={() => { scope = 'admin'; }}
									class="flex flex-col items-center justify-center px-3 py-2.5 rounded-md border cursor-pointer transition-colors {scope === 'admin'
										? 'border-accent bg-accent/5'
										: 'border-border bg-card-elevated hover:bg-muted/50'}"
								>
									<ShieldCheck class="w-4 h-4 {scope === 'admin' ? 'text-accent' : 'text-muted-foreground'} mb-1" />
									<span class="text-xs font-medium {scope === 'admin' ? 'text-foreground' : 'text-muted-foreground'}">Admin</span>
									<span class="text-[9px] text-muted-foreground mt-0.5">Full access</span>
								</button>
								<button
									type="button"
									onclick={() => { scope = 'read_only'; }}
									class="flex flex-col items-center justify-center px-3 py-2.5 rounded-md border cursor-pointer transition-colors {scope === 'read_only'
										? 'border-accent bg-accent/5'
										: 'border-border bg-card-elevated hover:bg-muted/50'}"
								>
									<Eye class="w-4 h-4 {scope === 'read_only' ? 'text-accent' : 'text-muted-foreground'} mb-1" />
									<span class="text-xs font-medium {scope === 'read_only' ? 'text-foreground' : 'text-muted-foreground'}">Read Only</span>
									<span class="text-[9px] text-muted-foreground mt-0.5">View data only</span>
								</button>
							</div>
						</div>

						<!-- Expiration: radio cards -->
						<div>
							<span class={labelClass}>Expiration</span>
							<div class="grid grid-cols-3 gap-3 mt-1">
								<button
									type="button"
									onclick={() => { expires = ''; }}
									class="flex flex-col items-center justify-center px-3 py-2.5 rounded-md border cursor-pointer transition-colors {expires === ''
										? 'border-accent bg-accent/5'
										: 'border-border bg-card-elevated hover:bg-muted/50'}"
								>
									<span class="text-xs font-medium {expires === '' ? 'text-foreground' : 'text-muted-foreground'}">Never</span>
									<span class="text-[9px] text-muted-foreground mt-0.5">No expiry</span>
								</button>
								<button
									type="button"
									onclick={() => { expires = '30d'; }}
									class="flex flex-col items-center justify-center px-3 py-2.5 rounded-md border cursor-pointer transition-colors {expires === '30d'
										? 'border-accent bg-accent/5'
										: 'border-border bg-card-elevated hover:bg-muted/50'}"
								>
									<span class="text-xs font-medium {expires === '30d' ? 'text-foreground' : 'text-muted-foreground'}">30 days</span>
									<span class="text-[9px] text-muted-foreground mt-0.5">Short-lived</span>
								</button>
								<button
									type="button"
									onclick={() => { expires = '90d'; }}
									class="flex flex-col items-center justify-center px-3 py-2.5 rounded-md border cursor-pointer transition-colors {expires === '90d'
										? 'border-accent bg-accent/5'
										: 'border-border bg-card-elevated hover:bg-muted/50'}"
								>
									<span class="text-xs font-medium {expires === '90d' ? 'text-foreground' : 'text-muted-foreground'}">90 days</span>
									<span class="text-[9px] text-muted-foreground mt-0.5">Quarterly</span>
								</button>
							</div>
						</div>
					</div>

					<!-- Footer -->
					<div class="px-5 py-3.5 border-t border-border flex justify-end space-x-2">
						<button
							type="button"
							onclick={handleClose}
							class="px-4 py-2 bg-muted text-muted-foreground hover:bg-muted/80 text-xs font-medium rounded-md transition-colors"
						>
							Cancel
						</button>
						<button
							type="submit"
							disabled={loading}
							class="px-4 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors disabled:opacity-50"
						>
							{loading ? 'Creating...' : 'Create Token'}
						</button>
					</div>
				</form>
			{:else}
				<!-- Token Created view -->
				<div class="p-5 space-y-4">
					<div class="flex items-center space-x-2">
						<div class="w-8 h-8 bg-emerald-500/10 rounded-lg flex items-center justify-center">
							<Check class="w-4 h-4 text-emerald-400" />
						</div>
						<div>
							<p class="text-sm font-medium text-foreground">Token created successfully</p>
							<p class="text-xs text-muted-foreground">Your new API token is ready to use.</p>
						</div>
					</div>

					<div>
						<label class={labelClass}>API Token</label>
						<div class="flex items-center space-x-2">
							<code class="flex-1 text-xs font-mono bg-yellow-500/10 border border-yellow-500/20 rounded px-3 py-2.5 text-foreground break-all select-all">{plaintext}</code>
							<button
								onclick={copyToClipboard}
								class="p-2 rounded-md hover:bg-muted/50 text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
								aria-label="Copy token"
							>
								{#if copied}
									<Check class="w-4 h-4 text-emerald-400" />
								{:else}
									<Copy class="w-4 h-4" />
								{/if}
							</button>
						</div>
					</div>

					<div class="bg-yellow-500/10 border border-yellow-500/20 rounded-md px-3 py-2 flex items-center space-x-2">
						<Key class="w-3.5 h-3.5 text-yellow-400 flex-shrink-0" />
						<span class="text-xs text-yellow-400">Copy this token now. You won't be able to see it again.</span>
					</div>
				</div>

				<!-- Footer -->
				<div class="px-5 py-3.5 border-t border-border flex justify-end">
					<button
						onclick={handleClose}
						class="px-4 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors"
					>
						Done
					</button>
				</div>
			{/if}
		</div>
	</div>
{/if}
