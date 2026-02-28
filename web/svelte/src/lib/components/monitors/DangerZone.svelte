<script lang="ts">
	import { goto } from '$app/navigation';
	import { Trash2, AlertTriangle } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast.svelte';
	import ConfirmModal from '$lib/components/ConfirmModal.svelte';

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
			showConfirm = false;
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
	<div class="px-5 py-3.5 border-b border-red-500/20 flex items-center space-x-2">
		<AlertTriangle class="w-4 h-4 text-destructive" />
		<h3 class="text-sm font-medium text-destructive">Danger Zone</h3>
	</div>

	<div class="px-5 py-4">
		<div class="flex items-start justify-between">
			<div>
				<p class="text-sm text-foreground font-medium">Delete this monitor</p>
				<p class="text-xs text-muted-foreground mt-1">
					Once deleted, all heartbeat data and incident history for this monitor will be permanently removed.
				</p>
			</div>

			<button
				onclick={() => { showConfirm = true; }}
				class="shrink-0 ml-4 flex items-center space-x-1.5 px-3 py-1.5 bg-red-500/10 text-red-400 hover:bg-red-500/20 border border-red-500/20 text-xs font-medium rounded-md transition-colors"
			>
				<Trash2 class="w-3.5 h-3.5" />
				<span>Delete Monitor</span>
			</button>
		</div>
	</div>
</div>

<ConfirmModal
	open={showConfirm}
	title="Delete Monitor"
	message="Are you sure you want to delete this monitor? All heartbeat data and incident history will be permanently removed. This cannot be undone."
	confirmLabel="Yes, delete monitor"
	variant="danger"
	loading={deleting}
	onConfirm={handleDelete}
	onCancel={() => { showConfirm = false; }}
/>
