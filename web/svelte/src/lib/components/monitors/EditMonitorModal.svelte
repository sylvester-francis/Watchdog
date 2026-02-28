<script lang="ts">
	import { X, AlertCircle } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import type { Monitor } from '$lib/types';

	interface Props {
		open: boolean;
		monitor: Monitor;
		onClose: () => void;
		onUpdated: () => void;
	}

	let { open = $bindable(), monitor, onClose, onUpdated }: Props = $props();

	// Editable fields (type and agent are read-only after creation)
	let name = $state('');
	let target = $state('');
	let intervalSeconds = $state(30);
	let timeoutSeconds = $state(10);
	let failureThreshold = $state(3);
	let enabled = $state(true);

	let loading = $state(false);
	let error = $state('');

	// Sync form state when monitor prop changes or modal opens
	$effect(() => {
		if (open && monitor) {
			name = monitor.name;
			target = monitor.target;
			intervalSeconds = monitor.interval_seconds;
			timeoutSeconds = monitor.timeout_seconds;
			failureThreshold = monitor.failure_threshold;
			enabled = monitor.enabled;
			error = '';
			loading = false;
		}
	});

	function handleClose() {
		error = '';
		loading = false;
		onClose();
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();

		if (!name.trim()) {
			error = 'Name is required';
			return;
		}
		if (!target.trim()) {
			error = 'Target is required';
			return;
		}

		loading = true;
		error = '';

		try {
			await monitorsApi.updateMonitor(monitor.id, {
				name: name.trim(),
				target: target.trim(),
				interval_seconds: intervalSeconds,
				timeout_seconds: timeoutSeconds,
				failure_threshold: failureThreshold,
				enabled
			});
			onUpdated();
			handleClose();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update monitor';
		} finally {
			loading = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') handleClose();
	}

	const inputClass = 'w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background';
	const labelClass = 'block text-xs font-medium text-muted-foreground mb-1.5';
	const readOnlyClass = 'w-full px-3 py-2 bg-muted/30 border border-border/50 rounded-md text-sm text-muted-foreground cursor-not-allowed';
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
			aria-label="Edit monitor"
			tabindex="-1"
		>
			<!-- Header -->
			<div class="px-5 py-3.5 border-b border-border flex items-center justify-between">
				<h3 class="text-sm font-medium text-foreground">Edit Monitor</h3>
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

					<!-- Name -->
					<div>
						<label for="edit-monitor-name" class={labelClass}>Name</label>
						<input
							id="edit-monitor-name"
							type="text"
							bind:value={name}
							required
							class={inputClass}
						/>
					</div>

					<!-- Type + Agent (read-only) -->
					<div class="grid grid-cols-2 gap-3">
						<div>
							<label for="edit-monitor-type" class={labelClass}>Type</label>
							<input
								id="edit-monitor-type"
								type="text"
								value={monitor.type.toUpperCase()}
								disabled
								class={readOnlyClass}
							/>
						</div>
						<div>
							<label for="edit-monitor-agent" class={labelClass}>Agent</label>
							<input
								id="edit-monitor-agent"
								type="text"
								value={monitor.agent_id.slice(0, 8) + '...'}
								disabled
								class={readOnlyClass}
								title={monitor.agent_id}
							/>
						</div>
					</div>

					<!-- Target -->
					<div>
						<label for="edit-monitor-target" class={labelClass}>Target</label>
						<input
							id="edit-monitor-target"
							type="text"
							bind:value={target}
							required
							class={inputClass}
						/>
					</div>

					<!-- Interval / Timeout / Failure Threshold -->
					<div class="grid grid-cols-3 gap-3">
						<div>
							<label for="edit-monitor-interval" class={labelClass}>Interval (s)</label>
							<input
								id="edit-monitor-interval"
								type="number"
								bind:value={intervalSeconds}
								min="5"
								max="3600"
								class={inputClass}
							/>
						</div>
						<div>
							<label for="edit-monitor-timeout" class={labelClass}>Timeout (s)</label>
							<input
								id="edit-monitor-timeout"
								type="number"
								bind:value={timeoutSeconds}
								min="1"
								max="300"
								class={inputClass}
							/>
						</div>
						<div>
							<label for="edit-monitor-threshold" class={labelClass}>Fail Threshold</label>
							<input
								id="edit-monitor-threshold"
								type="number"
								bind:value={failureThreshold}
								min="1"
								max="10"
								class={inputClass}
							/>
						</div>
					</div>

					<!-- Enabled toggle -->
					<div class="flex items-center justify-between py-1">
						<div>
							<span class="text-xs font-medium text-foreground">Enabled</span>
							<p class="text-[10px] text-muted-foreground mt-0.5">Disable to pause monitoring</p>
						</div>
						<button
							type="button"
							onclick={() => enabled = !enabled}
							class="relative w-9 h-5 rounded-full transition-colors {enabled ? 'bg-emerald-500' : 'bg-muted'}"
							role="switch"
							aria-checked={enabled}
							aria-label="Toggle monitor enabled"
						>
							<span
								class="absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white transition-transform {enabled ? 'translate-x-4' : 'translate-x-0'}"
							></span>
						</button>
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
						{loading ? 'Saving...' : 'Save Changes'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
