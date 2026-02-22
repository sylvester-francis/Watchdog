<script lang="ts">
	import { X, CheckCircle2, AlertCircle, AlertTriangle } from 'lucide-svelte';
	import { getToasts } from '$lib/stores/toast';

	const toasts = getToasts();

	const icons = {
		success: CheckCircle2,
		error: AlertCircle,
		warning: AlertTriangle
	};

	const colors = {
		success: 'border-emerald-500/20 bg-emerald-500/10',
		error: 'border-destructive/20 bg-destructive/10',
		warning: 'border-yellow-500/20 bg-yellow-500/10'
	};

	const textColors = {
		success: 'text-emerald-400',
		error: 'text-destructive',
		warning: 'text-yellow-400'
	};
</script>

<div class="fixed bottom-4 right-4 z-50 flex flex-col space-y-2">
	{#each toasts.items as toast (toast.id)}
		{@const Icon = icons[toast.type]}
		<div class="toast-enter flex items-center space-x-2 px-4 py-3 rounded-lg border {colors[toast.type]} max-w-sm shadow-lg">
			<Icon class="w-4 h-4 {textColors[toast.type]} flex-shrink-0" />
			<span class="text-sm {textColors[toast.type]} flex-1">{toast.message}</span>
			<button onclick={() => toasts.remove(toast.id)} class="p-0.5 rounded hover:bg-muted/50">
				<X class="w-3.5 h-3.5 text-muted-foreground" />
			</button>
		</div>
	{/each}
</div>
