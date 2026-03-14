<script lang="ts">
	import { Radio, CheckCircle2, XCircle } from 'lucide-svelte';

	interface Props {
		metadata: Record<string, string> | undefined;
	}

	let { metadata }: Props = $props();

	// Single GET result
	let snmpValue = $derived(metadata?.snmp_value ?? '');
	let snmpType = $derived(metadata?.snmp_type ?? '');
	let snmpOid = $derived(metadata?.snmp_oid ?? '');

	// Multi/Walk/Bulk results
	let snmpResults = $derived.by(() => {
		const raw = metadata?.snmp_results;
		if (!raw) return [];
		return raw.split('|').map((pair) => {
			const eqIdx = pair.indexOf('=');
			if (eqIdx === -1) return { oid: pair, value: '' };
			return { oid: pair.slice(0, eqIdx), value: pair.slice(eqIdx + 1) };
		});
	});

	let resultCount = $derived(parseInt(metadata?.snmp_count ?? '0'));
	let hasSingleResult = $derived(!!snmpValue || !!snmpOid);
	let hasMultiResults = $derived(snmpResults.length > 0);
	let hasAnyData = $derived(hasSingleResult || hasMultiResults);
</script>

{#if metadata && hasAnyData}
	<div class="bg-card border border-border rounded-lg">
		<div class="px-4 sm:px-5 py-3 sm:py-3.5 border-b border-border flex items-center justify-between">
			<div class="flex items-center space-x-2">
				<Radio class="w-4 h-4 text-muted-foreground" />
				<h3 class="text-sm font-medium text-foreground">SNMP Results</h3>
			</div>
			{#if resultCount > 0}
				<span class="text-[10px] font-mono text-muted-foreground">{resultCount} values</span>
			{/if}
		</div>

		<div class="p-4 sm:p-5 space-y-4">
			<!-- Single GET result -->
			{#if hasSingleResult && !hasMultiResults}
				<div class="space-y-2">
					{#if snmpOid}
						<div class="flex items-center space-x-2">
							<span class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">OID</span>
							<span class="text-xs font-mono text-muted-foreground">{snmpOid}</span>
						</div>
					{/if}
					<div class="bg-card-elevated border border-border rounded-md px-4 py-3">
						<div class="flex items-center justify-between">
							<span class="text-sm font-mono text-foreground break-all">{snmpValue || '(empty)'}</span>
							{#if snmpType}
								<span class="text-[10px] px-1.5 py-0.5 bg-muted rounded font-mono text-muted-foreground ml-3 flex-shrink-0">{snmpType}</span>
							{/if}
						</div>
					</div>
				</div>
			{/if}

			<!-- Multi/Walk/Bulk results table -->
			{#if hasMultiResults}
				<div class="overflow-x-auto">
					<table class="w-full">
						<thead>
							<tr class="border-b border-border">
								<th class="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">OID</th>
								<th class="px-3 py-2 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Value</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border/30">
							{#each snmpResults as result}
								<tr class="hover:bg-card-elevated transition-colors">
									<td class="px-3 py-2">
										<span class="text-xs font-mono text-muted-foreground break-all">{result.oid}</span>
									</td>
									<td class="px-3 py-2">
										{#if result.value === 'NoSuchObject'}
											<div class="flex items-center space-x-1">
												<XCircle class="w-3 h-3 text-red-400" />
												<span class="text-xs font-mono text-red-400">No such object</span>
											</div>
										{:else if result.value}
											<span class="text-xs font-mono text-foreground break-all">{result.value}</span>
										{:else}
											<span class="text-xs text-muted-foreground">(empty)</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>
	</div>
{/if}
