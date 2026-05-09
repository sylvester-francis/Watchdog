<script lang="ts">
	import type { Agent, MonitorType } from '$lib/types';
	import FormField from '$lib/ui/FormField.svelte';
	import Input from '$lib/ui/Input.svelte';
	import Select from '$lib/ui/Select.svelte';

	interface Props {
		name: string;
		type: MonitorType;
		agentId: string;
		target: string;
		intervalSeconds: number;
		timeoutSeconds: number;
		failureThreshold: number;
		agents: Agent[];
		monitorTypes: { value: MonitorType; label: string }[];
		targetPlaceholder: string;
	}

	let {
		name = $bindable(),
		type = $bindable(),
		agentId = $bindable(),
		target = $bindable(),
		intervalSeconds = $bindable(),
		timeoutSeconds = $bindable(),
		failureThreshold = $bindable(),
		agents,
		monitorTypes,
		targetPlaceholder,
	}: Props = $props();
</script>

<FormField label="Name" htmlFor="monitor-name" required>
	<Input id="monitor-name" type="text" bind:value={name} placeholder="My API Server" />
</FormField>

<div class="grid grid-cols-2 gap-3">
	<FormField label="Type" htmlFor="monitor-type">
		<Select id="monitor-type" bind:value={type}>
			{#each monitorTypes as mt}
				<option value={mt.value}>{mt.label}</option>
			{/each}
		</Select>
	</FormField>
	<FormField label="Agent" htmlFor="monitor-agent" required>
		<Select id="monitor-agent" bind:value={agentId}>
			<option value="" disabled>Select agent</option>
			{#each agents as agent}
				<option value={agent.id}>{agent.name}</option>
			{/each}
		</Select>
	</FormField>
</div>

<FormField label="Target" htmlFor="monitor-target" required>
	<Input id="monitor-target" type="text" bind:value={target} placeholder={targetPlaceholder} />
</FormField>

<div class="grid grid-cols-3 gap-3">
	<FormField label="Interval (s)" htmlFor="monitor-interval">
		<Input id="monitor-interval" type="number" bind:value={intervalSeconds} min={5} max={3600} />
	</FormField>
	<FormField label="Timeout (s)" htmlFor="monitor-timeout">
		<Input id="monitor-timeout" type="number" bind:value={timeoutSeconds} min={1} max={300} />
	</FormField>
	<FormField label="Fail Threshold" htmlFor="monitor-threshold">
		<Input id="monitor-threshold" type="number" bind:value={failureThreshold} min={1} max={10} />
	</FormField>
</div>
