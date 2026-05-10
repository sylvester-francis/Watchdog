<script lang="ts">
	import { X, AlertCircle } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import type { Monitor, Agent } from '$lib/types';
	import { Modal } from '@sylvester-francis/watchdog-ui';
	import { FormField } from '@sylvester-francis/watchdog-ui';
	import { Input } from '@sylvester-francis/watchdog-ui';
	import { Select } from '@sylvester-francis/watchdog-ui';
	import { Button } from '@sylvester-francis/watchdog-ui';

	interface Props {
		open: boolean;
		monitor: Monitor;
		agents: Agent[];
		onClose: () => void;
		onUpdated: () => void;
	}

	let { open = $bindable(), monitor, agents, onClose, onUpdated }: Props = $props();

	let name = $state('');
	let target = $state('');
	let agentId = $state('');
	let intervalSeconds = $state(30);
	let timeoutSeconds = $state(10);
	let failureThreshold = $state(3);
	let enabled = $state(true);
	let slaTargetPercent = $state<number | null>(null);

	let portScanPorts = $state('');
	let portScanRange = $state('');
	let portScanExpectedOpen = $state('');
	let bannerGrab = $state(true);

	let loading = $state(false);
	let error = $state('');

	$effect(() => {
		if (open && monitor) {
			name = monitor.name;
			target = monitor.target;
			agentId = monitor.agent_id;
			intervalSeconds = monitor.interval_seconds;
			timeoutSeconds = monitor.timeout_seconds;
			failureThreshold = monitor.failure_threshold;
			enabled = monitor.enabled;
			slaTargetPercent = monitor.sla_target_percent ?? null;

			const meta = monitor.metadata ?? {};
			if (monitor.type === 'port_scan') {
				portScanPorts = meta.ports || '';
				portScanRange = meta.port_range || '';
				portScanExpectedOpen = meta.expected_open || '';
				bannerGrab = meta.banner_grab === 'true';
			}

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

		if (monitor.type === 'port_scan') {
			const portPattern = /^[\d,\s-]*$/;
			if (portScanPorts.trim() && !portPattern.test(portScanPorts)) {
				error = 'Ports must be comma-separated numbers (e.g. 80,443)';
				return;
			}
			if (portScanExpectedOpen.trim() && !portPattern.test(portScanExpectedOpen)) {
				error = 'Expected open ports must be comma-separated numbers';
				return;
			}
			if (!portScanPorts.trim() && !portScanRange.trim()) {
				error = 'Please specify ports or a port range';
				return;
			}
		}

		loading = true;
		error = '';

		try {
			const payload: Record<string, unknown> = {
				name: name.trim(),
				target: target.trim(),
				interval_seconds: intervalSeconds,
				timeout_seconds: timeoutSeconds,
				failure_threshold: failureThreshold,
				enabled,
				agent_id: agentId
			};
			if (slaTargetPercent !== null && slaTargetPercent > 0) {
				payload.sla_target_percent = slaTargetPercent;
			}
			if (monitor.type === 'port_scan') {
				const meta: Record<string, string> = {};
				if (portScanPorts.trim()) meta.ports = portScanPorts.trim();
				if (portScanRange.trim()) meta.port_range = portScanRange.trim();
				if (portScanExpectedOpen.trim()) meta.expected_open = portScanExpectedOpen.trim();
				if (bannerGrab) meta.banner_grab = 'true';
				if (Object.keys(meta).length > 0) payload.metadata = meta;
			}
			await monitorsApi.updateMonitor(monitor.id, payload);
			onUpdated();
			handleClose();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update monitor';
		} finally {
			loading = false;
		}
	}
</script>

<Modal bind:open onclose={handleClose} size="lg">
	<div class="flex items-center justify-between mb-4">
		<h3 class="text-sm font-medium text-foreground">Edit Monitor</h3>
		<button
			type="button"
			onclick={handleClose}
			class="text-muted-foreground hover:text-foreground transition-colors"
			aria-label="Close"
		>
			<X class="w-4 h-4" />
		</button>
	</div>

	<form onsubmit={handleSubmit}>
		<div class="space-y-4">
			{#if error}
				<div class="bg-destructive/10 border border-destructive/20 rounded-md px-3 py-2 flex items-center space-x-2" role="alert">
					<AlertCircle class="w-3.5 h-3.5 text-destructive flex-shrink-0" />
					<span class="text-xs text-destructive">{error}</span>
				</div>
			{/if}

			<FormField label="Name" htmlFor="edit-monitor-name" required>
				<Input id="edit-monitor-name" type="text" bind:value={name} />
			</FormField>

			<div class="grid grid-cols-2 gap-3">
				<FormField label="Type" htmlFor="edit-monitor-type">
					<Input id="edit-monitor-type" type="text" value={monitor.type.toUpperCase()} disabled />
				</FormField>
				<FormField label="Agent" htmlFor="edit-monitor-agent" required>
					<Select id="edit-monitor-agent" bind:value={agentId}>
						{#each agents as agent}
							<option value={agent.id}>{agent.name}</option>
						{/each}
					</Select>
				</FormField>
			</div>

			<FormField label="Target" htmlFor="edit-monitor-target" required>
				<Input id="edit-monitor-target" type="text" bind:value={target} />
			</FormField>

			<div class="grid grid-cols-3 gap-3">
				<FormField label="Interval (s)" htmlFor="edit-monitor-interval">
					<Input id="edit-monitor-interval" type="number" bind:value={intervalSeconds} min={5} max={3600} />
				</FormField>
				<FormField label="Timeout (s)" htmlFor="edit-monitor-timeout">
					<Input id="edit-monitor-timeout" type="number" bind:value={timeoutSeconds} min={1} max={300} />
				</FormField>
				<FormField label="Fail Threshold" htmlFor="edit-monitor-threshold">
					<Input id="edit-monitor-threshold" type="number" bind:value={failureThreshold} min={1} max={10} />
				</FormField>
			</div>

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

			<FormField label="SLA Target %" htmlFor="edit-monitor-sla">
				<input
					id="edit-monitor-sla"
					type="number"
					step="0.01"
					min="0"
					max="100"
					placeholder="e.g. 99.9"
					value={slaTargetPercent ?? ''}
					onchange={(e) => {
						const v = parseFloat((e.target as HTMLInputElement).value);
						slaTargetPercent = isNaN(v) ? null : v;
					}}
					class="w-full bg-card-elevated border border-border rounded px-3 py-2 text-foreground placeholder:text-muted-foreground transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-1 focus:ring-offset-background"
				/>
				<p class="text-[10px] text-muted-foreground mt-1">Leave empty to disable SLA tracking</p>
			</FormField>

			{#if monitor.type === 'port_scan'}
				<div class="space-y-3 pt-1">
					<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">Port Scan Settings</div>
					<FormField label="Ports (comma-separated)" htmlFor="edit-monitor-ps-ports">
						<Input id="edit-monitor-ps-ports" type="text" bind:value={portScanPorts} placeholder="22,80,443,3306,8080" />
						<p class="text-[10px] text-muted-foreground mt-1">Supports ranges: 8000-9000</p>
					</FormField>
					<FormField label="Port Range (alternative)" htmlFor="edit-monitor-ps-range">
						<Input id="edit-monitor-ps-range" type="text" bind:value={portScanRange} placeholder="1-1024" />
					</FormField>
					<FormField label="Expected Open Ports (optional)" htmlFor="edit-monitor-ps-expected">
						<Input id="edit-monitor-ps-expected" type="text" bind:value={portScanExpectedOpen} placeholder="22,80,443" />
						<p class="text-[10px] text-muted-foreground mt-1">If set, alerts when expected ports close or unexpected ports open</p>
					</FormField>
					<div class="flex items-center justify-between pt-1">
						<div>
							<label for="edit-monitor-banner-grab" class="block text-xs font-medium text-muted-foreground mb-1.5">Service Detection</label>
							<p class="text-[10px] text-muted-foreground">Identify services and versions on open ports</p>
						</div>
						<label class="relative inline-flex items-center cursor-pointer">
							<input
								id="edit-monitor-banner-grab"
								type="checkbox"
								bind:checked={bannerGrab}
								class="sr-only peer"
							/>
							<div class="w-9 h-5 bg-muted rounded-full peer peer-checked:bg-accent transition-colors after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-4"></div>
						</label>
					</div>
				</div>
			{/if}
		</div>

		<div class="mt-4 pt-4 border-t border-border flex justify-end space-x-2">
			<Button variant="outline" size="sm" onclick={handleClose}>Cancel</Button>
			<Button variant="primary" size="sm" type="submit" disabled={loading}>
				{loading ? 'Saving...' : 'Save Changes'}
			</Button>
		</div>
	</form>
</Modal>
