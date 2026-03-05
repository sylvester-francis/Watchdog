<script lang="ts">
	import { X, AlertCircle } from 'lucide-svelte';
	import { maintenance as maintenanceApi, agents as agentsApi } from '$lib/api';
	import type { Agent } from '$lib/types';
	import { onMount } from 'svelte';

	interface Props {
		open: boolean;
		onClose: () => void;
		onCreated: () => void;
	}

	let { open = $bindable(), onClose, onCreated }: Props = $props();

	import type { MaintenanceRecurrence } from '$lib/types';

	let name = $state('');
	let agentId = $state('');
	let startsAt = $state('');
	let endsAt = $state('');
	let recurrence = $state<MaintenanceRecurrence>('once');
	let loading = $state(false);
	let error = $state('');
	let agents = $state<Agent[]>([]);

	const inputClass = 'w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background';
	const labelClass = 'block text-xs font-medium text-muted-foreground mb-1.5';

	function resetForm() {
		name = '';
		agentId = '';
		startsAt = '';
		endsAt = '';
		recurrence = 'once';
		error = '';
		loading = false;
	}

	function handleClose() {
		resetForm();
		onClose();
	}

	async function loadAgents() {
		try {
			const res = await agentsApi.listAgents();
			agents = res.data ?? [];
		} catch {
			// silent
		}
	}

	function toRFC3339(localDatetime: string): string {
		if (!localDatetime) return '';
		return new Date(localDatetime).toISOString();
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';

		if (!name.trim()) {
			error = 'Name is required.';
			return;
		}
		if (!agentId) {
			error = 'Please select an agent.';
			return;
		}
		if (!startsAt || !endsAt) {
			error = 'Start and end times are required.';
			return;
		}

		const startDate = new Date(startsAt);
		const endDate = new Date(endsAt);

		if (endDate <= startDate) {
			error = 'End time must be after start time.';
			return;
		}

		const durationMs = endDate.getTime() - startDate.getTime();
		if (durationMs > 30 * 24 * 60 * 60 * 1000) {
			error = 'Maintenance window cannot exceed 30 days.';
			return;
		}

		loading = true;

		try {
			await maintenanceApi.createWindow({
				agent_id: agentId,
				name: name.trim(),
				starts_at: toRFC3339(startsAt),
				ends_at: toRFC3339(endsAt),
				recurrence
			});
			onCreated();
			handleClose();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to schedule maintenance window.';
		} finally {
			loading = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') handleClose();
	}

	onMount(() => {
		loadAgents();
	});
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm animate-fade-in"
		onkeydown={handleKeydown}
	>
		<div
			class="bg-card border border-border rounded-lg shadow-lg w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto animate-fade-in-up"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
			role="dialog"
			aria-label="Schedule maintenance window"
			tabindex="-1"
		>
			<!-- Header -->
			<div class="px-5 py-3.5 border-b border-border flex items-center justify-between">
				<h3 class="text-sm font-medium text-foreground">Schedule Maintenance</h3>
				<button
					onclick={handleClose}
					class="text-muted-foreground hover:text-foreground transition-colors"
					aria-label="Close"
				>
					<X class="w-4 h-4" />
				</button>
			</div>

			<!-- Form -->
			<form onsubmit={handleSubmit}>
				<div class="p-5 space-y-4">
					{#if error}
						<div class="bg-destructive/10 border border-destructive/20 rounded-md px-3 py-2 flex items-center space-x-2" role="alert">
							<AlertCircle class="w-3.5 h-3.5 text-destructive flex-shrink-0" />
							<span class="text-xs text-destructive">{error}</span>
						</div>
					{/if}

					<div>
						<label for="mw-name" class={labelClass}>Name</label>
						<input
							id="mw-name"
							type="text"
							bind:value={name}
							required
							placeholder="e.g. Server patching"
							class={inputClass}
						/>
					</div>

					<div>
						<label for="mw-agent" class={labelClass}>Agent</label>
						<select
							id="mw-agent"
							bind:value={agentId}
							class={inputClass}
						>
							<option value="">Select an agent...</option>
							{#each agents as agent}
								<option value={agent.id}>{agent.name}</option>
							{/each}
						</select>
						<p class="text-[10px] text-muted-foreground/60 mt-1">All monitors on this agent will be suppressed during the window.</p>
					</div>

					<div class="grid grid-cols-2 gap-3">
						<div>
							<label for="mw-starts" class={labelClass}>Start Time</label>
							<input
								id="mw-starts"
								type="datetime-local"
								bind:value={startsAt}
								required
								class={inputClass}
							/>
						</div>
						<div>
							<label for="mw-ends" class={labelClass}>End Time</label>
							<input
								id="mw-ends"
								type="datetime-local"
								bind:value={endsAt}
								required
								class={inputClass}
							/>
						</div>
					</div>

					<div>
						<label for="mw-recurrence" class={labelClass}>Recurrence</label>
						<select
							id="mw-recurrence"
							bind:value={recurrence}
							class={inputClass}
						>
							<option value="once">Once</option>
							<option value="daily">Daily</option>
							<option value="weekly">Weekly</option>
							<option value="monthly">Monthly</option>
						</select>
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
						{loading ? 'Scheduling...' : 'Schedule'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
