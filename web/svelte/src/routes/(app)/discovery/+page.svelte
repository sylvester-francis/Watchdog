<script lang="ts">
	import { onMount } from 'svelte';
	import { Radar, CheckCircle2, XCircle, Wifi, WifiOff, Loader2 } from 'lucide-svelte';
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

	// Form state
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

			// Check for any running scan
			const running = scans.find(s => s.status === 'running' || s.status === 'pending');
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
					// Reload scan list
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

	onMount(() => {
		loadData();
		return () => {
			if (pollInterval) clearInterval(pollInterval);
		};
	});
</script>

<svelte:head>
	<title>Discovery - WatchDog</title>
</svelte:head>

<div class="animate-fade-in-up space-y-5">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div class="flex items-center space-x-3">
			<div class="w-8 h-8 bg-muted/50 rounded-lg flex items-center justify-center">
				<Radar class="w-4 h-4 text-muted-foreground" />
			</div>
			<div>
				<h1 class="text-base font-semibold text-foreground">Network Discovery</h1>
				<p class="text-xs text-muted-foreground">Scan your network to find SNMP-capable devices</p>
			</div>
		</div>
	</div>

	<!-- Start Scan Card -->
	<div class="bg-card border border-border rounded-lg">
		<div class="px-5 py-3.5 border-b border-border">
			<h3 class="text-sm font-medium text-foreground">Start Scan</h3>
		</div>
		<div class="p-5 space-y-4">
			<div class="grid grid-cols-2 gap-3">
				<div>
					<label for="disc-agent" class="block text-xs font-medium text-muted-foreground mb-1.5">Agent</label>
					<select
						id="disc-agent"
						bind:value={selectedAgentId}
						class="w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
					>
						<option value="" disabled>Select agent</option>
						{#each agents as agent}
							<option value={agent.id}>{agent.name} ({agent.status})</option>
						{/each}
					</select>
				</div>
				<div>
					<label for="disc-subnet" class="block text-xs font-medium text-muted-foreground mb-1.5">Subnet (CIDR)</label>
					<input
						id="disc-subnet"
						type="text"
						bind:value={subnet}
						placeholder="192.168.1.0/24"
						class="w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
					/>
				</div>
			</div>
			<div class="grid grid-cols-2 gap-3">
				<div>
					<label for="disc-community" class="block text-xs font-medium text-muted-foreground mb-1.5">Community String</label>
					<input
						id="disc-community"
						type="password"
						bind:value={community}
						placeholder="public"
						class="w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
					/>
				</div>
				<div>
					<label for="disc-version" class="block text-xs font-medium text-muted-foreground mb-1.5">SNMP Version</label>
					<select
						id="disc-version"
						bind:value={snmpVersion}
						class="w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
					>
						<option value="2c">v2c</option>
						<option value="3">v3</option>
					</select>
				</div>
			</div>
			<button
				onclick={handleStartScan}
				disabled={scanning || !selectedAgentId || !subnet}
				class="px-4 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors disabled:opacity-50 flex items-center space-x-2"
			>
				{#if scanning}
					<Loader2 class="w-3.5 h-3.5 animate-spin" />
					<span>Scanning...</span>
				{:else}
					<Radar class="w-3.5 h-3.5" />
					<span>Start Scan</span>
				{/if}
			</button>
		</div>
	</div>

	<!-- Active Scan Progress -->
	{#if activeScan && (activeScan.status === 'running' || activeScan.status === 'pending')}
		<div class="bg-card border border-border rounded-lg p-5">
			<div class="flex items-center space-x-3 mb-3">
				<Loader2 class="w-4 h-4 animate-spin text-accent" />
				<span class="text-sm font-medium text-foreground">Scanning {activeScan.subnet}...</span>
				<span class="text-xs text-muted-foreground">{activeDevices.length} devices found</span>
			</div>
			<div class="w-full h-2 bg-muted rounded-full overflow-hidden">
				<div class="h-full bg-accent rounded-full transition-all duration-500" style="width: {activeScan.host_count ? Math.min((activeDevices.length / activeScan.host_count) * 100, 100) : 10}%"></div>
			</div>
		</div>
	{/if}

	<!-- Results Table -->
	{#if activeDevices.length > 0}
		<div class="bg-card border border-border rounded-lg">
			<div class="px-5 py-3.5 border-b border-border flex items-center justify-between">
				<h3 class="text-sm font-medium text-foreground">Discovered Devices ({activeDevices.length})</h3>
				{#if activeScan}
					<span class="text-xs text-muted-foreground">{activeScan.subnet}</span>
				{/if}
			</div>
			<div class="overflow-x-auto">
				<table class="w-full">
					<thead>
						<tr class="text-left text-[10px] uppercase tracking-wider text-muted-foreground border-b border-border/50">
							<th class="px-5 py-2.5">IP</th>
							<th class="px-3 py-2.5">Name</th>
							<th class="px-3 py-2.5">SNMP</th>
							<th class="px-3 py-2.5">Ping</th>
							<th class="px-3 py-2.5">Template</th>
						</tr>
					</thead>
					<tbody>
						{#each activeDevices as device}
							<tr class="border-b border-border/20 hover:bg-muted/20 transition-colors">
								<td class="px-5 py-2.5 text-xs font-mono text-foreground">{device.ip}</td>
								<td class="px-3 py-2.5 text-xs text-foreground">{device.sys_name || device.hostname || '—'}</td>
								<td class="px-3 py-2.5">
									{#if device.snmp_reachable}
										<Wifi class="w-3.5 h-3.5 text-emerald-500" />
									{:else}
										<WifiOff class="w-3.5 h-3.5 text-muted-foreground/40" />
									{/if}
								</td>
								<td class="px-3 py-2.5">
									{#if device.ping_reachable}
										<CheckCircle2 class="w-3.5 h-3.5 text-emerald-500" />
									{:else}
										<XCircle class="w-3.5 h-3.5 text-muted-foreground/40" />
									{/if}
								</td>
								<td class="px-3 py-2.5 text-xs text-muted-foreground">{device.suggested_template_id || '—'}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}

	<!-- Scan History -->
	{#if scans.length > 0}
		<div class="bg-card border border-border rounded-lg">
			<div class="px-5 py-3.5 border-b border-border">
				<h3 class="text-sm font-medium text-foreground">Scan History</h3>
			</div>
			<div class="divide-y divide-border/20">
				{#each scans as scan}
					<button
						onclick={() => viewScan(scan.id)}
						class="w-full px-5 py-3 flex items-center justify-between hover:bg-muted/20 transition-colors text-left"
					>
						<div class="flex items-center space-x-3">
							<div class="w-2 h-2 rounded-full {scan.status === 'complete' ? 'bg-emerald-500' : scan.status === 'error' ? 'bg-red-500' : scan.status === 'running' ? 'bg-amber-500 animate-pulse' : 'bg-muted-foreground/40'}"></div>
							<span class="text-xs font-mono text-foreground">{scan.subnet}</span>
						</div>
						<div class="flex items-center space-x-4">
							<span class="text-[10px] text-muted-foreground">{scan.host_count} hosts</span>
							<span class="text-[10px] text-muted-foreground">{new Date(scan.created_at).toLocaleDateString()}</span>
						</div>
					</button>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Empty state -->
	{#if !loading && scans.length === 0 && activeDevices.length === 0}
		<div class="bg-card border border-border rounded-lg p-8 text-center">
			<Radar class="w-8 h-8 text-muted-foreground/30 mx-auto mb-3" />
			<p class="text-sm text-foreground font-medium mb-1">No scans yet</p>
			<p class="text-xs text-muted-foreground">Select an agent and subnet above to discover SNMP devices on your network</p>
		</div>
	{/if}
</div>
