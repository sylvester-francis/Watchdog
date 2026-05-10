<script lang="ts">
	import { X, Loader2 } from 'lucide-svelte';
	import type { IncidentInvestigation } from '$lib/types';
	import InvestigationPanel from './InvestigationPanel.svelte';
	import { Sheet, BottomSheet } from '@sylvester-francis/watchdog-ui';

	interface Props {
		open: boolean;
		loading: boolean;
		investigation: IncidentInvestigation | null;
		onClose: () => void;
	}

	let { open, loading, investigation, onClose }: Props = $props();

	const TOUCH_QUERY = '(hover: none) and (pointer: coarse)';

	let isMobile = $state(
		typeof window !== 'undefined' && window.matchMedia(TOUCH_QUERY).matches
	);

	$effect(() => {
		if (typeof window === 'undefined') return;

		const mql = window.matchMedia(TOUCH_QUERY);
		isMobile = mql.matches;

		function handleChange(e: MediaQueryListEvent) {
			isMobile = e.matches;
		}
		mql.addEventListener('change', handleChange);

		return () => mql.removeEventListener('change', handleChange);
	});
</script>

{#if isMobile}
	<BottomSheet {open} height="full" onclose={onClose}>
		<div class="shrink-0 border-b border-border px-5 pb-3 flex items-center justify-between">
			<h2 class="text-sm font-semibold text-foreground">Incident Investigation</h2>
			<button
				type="button"
				onclick={onClose}
				class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors"
				aria-label="Close"
			>
				<X class="w-4 h-4" />
			</button>
		</div>

		<div class="flex-1 overflow-y-auto p-5">
			{#if loading}
				<div class="flex items-center justify-center py-16">
					<Loader2 class="w-6 h-6 text-muted-foreground animate-spin" />
				</div>
			{:else if investigation}
				<InvestigationPanel {investigation} />
			{/if}
		</div>
	</BottomSheet>
{:else}
	<Sheet {open} side="right" size="2xl" onclose={onClose}>
		<div class="sticky top-0 -mx-4 -mt-4 mb-4 bg-background/95 backdrop-blur-sm border-b border-border px-6 py-4 flex items-center justify-between z-10">
			<h2 class="text-sm font-semibold text-foreground">Incident Investigation</h2>
			<button
				type="button"
				onclick={onClose}
				class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors"
				aria-label="Close"
			>
				<X class="w-4 h-4" />
			</button>
		</div>

		<div class="px-2 pb-2">
			{#if loading}
				<div class="flex items-center justify-center py-16">
					<Loader2 class="w-6 h-6 text-muted-foreground animate-spin" />
				</div>
			{:else if investigation}
				<InvestigationPanel {investigation} />
			{/if}
		</div>
	</Sheet>
{/if}
