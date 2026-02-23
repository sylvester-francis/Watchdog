<script lang="ts">
	import { X, AlertCircle } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import type { Agent, MonitorType } from '$lib/types';

	interface Props {
		open: boolean;
		agents: Agent[];
		onClose: () => void;
		onCreated: () => void;
	}

	let { open = $bindable(), agents, onClose, onCreated }: Props = $props();

	// Form state
	let name = $state('');
	let type = $state<MonitorType>('http');
	let target = $state('');
	let agentId = $state('');
	let intervalSeconds = $state(30);
	let timeoutSeconds = $state(10);
	let failureThreshold = $state(3);

	// HTTP-specific
	let expectedStatus = $state(200);

	// Database-specific
	let dbType = $state('postgres');
	let dbConnectionString = $state('');
	let dbPassword = $state('');

	// System-specific
	let systemMetric = $state('cpu');
	let systemThreshold = $state(80);

	let loading = $state(false);
	let error = $state('');

	const monitorTypes: { value: MonitorType; label: string }[] = [
		{ value: 'http', label: 'HTTP' },
		{ value: 'tcp', label: 'TCP' },
		{ value: 'ping', label: 'Ping' },
		{ value: 'dns', label: 'DNS' },
		{ value: 'tls', label: 'TLS' },
		{ value: 'docker', label: 'Docker' },
		{ value: 'database', label: 'Database' },
		{ value: 'system', label: 'System' }
	];

	const targetPlaceholders: Record<MonitorType, string> = {
		http: 'https://example.com',
		tcp: 'host:port',
		ping: '192.168.1.1',
		dns: 'example.com',
		tls: 'example.com:443',
		docker: 'container_name',
		database: 'localhost:5432',
		system: 'localhost'
	};

	function buildMetadata(): Record<string, string> | undefined {
		const meta: Record<string, string> = {};

		if (type === 'http' && expectedStatus !== 200) {
			meta.expected_status = String(expectedStatus);
		} else if (type === 'http') {
			meta.expected_status = String(expectedStatus);
		}

		if (type === 'database') {
			meta.db_type = dbType;
			if (dbConnectionString) meta.connection_string = dbConnectionString;
			if (dbPassword) meta.password = dbPassword;
		}

		if (type === 'system') {
			meta.metric = systemMetric;
			meta.threshold = String(systemThreshold);
		}

		return Object.keys(meta).length > 0 ? meta : undefined;
	}

	function resetForm() {
		name = '';
		type = 'http';
		target = '';
		agentId = '';
		intervalSeconds = 30;
		timeoutSeconds = 10;
		failureThreshold = 3;
		expectedStatus = 200;
		dbType = 'postgres';
		dbConnectionString = '';
		dbPassword = '';
		systemMetric = 'cpu';
		systemThreshold = 80;
		error = '';
		loading = false;
	}

	function handleClose() {
		resetForm();
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
		if (!agentId) {
			error = 'Please select an agent';
			return;
		}

		loading = true;
		error = '';

		try {
			await monitorsApi.createMonitor({
				name: name.trim(),
				type,
				target: target.trim(),
				agent_id: agentId,
				interval_seconds: intervalSeconds,
				timeout_seconds: timeoutSeconds,
				failure_threshold: failureThreshold,
				metadata: buildMetadata()
			});
			onCreated();
			handleClose();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create monitor';
		} finally {
			loading = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') handleClose();
	}

	const inputClass = 'w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background';
	const labelClass = 'block text-xs font-medium text-muted-foreground mb-1.5';
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
			aria-label="Create monitor"
		>
			<!-- Header -->
			<div class="px-5 py-3.5 border-b border-border flex items-center justify-between">
				<h3 class="text-sm font-medium text-foreground">Create Monitor</h3>
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
						<label for="monitor-name" class={labelClass}>Name</label>
						<input
							id="monitor-name"
							type="text"
							bind:value={name}
							required
							placeholder="My API Server"
							class={inputClass}
						/>
					</div>

					<!-- Type + Agent row -->
					<div class="grid grid-cols-2 gap-3">
						<div>
							<label for="monitor-type" class={labelClass}>Type</label>
							<select
								id="monitor-type"
								bind:value={type}
								class={inputClass}
							>
								{#each monitorTypes as mt}
									<option value={mt.value}>{mt.label}</option>
								{/each}
							</select>
						</div>
						<div>
							<label for="monitor-agent" class={labelClass}>Agent</label>
							<select
								id="monitor-agent"
								bind:value={agentId}
								required
								class={inputClass}
							>
								<option value="" disabled>Select agent</option>
								{#each agents as agent}
									<option value={agent.id}>{agent.name}</option>
								{/each}
							</select>
						</div>
					</div>

					<!-- Target -->
					<div>
						<label for="monitor-target" class={labelClass}>Target</label>
						<input
							id="monitor-target"
							type="text"
							bind:value={target}
							required
							placeholder={targetPlaceholders[type]}
							class={inputClass}
						/>
					</div>

					<!-- Interval / Timeout / Failure Threshold -->
					<div class="grid grid-cols-3 gap-3">
						<div>
							<label for="monitor-interval" class={labelClass}>Interval (s)</label>
							<input
								id="monitor-interval"
								type="number"
								bind:value={intervalSeconds}
								min="5"
								max="3600"
								class={inputClass}
							/>
						</div>
						<div>
							<label for="monitor-timeout" class={labelClass}>Timeout (s)</label>
							<input
								id="monitor-timeout"
								type="number"
								bind:value={timeoutSeconds}
								min="1"
								max="300"
								class={inputClass}
							/>
						</div>
						<div>
							<label for="monitor-threshold" class={labelClass}>Fail Threshold</label>
							<input
								id="monitor-threshold"
								type="number"
								bind:value={failureThreshold}
								min="1"
								max="10"
								class={inputClass}
							/>
						</div>
					</div>

					<!-- Type-specific fields -->
					{#if type === 'http'}
						<div>
							<label for="monitor-expected-status" class={labelClass}>Expected Status Code</label>
							<input
								id="monitor-expected-status"
								type="number"
								bind:value={expectedStatus}
								min="100"
								max="599"
								class={inputClass}
							/>
						</div>
					{/if}

					{#if type === 'database'}
						<div class="space-y-3 pt-1">
							<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">Database Settings</div>
							<div>
								<label for="monitor-db-type" class={labelClass}>DB Type</label>
								<select
									id="monitor-db-type"
									bind:value={dbType}
									class={inputClass}
								>
									<option value="postgres">PostgreSQL</option>
									<option value="mysql">MySQL</option>
									<option value="redis">Redis</option>
									<option value="mongodb">MongoDB</option>
								</select>
							</div>
							<div>
								<label for="monitor-db-conn" class={labelClass}>Connection String</label>
								<input
									id="monitor-db-conn"
									type="text"
									bind:value={dbConnectionString}
									placeholder="host=localhost port=5432 dbname=mydb user=postgres"
									class={inputClass}
								/>
							</div>
							<div>
								<label for="monitor-db-pass" class={labelClass}>Password</label>
								<input
									id="monitor-db-pass"
									type="password"
									bind:value={dbPassword}
									placeholder="Database password"
									class={inputClass}
								/>
							</div>
						</div>
					{/if}

					{#if type === 'system'}
						<div class="space-y-3 pt-1">
							<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">System Metric</div>
							<div class="grid grid-cols-2 gap-3">
								<div>
									<label for="monitor-sys-metric" class={labelClass}>Metric</label>
									<select
										id="monitor-sys-metric"
										bind:value={systemMetric}
										class={inputClass}
									>
										<option value="cpu">CPU</option>
										<option value="memory">Memory</option>
										<option value="disk">Disk</option>
									</select>
								</div>
								<div>
									<label for="monitor-sys-threshold" class={labelClass}>Threshold %</label>
									<input
										id="monitor-sys-threshold"
										type="number"
										bind:value={systemThreshold}
										min="1"
										max="100"
										class={inputClass}
									/>
								</div>
							</div>
						</div>
					{/if}
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
						{loading ? 'Creating...' : 'Create Monitor'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
