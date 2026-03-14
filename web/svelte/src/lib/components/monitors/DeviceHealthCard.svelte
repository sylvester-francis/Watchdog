<script lang="ts">
	import { onMount } from 'svelte';
	import { Server, Cpu, HardDrive, Network } from 'lucide-svelte';
	import { getDeviceTemplate } from '$lib/api/monitors';
	import type { DeviceTemplate, OIDEntry } from '$lib/types';

	interface Props {
		metadata: Record<string, string> | undefined;
		templateId: string;
	}

	let { metadata, templateId }: Props = $props();

	let template = $state<DeviceTemplate | null>(null);
	let loading = $state(true);
	let error = $state('');

	// Well-known system OIDs
	const SYSTEM_OIDS: Record<string, string> = {
		'1.3.6.1.2.1.1.1': 'sysDescr',
		'1.3.6.1.2.1.1.3': 'sysUptime',
		'1.3.6.1.2.1.1.5': 'sysName',
		'1.3.6.1.2.1.1.6': 'sysLocation',
		'1.3.6.1.2.1.1.4': 'sysContact'
	};

	// Parse pipe-delimited SNMP results into a map
	let resultsMap = $derived.by(() => {
		const map = new Map<string, string>();
		const raw = metadata?.snmp_results;
		if (raw) {
			for (const pair of raw.split('|')) {
				const eqIdx = pair.indexOf('=');
				if (eqIdx !== -1) {
					map.set(pair.slice(0, eqIdx), pair.slice(eqIdx + 1));
				}
			}
		}
		// Also include the single snmp_value if present
		if (metadata?.snmp_oid && metadata?.snmp_value) {
			map.set(metadata.snmp_oid, metadata.snmp_value);
		}
		return map;
	});

	// Resolve a value by OID or by matching template OID name
	function findValue(nameOrOid: string): string | undefined {
		// Direct OID lookup
		const direct = resultsMap.get(nameOrOid);
		if (direct) return direct;

		// Check sub-OIDs (walk results often have .0 suffix)
		for (const [oid, val] of resultsMap) {
			if (oid.startsWith(nameOrOid + '.')) return val;
		}

		// Search by template OID name
		if (template?.oids) {
			const entry = template.oids.find((o) => o.name === nameOrOid);
			if (entry) {
				const val = resultsMap.get(entry.oid);
				if (val) return val;
				for (const [oid, v] of resultsMap) {
					if (oid.startsWith(entry.oid + '.')) return v;
				}
			}
		}

		return undefined;
	}

	// System info
	let sysDescr = $derived(findValue('1.3.6.1.2.1.1.1') ?? findValue('sysDescr') ?? '');
	let sysUptime = $derived(findValue('1.3.6.1.2.1.1.3') ?? findValue('sysUptime') ?? '');
	let sysName = $derived(findValue('1.3.6.1.2.1.1.5') ?? findValue('sysName') ?? '');

	// Target IP from metadata
	let targetIp = $derived(metadata?.target ?? metadata?.snmp_target ?? metadata?.ip ?? '');

	// CPU utilization
	let cpuValue = $derived.by(() => {
		const raw = metadata?.cpu_utilization ?? findValue('cpuUtil') ?? findValue('cpu_utilization');
		if (!raw) return null;
		const val = parseFloat(raw);
		return isNaN(val) ? null : Math.min(val, 100);
	});

	// Memory
	let memUsed = $derived.by(() => {
		const raw = metadata?.memory_used ?? findValue('memUsed') ?? findValue('memory_used');
		return raw ? parseInt(raw) : null;
	});

	let memTotal = $derived.by(() => {
		const raw = metadata?.memory_total ?? findValue('memTotal') ?? findValue('memory_total');
		return raw ? parseInt(raw) : null;
	});

	let memPercent = $derived.by(() => {
		if (memUsed !== null && memTotal !== null && memTotal > 0) {
			return Math.min((memUsed / memTotal) * 100, 100);
		}
		// Try direct percentage
		const raw = metadata?.memory_utilization ?? findValue('memUtil') ?? findValue('memory_utilization');
		if (raw) {
			const val = parseFloat(raw);
			return isNaN(val) ? null : Math.min(val, 100);
		}
		return null;
	});

	// Interfaces - parse from results that match interface OIDs
	interface InterfaceInfo {
		index: string;
		name: string;
		status: 'up' | 'down' | 'unknown';
		inTraffic: string;
		outTraffic: string;
	}

	let interfaces = $derived.by(() => {
		const ifMap = new Map<string, Partial<InterfaceInfo>>();

		// ifDescr: 1.3.6.1.2.1.2.2.1.2
		// ifOperStatus: 1.3.6.1.2.1.2.2.1.8
		// ifInOctets: 1.3.6.1.2.1.2.2.1.10
		// ifOutOctets: 1.3.6.1.2.1.2.2.1.16
		const IF_DESCR = '1.3.6.1.2.1.2.2.1.2';
		const IF_OPER_STATUS = '1.3.6.1.2.1.2.2.1.8';
		const IF_IN_OCTETS = '1.3.6.1.2.1.2.2.1.10';
		const IF_OUT_OCTETS = '1.3.6.1.2.1.2.2.1.16';

		for (const [oid, val] of resultsMap) {
			let idx: string | null = null;
			let field: string | null = null;

			if (oid.startsWith(IF_DESCR + '.')) {
				idx = oid.slice(IF_DESCR.length + 1);
				field = 'name';
			} else if (oid.startsWith(IF_OPER_STATUS + '.')) {
				idx = oid.slice(IF_OPER_STATUS.length + 1);
				field = 'status';
			} else if (oid.startsWith(IF_IN_OCTETS + '.')) {
				idx = oid.slice(IF_IN_OCTETS.length + 1);
				field = 'inTraffic';
			} else if (oid.startsWith(IF_OUT_OCTETS + '.')) {
				idx = oid.slice(IF_OUT_OCTETS.length + 1);
				field = 'outTraffic';
			}

			if (idx && field) {
				if (!ifMap.has(idx)) ifMap.set(idx, { index: idx });
				const entry = ifMap.get(idx)!;
				if (field === 'name') {
					entry.name = val;
				} else if (field === 'status') {
					entry.status = val === '1' ? 'up' : val === '2' ? 'down' : 'unknown';
				} else if (field === 'inTraffic') {
					entry.inTraffic = val;
				} else if (field === 'outTraffic') {
					entry.outTraffic = val;
				}
			}
		}

		// Also check for rate_* metadata keys
		for (const [key, val] of Object.entries(metadata ?? {})) {
			const inMatch = key.match(/^rate_ifInOctets_(\d+)$/);
			if (inMatch) {
				const idx = inMatch[1];
				if (!ifMap.has(idx)) ifMap.set(idx, { index: idx });
				ifMap.get(idx)!.inTraffic = val;
			}
			const outMatch = key.match(/^rate_ifOutOctets_(\d+)$/);
			if (outMatch) {
				const idx = outMatch[1];
				if (!ifMap.has(idx)) ifMap.set(idx, { index: idx });
				ifMap.get(idx)!.outTraffic = val;
			}
		}

		return Array.from(ifMap.values())
			.filter((i) => i.name || i.status)
			.map((i) => ({
				index: i.index ?? '',
				name: i.name ?? `Interface ${i.index}`,
				status: i.status ?? 'unknown',
				inTraffic: i.inTraffic ?? '',
				outTraffic: i.outTraffic ?? ''
			})) as InterfaceInfo[];
	});

	let hasCpu = $derived(cpuValue !== null);
	let hasMemory = $derived(memPercent !== null);
	let hasInterfaces = $derived(interfaces.length > 0);
	let hasSysInfo = $derived(!!sysDescr || !!sysUptime || !!sysName);

	function formatBytes(raw: string): string {
		const val = parseFloat(raw);
		if (isNaN(val)) return raw;
		if (val >= 1_000_000_000) return (val / 1_000_000_000).toFixed(2) + ' GB/s';
		if (val >= 1_000_000) return (val / 1_000_000).toFixed(2) + ' MB/s';
		if (val >= 1_000) return (val / 1_000).toFixed(2) + ' KB/s';
		return val.toFixed(0) + ' B/s';
	}

	function formatUptime(raw: string): string {
		// Timeticks are in hundredths of a second
		const ticks = parseInt(raw);
		if (isNaN(ticks)) return raw;
		const totalSeconds = Math.floor(ticks / 100);
		const days = Math.floor(totalSeconds / 86400);
		const hours = Math.floor((totalSeconds % 86400) / 3600);
		const minutes = Math.floor((totalSeconds % 3600) / 60);
		const parts: string[] = [];
		if (days > 0) parts.push(`${days}d`);
		if (hours > 0) parts.push(`${hours}h`);
		parts.push(`${minutes}m`);
		return parts.join(' ');
	}

	function formatMemory(bytes: number): string {
		if (bytes >= 1_073_741_824) return (bytes / 1_073_741_824).toFixed(1) + ' GB';
		if (bytes >= 1_048_576) return (bytes / 1_048_576).toFixed(1) + ' MB';
		if (bytes >= 1024) return (bytes / 1024).toFixed(1) + ' KB';
		return bytes + ' B';
	}

	function barColor(percent: number): string {
		if (percent >= 90) return 'bg-red-500';
		if (percent >= 70) return 'bg-amber-500';
		return 'bg-emerald-500';
	}

	onMount(async () => {
		try {
			const res = await getDeviceTemplate(templateId);
			template = res.data;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load template';
		} finally {
			loading = false;
		}
	});
</script>

{#if loading}
	<div class="bg-card border border-border rounded-lg p-5">
		<div class="flex items-center space-x-2 text-muted-foreground">
			<Server class="w-4 h-4 animate-pulse" />
			<span class="text-sm">Loading device health...</span>
		</div>
	</div>
{:else if error}
	<div class="bg-card border border-border rounded-lg p-5">
		<p class="text-sm text-muted-foreground">Unable to load device template: {error}</p>
	</div>
{:else}
	<div class="bg-card border border-border rounded-lg">
		<!-- Header -->
		<div class="px-5 py-3.5 border-b border-border flex items-center justify-between">
			<div class="flex items-center space-x-2">
				<Server class="w-4 h-4 text-muted-foreground" />
				<h3 class="text-sm font-medium text-foreground">
					{template ? `${template.vendor} ${template.model}` : 'Device Health'}
				</h3>
			</div>
			{#if targetIp}
				<span class="text-xs font-mono text-muted-foreground">{targetIp}</span>
			{/if}
		</div>

		<div class="p-5 space-y-5">
			<!-- System Info -->
			{#if hasSysInfo}
				<div class="space-y-2">
					<h4 class="text-xs font-medium text-muted-foreground uppercase tracking-wider">System</h4>
					<div class="grid grid-cols-1 gap-2">
						{#if sysName}
							<div class="flex items-baseline space-x-2">
								<span class="text-xs text-muted-foreground w-20 flex-shrink-0">Name</span>
								<span class="text-sm font-mono text-foreground">{sysName}</span>
							</div>
						{/if}
						{#if sysDescr}
							<div class="flex items-baseline space-x-2">
								<span class="text-xs text-muted-foreground w-20 flex-shrink-0">Description</span>
								<span class="text-sm font-mono text-foreground break-all">{sysDescr}</span>
							</div>
						{/if}
						{#if sysUptime}
							<div class="flex items-baseline space-x-2">
								<span class="text-xs text-muted-foreground w-20 flex-shrink-0">Uptime</span>
								<span class="text-sm font-mono text-foreground">{formatUptime(sysUptime)}</span>
							</div>
						{/if}
					</div>
				</div>
			{/if}

			<!-- CPU -->
			{#if hasCpu}
				<div class="space-y-2">
					<div class="flex items-center space-x-2">
						<Cpu class="w-3.5 h-3.5 text-muted-foreground" />
						<h4 class="text-xs font-medium text-muted-foreground uppercase tracking-wider">CPU</h4>
					</div>
					<div class="space-y-1">
						<div class="flex items-center justify-between">
							<span class="text-sm text-foreground">{cpuValue!.toFixed(1)}%</span>
						</div>
						<div class="w-full h-2 bg-muted rounded-full overflow-hidden">
							<div
								class="h-full rounded-full transition-all {barColor(cpuValue!)}"
								style="width: {cpuValue}%"
							></div>
						</div>
					</div>
				</div>
			{/if}

			<!-- Memory -->
			{#if hasMemory}
				<div class="space-y-2">
					<div class="flex items-center space-x-2">
						<HardDrive class="w-3.5 h-3.5 text-muted-foreground" />
						<h4 class="text-xs font-medium text-muted-foreground uppercase tracking-wider">Memory</h4>
					</div>
					<div class="space-y-1">
						<div class="flex items-center justify-between">
							<span class="text-sm text-foreground">{memPercent!.toFixed(1)}%</span>
							{#if memUsed !== null && memTotal !== null}
								<span class="text-xs text-muted-foreground">
									{formatMemory(memUsed)} / {formatMemory(memTotal)}
								</span>
							{/if}
						</div>
						<div class="w-full h-2 bg-muted rounded-full overflow-hidden">
							<div
								class="h-full rounded-full transition-all {barColor(memPercent!)}"
								style="width: {memPercent}%"
							></div>
						</div>
					</div>
				</div>
			{/if}

			<!-- Interfaces -->
			{#if hasInterfaces}
				<div class="space-y-2">
					<div class="flex items-center space-x-2">
						<Network class="w-3.5 h-3.5 text-muted-foreground" />
						<h4 class="text-xs font-medium text-muted-foreground uppercase tracking-wider">Interfaces</h4>
					</div>
					<div class="overflow-x-auto">
						<table class="w-full">
							<thead>
								<tr class="border-b border-border">
									<th class="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Name</th>
									<th class="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Status</th>
									<th class="px-3 py-2 text-right text-[10px] font-medium text-muted-foreground uppercase tracking-wider">In</th>
									<th class="px-3 py-2 text-right text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Out</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-border/30">
								{#each interfaces as iface}
									<tr class="hover:bg-card-elevated transition-colors">
										<td class="px-3 py-2">
											<span class="text-xs font-mono text-foreground">{iface.name}</span>
										</td>
										<td class="px-3 py-2">
											<div class="flex items-center space-x-1.5">
												<span
													class="w-2 h-2 rounded-full {iface.status === 'up'
														? 'bg-emerald-500'
														: iface.status === 'down'
															? 'bg-red-500'
															: 'bg-amber-500'}"
												></span>
												<span class="text-xs text-muted-foreground">{iface.status}</span>
											</div>
										</td>
										<td class="px-3 py-2 text-right">
											{#if iface.inTraffic}
												<span class="text-xs font-mono text-foreground">{formatBytes(iface.inTraffic)}</span>
											{:else}
												<span class="text-xs text-muted-foreground">N/A</span>
											{/if}
										</td>
										<td class="px-3 py-2 text-right">
											{#if iface.outTraffic}
												<span class="text-xs font-mono text-foreground">{formatBytes(iface.outTraffic)}</span>
											{:else}
												<span class="text-xs text-muted-foreground">N/A</span>
											{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			{/if}

			<!-- Fallback: no structured data available -->
			{#if !hasSysInfo && !hasCpu && !hasMemory && !hasInterfaces}
				<p class="text-sm text-muted-foreground">No device health data available yet.</p>
			{/if}
		</div>
	</div>
{/if}
