<script lang="ts">
	import { ShieldCheck, CheckCircle2 } from 'lucide-svelte';
</script>

<svelte:head>
	<title>Unsubscribed · WatchDog</title>
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
			<CheckCircle2 class="w-8 h-8 text-success mx-auto mb-3" />
			<h2 class="text-base font-semibold text-foreground mb-1">You've been unsubscribed</h2>
			<p class="text-xs text-muted-foreground">
				You won't receive further notifications. Re-subscribe any time from the status page.
			</p>
		</div>
	</div>
</div>
