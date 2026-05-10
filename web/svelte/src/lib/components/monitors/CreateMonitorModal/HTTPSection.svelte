<script lang="ts">
	import { FormField } from '@sylvester-francis/watchdog-ui';
	import { Input } from '@sylvester-francis/watchdog-ui';

	interface Props {
		expectedStatus: number;
	}

	let { expectedStatus = $bindable() }: Props = $props();
</script>

<FormField label="Expected Status Code" htmlFor="monitor-expected-status">
	<Input id="monitor-expected-status" type="number" bind:value={expectedStatus} min={100} max={599} />
</FormField>
