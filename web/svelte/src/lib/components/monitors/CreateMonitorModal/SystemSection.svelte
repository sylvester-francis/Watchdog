<script lang="ts">
	import FormField from '$lib/ui/FormField.svelte';
	import Input from '$lib/ui/Input.svelte';
	import Select from '$lib/ui/Select.svelte';

	interface Props {
		systemMetric: string;
		systemThreshold: number;
	}

	let {
		systemMetric = $bindable(),
		systemThreshold = $bindable(),
	}: Props = $props();
</script>

<div class="space-y-3 pt-1">
	<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">System Metric</div>
	<div class="grid grid-cols-2 gap-3">
		<FormField label="Metric" htmlFor="monitor-sys-metric">
			<Select id="monitor-sys-metric" bind:value={systemMetric}>
				<option value="cpu">CPU</option>
				<option value="memory">Memory</option>
				<option value="disk">Disk</option>
			</Select>
		</FormField>
		<FormField label="Threshold %" htmlFor="monitor-sys-threshold">
			<Input id="monitor-sys-threshold" type="number" bind:value={systemThreshold} min={1} max={100} />
		</FormField>
	</div>
</div>
