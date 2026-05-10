<script lang="ts">
	import { FormField } from '@sylvester-francis/watchdog-ui';
	import { Input } from '@sylvester-francis/watchdog-ui';

	interface Props {
		portScanPorts: string;
		portScanRange: string;
		portScanExpectedOpen: string;
		bannerGrab: boolean;
	}

	let {
		portScanPorts = $bindable(),
		portScanRange = $bindable(),
		portScanExpectedOpen = $bindable(),
		bannerGrab = $bindable(),
	}: Props = $props();
</script>

<div class="space-y-3 pt-1">
	<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">Port Scan Settings</div>
	<FormField label="Ports (comma-separated)" htmlFor="monitor-ps-ports">
		<Input id="monitor-ps-ports" type="text" bind:value={portScanPorts} placeholder="22,80,443,3306,8080" />
		<p class="text-[10px] text-muted-foreground mt-1">Supports ranges: 8000-9000</p>
	</FormField>
	<FormField label="Port Range (alternative)" htmlFor="monitor-ps-range">
		<Input id="monitor-ps-range" type="text" bind:value={portScanRange} placeholder="1-1024" />
	</FormField>
	<FormField label="Expected Open Ports (optional)" htmlFor="monitor-ps-expected">
		<Input id="monitor-ps-expected" type="text" bind:value={portScanExpectedOpen} placeholder="22,80,443" />
		<p class="text-[10px] text-muted-foreground mt-1">If set, alerts when expected ports close or unexpected ports open</p>
	</FormField>
	<div class="flex items-center justify-between pt-1">
		<div>
			<label for="monitor-banner-grab" class="block text-xs font-medium text-muted-foreground mb-1.5">Service Detection</label>
			<p class="text-[10px] text-muted-foreground">Identify services and versions on open ports</p>
		</div>
		<label class="relative inline-flex items-center cursor-pointer">
			<input
				id="monitor-banner-grab"
				type="checkbox"
				bind:checked={bannerGrab}
				class="sr-only peer"
			/>
			<div class="w-9 h-5 bg-muted rounded-full peer peer-checked:bg-accent transition-colors after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-4"></div>
		</label>
	</div>
</div>
