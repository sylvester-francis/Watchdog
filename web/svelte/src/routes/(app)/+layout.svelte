<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount, onDestroy } from 'svelte';
	import { getAuth } from '$lib/stores/auth.svelte';
	import { setOnUnauthorized } from '$lib/api/client';
	import { createSSE } from '$lib/stores/sse';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import Header from '$lib/components/Header.svelte';
	import ToastContainer from '$lib/components/ToastContainer.svelte';

	const auth = getAuth();
	let ready = $state(false);
	let { children } = $props();

	// SSE connection at layout level — shared across all (app) pages
	const sse = createSSE(() => {});
	export { sse };

	// Register 401 redirect callback (safe outside component init via module)
	setOnUnauthorized(() => {
		sse.disconnect();
		goto(`/login`);
	});

	onMount(async () => {
		const user = await auth.check();
		if (!user) {
			goto(`/login`);
			return;
		}
		ready = true;
		sse.connect();
	});

	onDestroy(() => {
		sse.disconnect();
	});
</script>

{#if !ready}
	<div class="flex min-h-screen items-center justify-center">
		<div class="flex flex-col items-center space-y-3">
			<div class="skeleton w-8 h-8 rounded-full"></div>
			<p class="text-xs text-muted-foreground">Loading...</p>
		</div>
	</div>
{:else}
	<div class="flex min-h-screen">
		<Sidebar />
		<main class="flex-1 lg:ml-64 flex flex-col min-h-screen">
			<Header />
			<div class="p-6 flex-1">
				{@render children()}
			</div>
			<footer class="mt-auto px-6 h-12 flex items-center border-t border-border/50">
				<div class="flex flex-col sm:flex-row items-center justify-between gap-1 w-full">
					<p class="text-[11px] text-muted-foreground/50">Open source &middot; Self-hosted &middot; Zero-trust</p>
					<div class="flex items-center space-x-3">
						<a href="https://github.com/sylvester-francis/Watchdog" target="_blank" rel="noopener noreferrer" class="text-[11px] text-muted-foreground/50 hover:text-muted-foreground transition-colors">GitHub</a>
						<a href="https://discord.gg/PPPjZDVS" target="_blank" rel="noopener noreferrer" class="text-[11px] text-muted-foreground/50 hover:text-muted-foreground transition-colors">Discord</a>
					</div>
				</div>
			</footer>
		</main>
	</div>
	<ToastContainer />
{/if}
