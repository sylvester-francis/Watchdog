<script lang="ts">
	import { goto } from '$app/navigation';
	import { base } from '$app/paths';
	import { Menu } from 'lucide-svelte';
	import { getAuth } from '$lib/stores/auth';

	const auth = getAuth();

	async function handleLogout() {
		await auth.logout();
		goto(`${base}/login`);
	}

	// We dispatch a custom event for the sidebar toggle
	function toggleSidebar() {
		document.querySelector('aside')?.classList.toggle('-translate-x-full');
	}
</script>

<header class="sticky top-0 z-40 bg-background/80 backdrop-blur-sm border-b border-border px-6 h-14 flex items-center">
	<div class="flex items-center justify-between w-full">
		<div class="flex items-center space-x-3">
			<button
				onclick={toggleSidebar}
				aria-label="Toggle navigation menu"
				class="lg:hidden p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-smooth"
			>
				<Menu class="w-5 h-5" />
			</button>
		</div>
		<div class="flex items-center space-x-4">
			{#if auth.user}
				<span class="text-xs text-muted-foreground">{auth.user.email}</span>
			{/if}
			<button
				onclick={handleLogout}
				class="text-xs text-muted-foreground hover:text-foreground transition-smooth"
			>
				Logout
			</button>
		</div>
	</div>
</header>
