<script lang="ts">
	import { X, Loader2 } from 'lucide-svelte';
	import type { IncidentInvestigation } from '$lib/types';
	import InvestigationPanel from './InvestigationPanel.svelte';

	interface Props {
		open: boolean;
		loading: boolean;
		investigation: IncidentInvestigation | null;
		onClose: () => void;
	}

	let { open, loading, investigation, onClose }: Props = $props();

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && open) onClose();
	}

	$effect(() => {
		if (typeof document === 'undefined') return;
		if (open) {
			document.body.style.overflow = 'hidden';
		} else {
			document.body.style.overflow = '';
		}
		return () => {
			document.body.style.overflow = '';
		};
	});
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<button
		type="button"
		onclick={onClose}
		aria-label="Close panel"
		class="fixed inset-0 z-40 bg-black/50"
	></button>

	<div
		role="dialog"
		aria-modal="true"
		class="fixed z-50 bg-card border-border shadow-2xl overflow-y-auto flex flex-col
		       inset-x-0 bottom-0 h-[92dvh] border-t rounded-t-2xl
		       lg:inset-y-0 lg:right-0 lg:bottom-auto lg:left-auto lg:h-full lg:w-full lg:max-w-2xl lg:border-l lg:border-t-0 lg:rounded-none"
	>
		<div class="sticky top-0 z-10 shrink-0 bg-background/95 backdrop-blur-sm border-b border-border px-5 py-3 flex items-center justify-between">
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
	</div>
{/if}
