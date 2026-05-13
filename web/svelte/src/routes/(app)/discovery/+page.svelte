<script lang="ts">
	import { onMount } from 'svelte';
	import { Radar, Loader2 } from 'lucide-svelte';
	import { discovery as discoveryApi, agents as agentsApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast.svelte';
	import type { Agent, DiscoveryScan, DiscoveredDevice } from '$lib/types';

	const toast = getToasts();

	let agents = $state<Agent[]>([]);
	let scans = $state<DiscoveryScan[]>([]);
	let activeDevices = $state<DiscoveredDevice[]>([]);
	let activeScan = $state<DiscoveryScan | null>(null);
	let loading = $state(true);
	let scanning = $state(false);

	let selectedAgentId = $state('');
	let subnet = $state('');
	let community = $state('public');
	let snmpVersion = $state('2c');

	let pollInterval: ReturnType<typeof setInterval> | null = null;

	async function loadData() {
		loading = true;
		try {
			const [agentsRes, scansRes] = await Promise.all([
				agentsApi.listAgents(),
				discoveryApi.listScans()
			]);
			agents = agentsRes.data ?? [];
			scans = scansRes.data ?? [];

			const running = scans.find((s) => s.status === 'running' || s.status === 'pending');
			if (running) {
				activeScan = running;
				scanning = true;
				startPolling(running.id);
			}
		} catch {
			toast.error('Failed to load discovery data');
		} finally {
			loading = false;
		}
	}

	function startPolling(scanId: string) {
		if (pollInterval) clearInterval(pollInterval);
		pollInterval = setInterval(async () => {
			try {
				const res = await discoveryApi.getScan(scanId);
				activeScan = res.data.scan;
				activeDevices = res.data.devices ?? [];

				if (activeScan.status === 'complete' || activeScan.status === 'error') {
					scanning = false;
					if (pollInterval) clearInterval(pollInterval);
					pollInterval = null;
					const scansRes = await discoveryApi.listScans();
					scans = scansRes.data ?? [];
					if (activeScan.status === 'complete') {
						toast.success(`Discovery complete: ${activeDevices.length} devices found`);
					} else {
						toast.error(`Discovery failed: ${activeScan.error_message}`);
					}
				}
			} catch {
				// Ignore poll errors
			}
		}, 3000);
	}

	async function handleStartScan() {
		if (!selectedAgentId || !subnet) return;

		scanning = true;
		try {
			const res = await discoveryApi.startScan({
				agent_id: selectedAgentId,
				subnet: subnet.trim(),
				community: community.trim(),
				snmp_version: snmpVersion
			});
			activeScan = { id: res.data.id, status: 'pending', subnet: res.data.subnet } as DiscoveryScan;
			activeDevices = [];
			startPolling(res.data.id);
			toast.success('Discovery scan started');
		} catch (err) {
			scanning = false;
			toast.error(err instanceof Error ? err.message : 'Failed to start scan');
		}
	}

	async function viewScan(scanId: string) {
		try {
			const res = await discoveryApi.getScan(scanId);
			activeScan = res.data.scan;
			activeDevices = res.data.devices ?? [];
		} catch {
			toast.error('Failed to load scan details');
		}
	}

	function scanStatusPipClass(status: string): string {
		if (status === 'complete') return 'bg-success';
		if (status === 'error') return 'bg-destructive';
		if (status === 'running') return 'bg-warning animate-pulse';
		return 'bg-muted-foreground/40';
	}

	onMount(() => {
		loadData();
		return () => {
			if (pollInterval) clearInterval(pollInterval);
		};
	});

	const inputClass =
		'w-full border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground/50 focus:border-foreground/50 focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-foreground/30';
</script>

<svelte:head>
	<title>Discovery - WatchDog</title>
</svelte:head>

<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-6 sm:px-6 sm:py-10">
	<!-- Header -->
	<header>
		<div class="flex items-center gap-2 font-mono tabular-nums text-xs text-muted-foreground">
			<span class="uppercase tracking-wider">Network · Discovery</span>
		</div>
		<h1 class="mt-1.5 text-xl font-medium text-foreground sm:text-2xl md:text-3xl">Network Discovery</h1>
		<p class="mt-1 text-sm text-muted-foreground">Scan your network to find SNMP-capable devices.</p>
	</header>

	<!-- Start Scan -->
	<section class="mt-8">
		<div class="border-b border-border pb-3">
			<h3 class="text-sm font-medium text-foreground">Start Scan</h3>
		</div>
		<div class="space-y-4 pt-4">
			<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
				<div>
					<label for="disc-agent" class="mb-1.5 block text-xs font-medium text-muted-foreground">Agent</label>
					<select id="disc-agent" bind:value={selectedAgentId} class={inputClass}>
						<option value="" disabled>Select agent</option>
						{#each agents as agent}
							<option value={agent.id}>{agent.name} ({agent.status})</option>
						{/each}
					</select>
				</div>
				<div>
					<label for="disc-subnet" class="mb-1.5 block text-xs font-medium text-muted-foreground">Subnet (CIDR)</label>
					<input
						id="disc-subnet"
						type="text"
						bind:value={subnet}
						placeholder="192.168.1.0/24"
						class={inputClass}
					/>
				</div>
			</div>
			<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
				<div>
					<label for="disc-community" class="mb-1.5 block text-xs font-medium text-muted-foreground">Community String</label>
					<input
						id="disc-community"
						type="password"
						bind:value={community}
						placeholder="public"
						class={inputClass}
					/>
				</div>
				<div>
					<label for="disc-version" class="mb-1.5 block text-xs font-medium text-muted-foreground">SNMP Version</label>
					<select id="disc-version" bind:value={snmpVersion} class={inputClass}>
						<option value="2c">v2c</option>
						<option value="3">v3</option>
					</select>
				</div>
			</div>
			<button
				onclick={handleStartScan}
				disabled={scanning || !selectedAgentId || !subnet}
				class="flex items-center gap-2 bg-accent px-3 py-1.5 text-xs font-medium text-background transition-opacity hover:opacity-90 disabled:opacity-50"
			>
				{#if scanning}
					<Loader2 class="h-3.5 w-3.5 animate-spin" />
					<span>Scanning…</span>
				{:else}
					<Radar class="h-3.5 w-3.5" />
					<span>Start Scan</span>
				{/if}
			</button>
		</div>
	</section>

	<!-- Active Scan Progress -->
	{#if activeScan && (activeScan.status === 'running' || activeScan.status === 'pending')}
		<section class="mt-10">
			<div class="border-b border-border pb-3">
				<h3 class="text-sm font-medium text-foreground">Active Scan</h3>
			</div>
			<div class="pt-4">
				<div class="mb-3 flex items-center gap-3 font-mono tabular-nums text-xs">
					<Loader2 class="h-3.5 w-3.5 animate-spin text-accent" />
					<span class="text-foreground">Scanning {activeScan.subnet}…</span>
					<span class="text-muted-foreground">{activeDevices.length} devices found</span>
				</div>
				<div class="h-1.5 w-full overflow-hidden bg-muted">
					<div
						class="h-full bg-accent transition-all duration-500"
						style="width: {activeScan.host_count ? Math.min((activeDevices.length / activeScan.host_count) * 100, 100) : 10}%"
					></div>
				</div>
			</div>
		</section>
	{/if}

	<!-- Results Table -->
	{#if activeDevices.length > 0}
		<section class="mt-10">
			<div class="flex items-baseline justify-between gap-2 border-b border-border pb-3">
				<div class="flex items-baseline gap-2">
					<h3 class="text-sm font-medium text-foreground">Discovered Devices</h3>
					<span class="font-mono tabular-nums text-[11px] text-muted-foreground">{activeDevices.length}</span>
				</div>
				{#if activeScan}
					<span class="font-mono tabular-nums text-[11px] text-muted-foreground">{activeScan.subnet}</span>
				{/if}
			</div>
			<div class="overflow-x-auto">
				<table class="w-full">
					<thead>
						<tr class="border-b border-border">
							<th class="py-2.5 pl-1 pr-4 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">IP</th>
							<th class="px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Name</th>
							<th class="px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">SNMP</th>
							<th class="px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Ping</th>
							<th class="px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Template</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border/40">
						{#each activeDevices as device}
							<tr class="transition-colors hover:bg-muted/30">
								<td class="py-3 pl-1 pr-4 font-mono tabular-nums text-xs text-foreground">{device.ip}</td>
								<td class="px-4 py-3 text-xs text-foreground">{device.sys_name || device.hostname || '—'}</td>
								<td class="px-4 py-3 font-mono tabular-nums text-[11px] uppercase tracking-wider {device.snmp_reachable ? 'text-success' : 'text-muted-foreground/50'}">
									{device.snmp_reachable ? 'Yes' : 'No'}
								</td>
								<td class="px-4 py-3 font-mono tabular-nums text-[11px] uppercase tracking-wider {device.ping_reachable ? 'text-success' : 'text-muted-foreground/50'}">
									{device.ping_reachable ? 'Yes' : 'No'}
								</td>
								<td class="px-4 py-3 font-mono tabular-nums text-xs text-muted-foreground">{device.suggested_template_id || '—'}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</section>
	{/if}

	<!-- Scan History -->
	{#if scans.length > 0}
		<section class="mt-10">
			<div class="border-b border-border pb-3">
				<h3 class="text-sm font-medium text-foreground">Scan History</h3>
			</div>
			<div class="divide-y divide-border/40">
				{#each scans as scan}
					<button
						onclick={() => viewScan(scan.id)}
						class="flex w-full items-center justify-between py-3 text-left transition-colors hover:bg-muted/30"
					>
						<div class="flex items-center gap-2">
							<span class="inline-block h-1.5 w-1.5 rounded-full {scanStatusPipClass(scan.status)}"></span>
							<span class="font-mono tabular-nums text-xs text-foreground">{scan.subnet}</span>
						</div>
						<div class="flex items-center gap-4 font-mono tabular-nums text-[11px] text-muted-foreground">
							<span>{scan.host_count} hosts</span>
							<span>{new Date(scan.created_at).toLocaleDateString()}</span>
						</div>
					</button>
				{/each}
			</div>
		</section>
	{/if}

	<!-- Empty state -->
	{#if !loading && scans.length === 0 && activeDevices.length === 0}
		<section class="mt-10">
			<div class="border-b border-border pb-3">
				<h3 class="text-sm font-medium text-foreground">No scans yet</h3>
			</div>
			<p class="pt-4 text-xs text-muted-foreground">
				Select an agent and subnet above to discover SNMP devices on your network.
			</p>
		</section>
	{/if}
</div>
