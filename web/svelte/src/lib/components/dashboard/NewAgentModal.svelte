<script lang="ts">
	import { Copy, Check, X } from 'lucide-svelte';
	import { agents as agentsApi } from '$lib/api';
	import { Modal } from '@sylvester-francis/watchdog-ui';
	import { Button } from '@sylvester-francis/watchdog-ui';
	import { Input } from '@sylvester-francis/watchdog-ui';
	import { FormField } from '@sylvester-francis/watchdog-ui';

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

<Modal bind:open onclose={handleClose}>
	<div class="space-y-4">
		<div class="flex items-center justify-between">
			<h3 class="text-sm font-medium text-foreground">Create Agent</h3>
			<button onclick={handleClose} class="text-muted-foreground hover:text-foreground transition-colors" aria-label="Close">
				<X class="w-4 h-4" />
			</button>
		</div>

		{#if !createdAgent}
			<form onsubmit={handleSubmit} class="space-y-4">
				<FormField label="Agent Name" htmlFor="agent-name" required error={error || null}>
					<Input
						id="agent-name"
						name="name"
						bind:value={name}
						placeholder="e.g. Homelab, Production, Staging"
						error={!!error}
					/>
				</FormField>
				<div class="flex justify-end space-x-2 pt-2">
					<Button variant="outline" size="sm" onclick={handleClose}>Cancel</Button>
					<Button variant="primary" size="sm" type="submit" disabled={loading}>
						{loading ? 'Creating...' : 'Create Agent'}
					</Button>
				</div>
			</form>
		{:else}
			<div class="space-y-3">
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
				<div class="flex justify-end pt-2">
					<Button variant="primary" size="sm" onclick={handleClose}>Done</Button>
				</div>
			</div>
		{/if}
	</div>
</Modal>
