<script lang="ts">
	import { Shield, Lock, AlertTriangle, CheckCircle, XCircle } from 'lucide-svelte';
	import { getCertDetails } from '$lib/api/monitors';
	import type { CertDetails } from '$lib/types';
	import { onMount } from 'svelte';

	interface Props {
		monitorId: string;
	}

	let { monitorId }: Props = $props();
	let cert = $state<CertDetails | null>(null);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		try {
			const res = await getCertDetails(monitorId);
			cert = res.data;
		} catch {
			error = 'No certificate data available';
		} finally {
			loading = false;
		}
	});

	let expiryColor = $derived(() => {
		if (!cert?.expiry_days) return 'text-muted-foreground';
		if (cert.expiry_days < 14) return 'text-destructive';
		if (cert.expiry_days < 30) return 'text-yellow-500';
		return 'text-green-500';
	});

	let expiryBg = $derived(() => {
		if (!cert?.expiry_days) return 'bg-muted';
		if (cert.expiry_days < 14) return 'bg-destructive/10';
		if (cert.expiry_days < 30) return 'bg-yellow-500/10';
		return 'bg-green-500/10';
	});
</script>

{#if loading}
	<div class="bg-card border border-border rounded-lg p-6">
		<div class="animate-pulse space-y-3">
			<div class="h-4 bg-muted rounded w-1/3"></div>
			<div class="h-3 bg-muted rounded w-2/3"></div>
			<div class="h-3 bg-muted rounded w-1/2"></div>
		</div>
	</div>
{:else if error}
	<!-- silently hide if no cert data -->
{:else if cert}
	<div class="bg-card border border-border rounded-lg p-6">
		<div class="flex items-center justify-between mb-4">
			<div class="flex items-center space-x-2">
				<Shield class="w-4 h-4 text-muted-foreground" />
				<h3 class="text-sm font-semibold text-foreground">Certificate Details</h3>
			</div>
			<span class="text-[10px] text-muted-foreground">
				Checked {new Date(cert.last_checked_at).toLocaleString()}
			</span>
		</div>

		<div class="grid grid-cols-2 lg:grid-cols-3 gap-4">
			<!-- Expiry -->
			<div class="space-y-1">
				<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Expires In</p>
				<div class="flex items-center space-x-1.5">
					{#if cert.expiry_days !== null && cert.expiry_days < 14}
						<AlertTriangle class="w-3.5 h-3.5 text-destructive" />
					{/if}
					<p class="text-lg font-semibold font-mono {expiryColor()}">
						{cert.expiry_days !== null ? `${cert.expiry_days}d` : '—'}
					</p>
				</div>
			</div>

			<!-- Issuer -->
			<div class="space-y-1">
				<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Issuer</p>
				<p class="text-sm text-foreground truncate">{cert.issuer || '—'}</p>
			</div>

			<!-- Algorithm -->
			<div class="space-y-1">
				<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Algorithm</p>
				<p class="text-sm text-foreground font-mono">{cert.algorithm}</p>
			</div>

			<!-- Key Size -->
			<div class="space-y-1">
				<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Key Size</p>
				<p class="text-sm text-foreground font-mono">{cert.key_size} bit</p>
			</div>

			<!-- Chain Valid -->
			<div class="space-y-1">
				<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Chain</p>
				<div class="flex items-center space-x-1.5">
					{#if cert.chain_valid}
						<CheckCircle class="w-3.5 h-3.5 text-green-500" />
						<span class="text-sm text-green-500 font-medium">Valid</span>
					{:else}
						<XCircle class="w-3.5 h-3.5 text-destructive" />
						<span class="text-sm text-destructive font-medium">Invalid</span>
					{/if}
				</div>
			</div>

			<!-- Serial -->
			<div class="space-y-1">
				<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Serial</p>
				<p class="text-xs text-muted-foreground font-mono truncate" title={cert.serial_number}>
					{cert.serial_number.length > 20 ? cert.serial_number.slice(0, 20) + '...' : cert.serial_number}
				</p>
			</div>
		</div>

		<!-- SANs -->
		{#if cert.sans && cert.sans.length > 0}
			<div class="mt-4 pt-3 border-t border-border">
				<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider mb-2">Subject Alternative Names</p>
				<div class="flex flex-wrap gap-1.5">
					{#each cert.sans as san}
						<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-mono bg-muted text-muted-foreground">
							<Lock class="w-2.5 h-2.5 mr-1" />
							{san}
						</span>
					{/each}
				</div>
			</div>
		{/if}
	</div>
{/if}
