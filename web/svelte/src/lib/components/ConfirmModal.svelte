<script lang="ts">
	import { X, AlertTriangle } from 'lucide-svelte';

	interface Props {
		open: boolean;
		title: string;
		message: string;
		confirmLabel?: string;
		cancelLabel?: string;
		variant?: 'danger' | 'warning';
		loading?: boolean;
		onConfirm: () => void;
		onCancel: () => void;
	}

	let {
		open,
		title,
		message,
		confirmLabel = 'Confirm',
		cancelLabel = 'Cancel',
		variant = 'danger',
		loading = false,
		onConfirm,
		onCancel
	}: Props = $props();

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && !loading) onCancel();
	}

	function handleBackdrop() {
		if (!loading) onCancel();
	}

	const confirmBtnClass = $derived(
		variant === 'danger'
			? 'bg-red-500 text-white hover:bg-red-600'
			: 'bg-yellow-500 text-white hover:bg-yellow-600'
	);

	const iconBgClass = $derived(
		variant === 'danger'
			? 'bg-red-500/10'
			: 'bg-yellow-500/10'
	);

	const iconColorClass = $derived(
		variant === 'danger'
			? 'text-red-400'
			: 'text-yellow-400'
	);
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm animate-fade-in"
		onkeydown={handleKeydown}
		onclick={handleBackdrop}
	>
		<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
		<div
			class="bg-card border border-border rounded-lg shadow-lg w-full max-w-sm mx-4 animate-fade-in-up"
			onclick={(e) => e.stopPropagation()}
			role="dialog"
			aria-label={title}
		>
			<!-- Header -->
			<div class="px-5 py-3.5 border-b border-border flex items-center justify-between">
				<div class="flex items-center space-x-2">
					<div class="w-7 h-7 {iconBgClass} rounded-lg flex items-center justify-center">
						<AlertTriangle class="w-3.5 h-3.5 {iconColorClass}" />
					</div>
					<h3 class="text-sm font-medium text-foreground">{title}</h3>
				</div>
				<button
					onclick={onCancel}
					disabled={loading}
					class="text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
					aria-label="Close"
				>
					<X class="w-4 h-4" />
				</button>
			</div>

			<!-- Body -->
			<div class="px-5 py-4">
				<p class="text-sm text-muted-foreground leading-relaxed">{message}</p>
			</div>

			<!-- Footer -->
			<div class="px-5 py-3.5 border-t border-border flex justify-end space-x-2">
				<button
					onclick={onCancel}
					disabled={loading}
					class="px-4 py-2 bg-muted text-muted-foreground hover:bg-muted/80 text-xs font-medium rounded-md transition-colors disabled:opacity-50"
				>
					{cancelLabel}
				</button>
				<button
					onclick={onConfirm}
					disabled={loading}
					class="px-4 py-2 {confirmBtnClass} text-xs font-medium rounded-md transition-colors disabled:opacity-50"
				>
					{loading ? 'Processing...' : confirmLabel}
				</button>
			</div>
		</div>
	</div>
{/if}
