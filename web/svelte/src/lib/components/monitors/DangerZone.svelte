<script lang="ts">
	import { goto } from '$app/navigation';
	import { Trash2 } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast.svelte';
	import ConfirmModal from '$lib/components/ConfirmModal.svelte';
	import { Button } from '@sylvester-francis/watchdog-ui';

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

<section>
	<div class="border-b border-destructive/30 pb-3">
		<h3 class="text-sm font-medium text-destructive">Danger Zone</h3>
	</div>

	<div class="flex flex-col items-start gap-3 pt-4 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
		<div>
			<p class="text-sm font-medium text-foreground">Delete this monitor</p>
			<p class="mt-1 text-xs text-muted-foreground">
				Once deleted, all heartbeat data and incident history for this monitor will be permanently removed.
			</p>
		</div>

		<Button variant="destructive" size="sm" onclick={() => { showConfirm = true; }}>
			<span class="flex items-center gap-1.5">
				<Trash2 class="h-3.5 w-3.5" />
				<span>Delete Monitor</span>
			</span>
		</Button>
	</div>
</section>

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
