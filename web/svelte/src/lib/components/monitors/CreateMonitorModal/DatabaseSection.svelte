<script lang="ts">
	import { FormField } from '@sylvester-francis/watchdog-ui';
	import { Input } from '@sylvester-francis/watchdog-ui';
	import { Select } from '@sylvester-francis/watchdog-ui';

	interface Props {
		dbType: string;
		dbConnectionString: string;
		dbPassword: string;
	}

	let {
		dbType = $bindable(),
		dbConnectionString = $bindable(),
		dbPassword = $bindable(),
	}: Props = $props();
</script>

<div class="space-y-3 pt-1">
	<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">Database Settings</div>
	<FormField label="DB Type" htmlFor="monitor-db-type">
		<Select id="monitor-db-type" bind:value={dbType}>
			<option value="postgres">PostgreSQL</option>
			<option value="mysql">MySQL</option>
			<option value="redis">Redis</option>
			<option value="mongodb">MongoDB</option>
		</Select>
	</FormField>
	<FormField label="Connection String" htmlFor="monitor-db-conn">
		<Input id="monitor-db-conn" type="text" bind:value={dbConnectionString} placeholder="host=localhost port=5432 dbname=mydb user=postgres" />
	</FormField>
	<FormField label="Password" htmlFor="monitor-db-pass">
		<Input id="monitor-db-pass" type="password" bind:value={dbPassword} placeholder="Database password" />
	</FormField>
</div>
