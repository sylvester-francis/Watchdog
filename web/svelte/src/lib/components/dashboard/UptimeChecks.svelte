<script lang="ts">
	interface Props {
		checkResults: number[];
	}

	let { checkResults }: Props = $props();

	const maxSegments = 20;
	let segments = $derived(() => {
		const result: ('up' | 'down' | 'empty')[] = [];
		for (let i = 0; i < checkResults.length && i < maxSegments; i++) {
			result.push(checkResults[i] === 1 ? 'up' : 'down');
		}
		for (let i = result.length; i < maxSegments; i++) {
			result.push('empty');
		}
		return result;
	});
</script>

<div class="flex items-center space-x-px">
	{#each segments() as seg}
		<div
			class="w-[5px] h-4 rounded-sm {seg === 'up' ? 'bg-emerald-500/70' : seg === 'down' ? 'bg-red-500/70' : 'bg-muted/40'}"
		></div>
	{/each}
</div>
