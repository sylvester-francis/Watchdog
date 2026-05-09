<script lang="ts">
	import FormField from '$lib/ui/FormField.svelte';
	import Input from '$lib/ui/Input.svelte';

	interface Props {
		expectedStatus: number;
	}

	let { expectedStatus = $bindable() }: Props = $props();
</script>

<FormField label="Expected Status Code" htmlFor="monitor-expected-status">
	<Input id="monitor-expected-status" type="number" bind:value={expectedStatus} min={100} max={599} />
</FormField>
