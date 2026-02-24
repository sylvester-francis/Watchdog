<script lang="ts">
	import { X, Copy, Check } from 'lucide-svelte';
	import { agents as agentsApi } from '$lib/api';

	interface Props {
		open: boolean;
		onClose: () => void;
		onCreated: () => void;
	}

	let { open = $bindable(), onClose, onCreated }: Props = $props();

	let name = $state('');
	let loading = $state(false);
	let error = $state('');
	let createdAgent = $state<{ id: string; name: string; api_key: string } | null>(null);
	let copied = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!name.trim()) return;

		loading = true;
		error = '';
		try {
			const res = await agentsApi.createAgent(name.trim());
			createdAgent = res.data;
			onCreated();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create agent';
		} finally {
			loading = false;
		}
	}

	function handleClose() {
		name = '';
		error = '';
		createdAgent = null;
		copied = false;
		onClose();
	}

	async function copyKey() {
		if (!createdAgent) return;
		await navigator.clipboard.writeText(createdAgent.api_key);
		copied = true;
		setTimeout(() => { copied = false; }, 2000);
	}
</script>

{#if open}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
		role="dialog"
		aria-label="Create agent"
	>
		<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
		<div
			class="bg-card border border-border rounded-lg shadow-lg w-full max-w-md mx-4 max-h-[90vh] overflow-y-auto"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => { if (e.key === 'Escape') handleClose(); }}
			role="document"
		>
			<div class="px-4 py-3 border-b border-border flex items-center justify-between">
				<h3 class="text-sm font-medium text-foreground">Create Agent</h3>
				<button onclick={handleClose} class="text-muted-foreground hover:text-foreground" aria-label="Close">
					<X class="w-4 h-4" />
				</button>
			</div>

			{#if !createdAgent}
				<form onsubmit={handleSubmit}>
					<div class="p-4">
						<label for="agent-name" class="block text-xs text-muted-foreground mb-1">Agent Name</label>
						<input
							id="agent-name"
							type="text"
							bind:value={name}
							required
							placeholder="e.g. Homelab, Production, Staging"
							class="w-full px-3 py-2 bg-background border border-border rounded-md text-sm text-foreground placeholder:text-muted-foreground/50 focus:outline-none focus:ring-1 focus:ring-accent"
						/>
						{#if error}
							<p class="text-xs text-red-400 mt-2">{error}</p>
						{/if}
					</div>
					<div class="px-4 py-3 border-t border-border flex justify-end space-x-2">
						<button type="button" onclick={handleClose} class="px-3 py-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors">
							Cancel
						</button>
						<button
							type="submit"
							disabled={loading}
							class="px-3 py-1.5 bg-accent text-accent-foreground text-xs font-medium rounded-md hover:bg-accent/90 transition-colors disabled:opacity-50"
						>
							{loading ? 'Creating...' : 'Create Agent'}
						</button>
					</div>
				</form>
			{:else}
				<div class="p-4 space-y-3">
					<div>
						<p class="text-xs text-muted-foreground mb-1">Agent created successfully</p>
						<p class="text-sm font-medium text-foreground">{createdAgent.name}</p>
					</div>
					<div>
						<p class="text-xs text-muted-foreground mb-1">API Key (shown once)</p>
						<div class="flex items-center space-x-2">
							<code class="flex-1 text-xs font-mono bg-muted/50 rounded px-2 py-1.5 text-foreground break-all select-all">{createdAgent.api_key}</code>
							<button onclick={copyKey} class="p-1.5 rounded-md hover:bg-muted/50 text-muted-foreground hover:text-foreground transition-colors" aria-label="Copy API key">
								{#if copied}
									<Check class="w-4 h-4 text-emerald-400" />
								{:else}
									<Copy class="w-4 h-4" />
								{/if}
							</button>
						</div>
					</div>
					<div>
						<p class="text-xs text-muted-foreground mb-1">Install command</p>
						<code class="block text-xs font-mono bg-muted/50 rounded px-2 py-1.5 text-foreground select-all">curl -fsSL https://usewatchdog.dev/install | bash</code>
					</div>
				</div>
				<div class="px-4 py-3 border-t border-border flex justify-end">
					<button onclick={handleClose} class="px-3 py-1.5 bg-accent text-accent-foreground text-xs font-medium rounded-md hover:bg-accent/90 transition-colors">
						Done
					</button>
				</div>
			{/if}
		</div>
	</div>
{/if}
