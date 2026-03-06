<script lang="ts">
	import { Network, AlertTriangle, ShieldCheck, ShieldAlert } from 'lucide-svelte';
	import type { PortDetail } from '$lib/types';

	interface Props {
		metadata: Record<string, string> | undefined;
	}

	let { metadata }: Props = $props();

	let openPorts = $derived(metadata?.open_ports?.split(',').filter(Boolean) ?? []);
	let closedPorts = $derived(metadata?.closed_ports?.split(',').filter(Boolean) ?? []);
	let missingPorts = $derived(metadata?.missing_ports?.split(',').filter(Boolean) ?? []);
	let unexpectedPorts = $derived(metadata?.unexpected_ports?.split(',').filter(Boolean) ?? []);
	let expectedPorts = $derived(metadata?.expected_ports?.split(',').filter(Boolean) ?? []);
	let openCount = $derived(parseInt(metadata?.open_count ?? '0'));
	let scannedCount = $derived(parseInt(metadata?.scanned_count ?? '0'));
	let closedCount = $derived(scannedCount - openCount);
	let hasDrift = $derived(missingPorts.length > 0 || unexpectedPorts.length > 0);

	let openPercent = $derived(scannedCount > 0 ? (openCount / scannedCount) * 100 : 0);

	let portDetails: PortDetail[] = $derived.by(() => {
		const raw = metadata?.port_details;
		if (!raw) return [];
		try {
			return JSON.parse(raw) as PortDetail[];
		} catch {
			return [];
		}
	});

	let portDetailMap = $derived(
		new Map(portDetails.map((pd) => [String(pd.port), pd]))
	);

	let hasServiceData = $derived(portDetails.some((pd) => pd.service));

	let unknownServicePorts = $derived(
		portDetails.filter((pd) => !pd.service).map((pd) => String(pd.port))
	);
</script>

{#if metadata && (metadata.open_ports !== undefined || metadata.scanned_count !== undefined)}
	<div class="bg-card border border-border rounded-lg">
		<div class="px-4 sm:px-5 py-3 sm:py-3.5 border-b border-border flex flex-col sm:flex-row sm:items-center justify-between gap-2 sm:gap-0">
			<div class="flex items-center space-x-2">
				<Network class="w-4 h-4 text-muted-foreground" />
				<h3 class="text-sm font-medium text-foreground">Port Scan Results</h3>
				{#if hasServiceData}
					<span class="text-[10px] px-1.5 py-0.5 bg-blue-500/10 text-blue-400 rounded font-medium">Services</span>
				{/if}
			</div>
			<div class="flex items-center space-x-3">
				{#if hasDrift}
					<div class="flex items-center space-x-1">
						<ShieldAlert class="w-3.5 h-3.5 text-red-400" />
						<span class="text-xs font-medium text-red-400">Drift Detected</span>
					</div>
				{:else}
					<div class="flex items-center space-x-1">
						<ShieldCheck class="w-3.5 h-3.5 text-emerald-400" />
						<span class="text-xs font-medium text-emerald-400">Clean</span>
					</div>
				{/if}
			</div>
		</div>

		<div class="p-4 sm:p-5 space-y-4 sm:space-y-5">
			{#if scannedCount > 0}
				<div>
					<div class="flex items-center justify-between mb-2">
						<div class="flex items-center space-x-3">
							<div class="flex items-center space-x-1.5">
								<div class="w-2 h-2 rounded-full bg-emerald-400"></div>
								<span class="text-[10px] text-muted-foreground font-mono">{openCount} open</span>
							</div>
							<div class="flex items-center space-x-1.5">
								<div class="w-2 h-2 rounded-full bg-muted-foreground/30"></div>
								<span class="text-[10px] text-muted-foreground font-mono">{closedCount} closed</span>
							</div>
						</div>
						<span class="text-[10px] font-mono text-muted-foreground">{scannedCount} scanned</span>
					</div>
					<div class="w-full h-2.5 bg-muted rounded-full overflow-hidden flex">
						{#if openCount > 0}
							<div
								class="bg-emerald-500 h-full rounded-l-full {closedCount === 0 ? 'rounded-r-full' : ''}"
								style="width: {openPercent}%"
							></div>
						{/if}
						{#if closedCount > 0}
							<div
								class="bg-muted-foreground/20 h-full {openCount === 0 ? 'rounded-l-full' : ''} rounded-r-full"
								style="width: {100 - openPercent}%"
							></div>
						{/if}
					</div>
				</div>
			{/if}

			{#if hasDrift}
				<div class="bg-red-500/10 border border-red-500/20 rounded-md px-3 sm:px-3.5 py-2 sm:py-2.5 flex items-start space-x-2">
					<AlertTriangle class="w-3.5 h-3.5 text-red-400 mt-0.5 flex-shrink-0" />
					<div class="text-xs text-red-400 space-y-0.5">
						{#if missingPorts.length > 0}
							<p>Expected ports not open: <span class="font-mono font-medium">{missingPorts.join(', ')}</span></p>
						{/if}
						{#if unexpectedPorts.length > 0}
							<p>Unexpected ports open: <span class="font-mono font-medium">{unexpectedPorts.join(', ')}</span></p>
						{/if}
					</div>
				</div>
			{/if}

			{#if openPorts.length > 0 || missingPorts.length > 0 || unexpectedPorts.length > 0 || closedPorts.length > 0}
				<div class="space-y-3">
					{#if openPorts.length > 0}
						<div>
							<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium mb-2">Open</div>
							<div class="flex flex-wrap gap-1.5">
								{#each openPorts as port}
									{@const detail = portDetailMap.get(port)}
									<span
										class="px-2 py-0.5 text-xs font-mono rounded-md bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 inline-flex items-center gap-1"
										title={detail?.banner || ''}
									>
										{port}
										{#if detail?.service}
											<span class="text-emerald-400/70 text-[10px]">{detail.service}{#if detail.version}/{detail.version}{/if}</span>
										{/if}
									</span>
								{/each}
							</div>
						</div>
					{/if}

					{#if missingPorts.length > 0}
						<div>
							<div class="text-[10px] uppercase tracking-wider text-red-400/80 font-medium mb-2">Missing (Expected Open)</div>
							<div class="flex flex-wrap gap-1.5">
								{#each missingPorts as port}
									<span class="px-2 py-0.5 text-xs font-mono rounded-md bg-red-500/10 text-red-400 border border-red-500/20">
										{port}
									</span>
								{/each}
							</div>
						</div>
					{/if}

					{#if unexpectedPorts.length > 0}
						<div>
							<div class="text-[10px] uppercase tracking-wider text-yellow-400/80 font-medium mb-2">Unexpected</div>
							<div class="flex flex-wrap gap-1.5">
								{#each unexpectedPorts as port}
									{@const detail = portDetailMap.get(port)}
									<span
										class="px-2 py-0.5 text-xs font-mono rounded-md bg-yellow-500/10 text-yellow-400 border border-yellow-500/20 inline-flex items-center gap-1"
										title={detail?.banner || ''}
									>
										{port}
										{#if detail?.service}
											<span class="text-yellow-400/70 text-[10px]">{detail.service}</span>
										{/if}
									</span>
								{/each}
							</div>
						</div>
					{/if}

					{#if unknownServicePorts.length > 0 && hasServiceData}
						<div>
							<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium mb-2">Unknown Service</div>
							<div class="flex flex-wrap gap-1.5">
								{#each unknownServicePorts as port}
									<span class="px-2 py-0.5 text-xs font-mono rounded-md bg-muted/50 text-muted-foreground border border-border">
										{port}
									</span>
								{/each}
							</div>
						</div>
					{/if}

					{#if closedPorts.length > 0}
						<div>
							<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium mb-2">Closed</div>
							<div class="flex flex-wrap gap-1.5">
								{#each closedPorts as port}
									<span class="px-2 py-0.5 text-xs font-mono rounded-md bg-muted/50 text-muted-foreground border border-border">
										{port}
									</span>
								{/each}
							</div>
						</div>
					{/if}
				</div>
			{/if}

			{#if expectedPorts.length > 0}
				<div class="pt-3 border-t border-border/50">
					<p class="text-[10px] text-muted-foreground">
						<span class="uppercase tracking-wider font-medium">Expected:</span>
						<span class="font-mono ml-1">{expectedPorts.join(', ')}</span>
					</p>
				</div>
			{/if}
		</div>
	</div>
{/if}
