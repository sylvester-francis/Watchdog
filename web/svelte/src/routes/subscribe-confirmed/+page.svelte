<script lang="ts">
	import { page } from '$app/state';
	import { ShieldCheck, CheckCircle2, AlertCircle } from 'lucide-svelte';

	const error = $derived(page.url.searchParams.get('error') === '1');
</script>

<svelte:head>
	<title>Subscription · WatchDog</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center p-6">
	<div class="w-full max-w-sm">
		<div class="text-center mb-8">
			<div class="inline-flex items-center justify-center w-10 h-10 bg-accent rounded-lg mb-3">
				<ShieldCheck class="w-5 h-5 text-white" />
			</div>
			<h1 class="text-lg font-semibold text-foreground">WatchDog</h1>
		</div>

		<div class="bg-card border border-border/50 rounded-lg p-6 text-center">
			{#if error}
				<AlertCircle class="w-8 h-8 text-destructive mx-auto mb-3" />
				<h2 class="text-base font-semibold text-foreground mb-1">Confirmation link expired</h2>
				<p class="text-xs text-muted-foreground">
					The link is invalid or already used. Subscribe again from the status page.
				</p>
			{:else}
				<CheckCircle2 class="w-8 h-8 text-success mx-auto mb-3" />
				<h2 class="text-base font-semibold text-foreground mb-1">You're subscribed</h2>
				<p class="text-xs text-muted-foreground">
					You'll receive an email when a monitor on this status page goes down.
				</p>
			{/if}
		</div>
	</div>
</div>
