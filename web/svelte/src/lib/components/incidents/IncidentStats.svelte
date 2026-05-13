<script lang="ts">
	import { StatCell, StatGrid } from '@sylvester-francis/watchdog-ui';

	interface Props {
		open: number;
		acknowledged: number;
		resolved: number;
	}

	let { open, acknowledged, resolved }: Props = $props();
</script>

<StatGrid columns={3}>
	<StatCell label="Open" value={open} valueTone={open > 0 ? 'destructive' : 'default'} />
	<StatCell label="Acknowledged" value={acknowledged} valueTone={acknowledged > 0 ? 'warning' : 'default'} />
	<StatCell label="Resolved" value={resolved} />
</StatGrid>
