<script lang="ts">
	import { LayoutDashboard } from 'lucide-svelte';
</script>

<svelte:head>
	<title>Dashboard - WatchDog</title>
</svelte:head>

<div class="animate-fade-in-up">
	<div class="flex items-center space-x-3 mb-6">
		<LayoutDashboard class="w-5 h-5 text-muted-foreground" />
		<h1 class="text-base font-semibold text-foreground">Dashboard</h1>
	</div>

	<div class="bg-card border border-border/50 rounded-lg p-8 text-center">
		<p class="text-sm text-muted-foreground">Dashboard coming in Phase B.</p>
		<p class="text-xs text-muted-foreground/50 mt-2">The SvelteKit migration is in progress. This page will be built next.</p>
	</div>
</div>
