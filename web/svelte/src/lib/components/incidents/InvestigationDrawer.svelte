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

	// Mobile detection via matchMedia
	let isMobile = $state(false);

	// Drag state
	let dragging = $state(false);
	let dragOffset = $state(0);
	let sheetHeight = $state<'half' | 'full'>('half');
	let closing = $state(false);

	// Velocity tracking (rolling 3-sample window)
	let velocitySamples: { y: number; t: number }[] = [];
	let dragStartY = 0;

	$effect(() => {
		if (typeof window === 'undefined') return;

		const mql = window.matchMedia('(min-width: 1024px)');
		isMobile = !mql.matches;

		function handleChange(e: MediaQueryListEvent) {
			isMobile = !e.matches;
		}
		mql.addEventListener('change', handleChange);

		return () => mql.removeEventListener('change', handleChange);
	});

	// Body scroll lock when open
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

	// Reset state when opening
	$effect(() => {
		if (open) {
			sheetHeight = 'half';
			dragOffset = 0;
			closing = false;
		}
	});

	// Escape key handler
	$effect(() => {
		if (!open || typeof window === 'undefined') return;

		function handleKeydown(e: KeyboardEvent) {
			if (e.key === 'Escape') onClose();
		}
		window.addEventListener('keydown', handleKeydown);
		return () => window.removeEventListener('keydown', handleKeydown);
	});

	function onPointerDown(e: PointerEvent) {
		if (!isMobile) return;
		dragging = true;
		dragStartY = e.clientY;
		dragOffset = 0;
		velocitySamples = [{ y: e.clientY, t: e.timeStamp }];
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
	}

	function onPointerMove(e: PointerEvent) {
		if (!dragging) return;
		const dy = e.clientY - dragStartY;
		// Only allow downward drag when at half height, both directions otherwise
		if (sheetHeight === 'half') {
			dragOffset = Math.max(-60, dy); // allow small upward drag to expand
		} else {
			dragOffset = dy;
		}

		// Rolling velocity window (keep last 3 samples)
		velocitySamples.push({ y: e.clientY, t: e.timeStamp });
		if (velocitySamples.length > 3) velocitySamples.shift();
	}

	function onPointerUp(_e: PointerEvent) {
		if (!dragging) return;
		dragging = false;

		// Calculate velocity from rolling window
		let velocity = 0;
		if (velocitySamples.length >= 2) {
			const first = velocitySamples[0];
			const last = velocitySamples[velocitySamples.length - 1];
			const dt = last.t - first.t;
			if (dt > 0) {
				velocity = (last.y - first.y) / dt; // px/ms, positive = downward
			}
		}

		if (sheetHeight === 'half') {
			if (dragOffset > 120 || velocity > 0.5) {
				// Dismiss
				dismiss();
			} else if (dragOffset < -60 || velocity < -0.5) {
				// Expand to full
				sheetHeight = 'full';
				dragOffset = 0;
			} else {
				// Snap back
				dragOffset = 0;
			}
		} else {
			// Full height
			if (dragOffset > 120 || velocity > 0.5) {
				// Dismiss
				dismiss();
			} else if (dragOffset > 60) {
				// Collapse to half
				sheetHeight = 'half';
				dragOffset = 0;
			} else {
				// Snap back
				dragOffset = 0;
			}
		}

		velocitySamples = [];
	}

	function onPointerCancel(_e: PointerEvent) {
		dragging = false;
		dragOffset = 0;
		velocitySamples = [];
	}

	function dismiss() {
		closing = true;
		dragOffset = 0;
	}

	function onTransitionEnd(e: TransitionEvent) {
		if (closing && e.propertyName === 'transform') {
			closing = false;
			onClose();
		}
	}

	function getSheetStyle(): string {
		if (!isMobile) return '';

		if (closing) {
			return 'transform: translateY(100%); transition: transform 0.3s cubic-bezier(0.32, 0.72, 0, 1);';
		}

		if (dragging) {
			return `transform: translateY(${Math.max(0, dragOffset)}px); transition: none;`;
		}

		return 'transition: transform 0.3s cubic-bezier(0.32, 0.72, 0, 1);';
	}
</script>

{#if open}
	<!-- Backdrop -->
	<button
		class="fixed inset-0 bg-black/50 z-40"
		onclick={() => { if (isMobile) dismiss(); else onClose(); }}
		aria-label="Close investigation panel"
	></button>

	<!-- Panel -->
	{#if isMobile}
		<!-- Mobile bottom sheet -->
		<div
			class="fixed bottom-0 left-0 right-0 z-50 bg-background border-t border-border rounded-t-2xl shadow-2xl flex flex-col animate-slide-in-up {sheetHeight === 'full' ? 'h-92dvh' : 'h-55dvh'}"
			style={getSheetStyle()}
			ontransitionend={onTransitionEnd}
			role="dialog"
			aria-label="Incident Investigation"
		>
			<!-- Drag handle -->
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div
				class="shrink-0 flex justify-center py-3 cursor-grab touch-none"
				onpointerdown={onPointerDown}
				onpointermove={onPointerMove}
				onpointerup={onPointerUp}
				onpointercancel={onPointerCancel}
			>
				<div class="w-10 h-1 rounded-full bg-muted-foreground/30"></div>
			</div>

			<!-- Header -->
			<div class="shrink-0 border-b border-border px-5 pb-3 flex items-center justify-between">
				<h2 class="text-sm font-semibold text-foreground">Incident Investigation</h2>
				<button
					onclick={() => dismiss()}
					class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors"
					aria-label="Close"
				>
					<X class="w-4 h-4" />
				</button>
			</div>

			<!-- Scrollable body -->
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
	{:else}
		<!-- Desktop right slide-over -->
		<div
			class="fixed inset-y-0 right-0 z-50 w-full max-w-2xl bg-background border-l border-border shadow-2xl overflow-y-auto animate-slide-in-right"
			role="dialog"
			aria-label="Incident Investigation"
		>
			<div class="sticky top-0 bg-background/95 backdrop-blur-sm border-b border-border px-6 py-4 flex items-center justify-between z-10">
				<h2 class="text-sm font-semibold text-foreground">Incident Investigation</h2>
				<button
					onclick={onClose}
					class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors"
					aria-label="Close"
				>
					<X class="w-4 h-4" />
				</button>
			</div>

			<div class="p-6">
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
{/if}
