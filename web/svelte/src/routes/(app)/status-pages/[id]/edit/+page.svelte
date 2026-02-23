<script lang="ts">
	import { onMount } from 'svelte';
	import { base } from '$app/paths';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { ChevronRight, AlertCircle } from 'lucide-svelte';
	import { statusPages as statusPagesApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast';
	import type { StatusPage } from '$lib/types';

	const toast = getToasts();

	interface AvailableMonitor {
		id: string;
		name: string;
		type: string;
		target: string;
		status: string;
	}

	let statusPage = $state<StatusPage | null>(null);
	let availableMonitors = $state<AvailableMonitor[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state('');
	let loadError = $state('');

	// Form state
	let name = $state('');
	let description = $state('');
	let isPublic = $state(false);
	let selectedMonitorIds = $state<Set<string>>(new Set());

	let pageId = $derived(page.params.id ?? '');

	const inputClass = 'w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background';
	const labelClass = 'block text-xs font-medium text-muted-foreground mb-1.5';

	function statusDotClass(status: string): string {
		if (status === 'up') return 'bg-emerald-400';
		if (status === 'down') return 'bg-red-400';
		if (status === 'degraded') return 'bg-amber-400';
		return 'bg-muted-foreground/50';
	}

	function toggleMonitor(id: string) {
		const next = new Set(selectedMonitorIds);
		if (next.has(id)) {
			next.delete(id);
		} else {
			next.add(id);
		}
		selectedMonitorIds = next;
	}

	async function loadData() {
		loading = true;
		loadError = '';

		try {
			const res = await statusPagesApi.getStatusPage(pageId);
			statusPage = res.data;
			availableMonitors = res.available_monitors ?? [];

			// Populate form from loaded data
			name = statusPage.name;
			description = statusPage.description ?? '';
			isPublic = statusPage.is_public;
			selectedMonitorIds = new Set(statusPage.monitor_ids ?? []);
		} catch (err) {
			const msg = err instanceof Error ? err.message : 'Failed to load status page';
			loadError = msg;
			toast.error(msg);
		} finally {
			loading = false;
		}
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();

		if (!name.trim()) {
			error = 'Name is required';
			return;
		}

		saving = true;
		error = '';

		try {
			await statusPagesApi.updateStatusPage(pageId, {
				name: name.trim(),
				description: description.trim(),
				is_public: isPublic,
				monitor_ids: Array.from(selectedMonitorIds)
			});
			toast.success('Status page updated');
			goto(`${base}/status-pages`);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update status page';
		} finally {
			saving = false;
		}
	}

	onMount(() => {
		loadData();
	});
</script>

<svelte:head>
	<title>{statusPage?.name ?? 'Edit Status Page'} - WatchDog</title>
</svelte:head>

{#if loading}
	<!-- Skeleton loading state -->
	<div class="animate-fade-in-up space-y-4">
		<!-- Breadcrumb skeleton -->
		<div class="h-4 w-56 bg-muted/50 rounded animate-pulse"></div>

		<!-- Form card skeleton -->
		<div class="bg-card border border-border rounded-lg">
			<div class="p-5 space-y-4">
				<div class="h-4 w-20 bg-muted/50 rounded animate-pulse"></div>
				<div class="h-10 w-full bg-muted/30 rounded-md animate-pulse"></div>
				<div class="h-4 w-24 bg-muted/50 rounded animate-pulse"></div>
				<div class="h-20 w-full bg-muted/30 rounded-md animate-pulse"></div>
				<div class="h-4 w-16 bg-muted/50 rounded animate-pulse"></div>
				<div class="h-6 w-48 bg-muted/30 rounded animate-pulse"></div>
				<div class="h-4 w-24 bg-muted/50 rounded animate-pulse mt-4"></div>
				{#each Array(3) as _}
					<div class="flex items-center px-3 py-2.5 border border-border/20 rounded-md">
						<div class="w-4 h-4 bg-muted/50 rounded animate-pulse mr-3"></div>
						<div class="flex-1 space-y-1">
							<div class="h-3.5 w-32 bg-muted/50 rounded animate-pulse"></div>
							<div class="h-3 w-48 bg-muted/30 rounded animate-pulse"></div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</div>
{:else if loadError && !statusPage}
	<!-- Error state -->
	<div class="animate-fade-in-up">
		<div class="bg-card border border-border rounded-lg p-8 text-center">
			<p class="text-sm text-foreground font-medium mb-1">Failed to load status page</p>
			<p class="text-xs text-muted-foreground mb-4">{loadError}</p>
			<a
				href="{base}/status-pages"
				class="inline-flex items-center px-4 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors"
			>
				Back to Status Pages
			</a>
		</div>
	</div>
{:else if statusPage}
	<div class="animate-fade-in-up space-y-5">
		<!-- Breadcrumb -->
		<nav class="flex items-center space-x-1.5 text-xs">
			<a href="{base}/status-pages" class="text-muted-foreground hover:text-foreground transition-colors">
				Status Pages
			</a>
			<ChevronRight class="w-3 h-3 text-muted-foreground/50" />
			<span class="text-foreground font-medium truncate max-w-[200px]">{statusPage.name}</span>
		</nav>

		<!-- Form card -->
		<div class="bg-card border border-border rounded-lg">
			<form onsubmit={handleSubmit}>
				<div class="p-5 space-y-4">
					{#if error}
						<div class="bg-destructive/10 border border-destructive/20 rounded-md px-3 py-2 flex items-center space-x-2" role="alert">
							<AlertCircle class="w-3.5 h-3.5 text-destructive flex-shrink-0" />
							<span class="text-xs text-destructive">{error}</span>
						</div>
					{/if}

					<!-- Name -->
					<div>
						<label for="sp-name" class={labelClass}>Name</label>
						<input
							id="sp-name"
							type="text"
							bind:value={name}
							required
							placeholder="My Status Page"
							class={inputClass}
						/>
					</div>

					<!-- Description -->
					<div>
						<label for="sp-description" class={labelClass}>Description</label>
						<textarea
							id="sp-description"
							bind:value={description}
							placeholder="A brief description of what this status page covers..."
							rows="3"
							class={inputClass}
						></textarea>
					</div>

					<!-- Public toggle -->
					<div class="flex items-center space-x-3">
						<input
							id="sp-public"
							type="checkbox"
							bind:checked={isPublic}
							class="w-4 h-4 rounded border-border bg-card-elevated text-accent focus:ring-ring focus:ring-offset-background"
						/>
						<label for="sp-public" class="text-xs text-foreground">
							Make this page publicly accessible
						</label>
					</div>

					<!-- Monitor selection -->
					<div>
						<div class="text-xs font-medium text-muted-foreground mb-2">
							Monitors
							{#if selectedMonitorIds.size > 0}
								<span class="text-muted-foreground/60 font-normal ml-1">
									({selectedMonitorIds.size} selected)
								</span>
							{/if}
						</div>

						{#if availableMonitors.length === 0}
							<div class="border border-border/30 rounded-md p-4 text-center">
								<p class="text-xs text-muted-foreground">
									No monitors available. Create monitors first, then assign them here.
								</p>
							</div>
						{:else}
							<div class="border border-border/30 rounded-md divide-y divide-border/20 max-h-64 overflow-y-auto">
								{#each availableMonitors as monitor (monitor.id)}
									<button
										type="button"
										onclick={() => toggleMonitor(monitor.id)}
										class="flex items-center w-full px-3 py-2.5 hover:bg-card-elevated transition-colors text-left"
									>
										<input
											type="checkbox"
											checked={selectedMonitorIds.has(monitor.id)}
											tabindex={-1}
											class="w-3.5 h-3.5 rounded border-border bg-card-elevated text-accent focus:ring-ring focus:ring-offset-background mr-3 pointer-events-none"
										/>
										<div class="w-2 h-2 rounded-full {statusDotClass(monitor.status)} mr-2.5 shrink-0"></div>
										<div class="flex-1 min-w-0">
											<div class="flex items-center space-x-2">
												<span class="text-xs text-foreground truncate">{monitor.name}</span>
												<span class="text-[9px] text-muted-foreground font-mono uppercase px-1.5 py-0.5 rounded bg-muted/50 shrink-0">
													{monitor.type}
												</span>
											</div>
											<p class="text-[10px] text-muted-foreground font-mono truncate mt-0.5">
												{monitor.target}
											</p>
										</div>
									</button>
								{/each}
							</div>
						{/if}
					</div>
				</div>

				<!-- Footer -->
				<div class="px-5 py-3.5 border-t border-border flex justify-end space-x-2">
					<a
						href="{base}/status-pages"
						class="px-4 py-2 bg-muted text-muted-foreground hover:bg-muted/80 text-xs font-medium rounded-md transition-colors"
					>
						Cancel
					</a>
					<button
						type="submit"
						disabled={saving}
						class="px-4 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors disabled:opacity-50"
					>
						{saving ? 'Saving...' : 'Save Changes'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
