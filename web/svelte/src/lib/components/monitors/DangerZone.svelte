<script lang="ts">
	import { goto } from '$app/navigation';
	import { Trash2, AlertTriangle } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast';

	interface Props {
		monitorId: string;
	}

	let { monitorId }: Props = $props();

	const toast = getToasts();

	let showConfirm = $state(false);
	let deleting = $state(false);

	async function handleDelete() {
		deleting = true;
		try {
			await monitorsApi.deleteMonitor(monitorId);
			toast.success('Monitor deleted');
			goto(`/monitors`);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to delete monitor');
		} finally {
			deleting = false;
		}
	}
</script>

<div class="bg-card border border-red-500/20 rounded-lg">
	<div class="px-4 py-3 border-b border-red-500/20 flex items-center space-x-2">
		<AlertTriangle class="w-4 h-4 text-red-400" />
		<h2 class="text-sm font-medium text-red-400">Danger Zone</h2>
	</div>

	<div class="p-4">
		<div class="flex items-start justify-between">
			<div>
				<p class="text-sm text-foreground font-medium">Delete this monitor</p>
				<p class="text-xs text-muted-foreground mt-1">
					Once deleted, all heartbeat data and incident history for this monitor will be permanently removed.
				</p>
			</div>

			{#if !showConfirm}
				<button
					onclick={() => { showConfirm = true; }}
					class="shrink-0 ml-4 flex items-center space-x-1.5 px-3 py-1.5 bg-red-500/10 text-red-400 hover:bg-red-500/20 border border-red-500/20 text-xs font-medium rounded-md transition-colors"
				>
					<Trash2 class="w-3.5 h-3.5" />
					<span>Delete</span>
				</button>
			{/if}
		</div>

		{#if showConfirm}
			<div class="mt-4 p-3 bg-red-500/5 border border-red-500/20 rounded-md">
				<p class="text-xs text-red-400 mb-3">
					Are you sure? This will delete all heartbeat data and cannot be undone.
				</p>
				<div class="flex items-center space-x-2">
					<button
						onclick={handleDelete}
						disabled={deleting}
						class="px-3 py-1.5 bg-red-500 text-white hover:bg-red-600 text-xs font-medium rounded-md transition-colors disabled:opacity-50"
					>
						{deleting ? 'Deleting...' : 'Yes, delete monitor'}
					</button>
					<button
						onclick={() => { showConfirm = false; }}
						class="px-3 py-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
					>
						Cancel
					</button>
				</div>
			</div>
		{/if}
	</div>
</div>
