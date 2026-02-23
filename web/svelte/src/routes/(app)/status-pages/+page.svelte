<script lang="ts">
	import { onMount } from 'svelte';
	import { Plus, Globe, ExternalLink, Pencil, Trash2 } from 'lucide-svelte';
	import { statusPages as statusPagesApi } from '$lib/api';
	import { getAuth } from '$lib/stores/auth.svelte';
	import { getToasts } from '$lib/stores/toast.svelte';
	import type { StatusPage } from '$lib/types';
	import CreateStatusPageModal from '$lib/components/status-pages/CreateStatusPageModal.svelte';
	import ConfirmModal from '$lib/components/ConfirmModal.svelte';

	const auth = getAuth();
	const toast = getToasts();

	let statusPages = $state<StatusPage[]>([]);
	let loading = $state(true);
	let showCreateModal = $state(false);

	// Delete confirmation modal
	let deleting = $state(false);
	let confirmModal = $state<{
		open: boolean;
		pageId: string;
		pageName: string;
	}>({ open: false, pageId: '', pageName: '' });

	let username = $derived(auth.user?.username ?? '');

	function publicUrl(sp: StatusPage): string {
		const origin = typeof window !== 'undefined' ? window.location.origin : '';
		return `${origin}/status/@${username}/${sp.slug}`;
	}

	function truncate(text: string, maxLen: number): string {
		if (!text || text.length <= maxLen) return text;
		return text.slice(0, maxLen) + '...';
	}

	function handleDelete(id: string) {
		const sp = statusPages.find(s => s.id === id);
		confirmModal = { open: true, pageId: id, pageName: sp?.name ?? 'this status page' };
	}

	async function executeDelete() {
		deleting = true;
		try {
			await statusPagesApi.deleteStatusPage(confirmModal.pageId);
			statusPages = statusPages.filter((sp) => sp.id !== confirmModal.pageId);
			confirmModal = { open: false, pageId: '', pageName: '' };
			toast.success('Status page deleted');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to delete status page');
		} finally {
			deleting = false;
		}
	}

	function closeConfirmModal() {
		if (!deleting) confirmModal = { open: false, pageId: '', pageName: '' };
	}

	async function loadData() {
		try {
			const res = await statusPagesApi.listStatusPages();
			statusPages = res.data ?? [];
		} catch {
			// Keep defaults on error
		} finally {
			loading = false;
		}
	}

	function handleCreated() {
		loadData();
		toast.success('Status page created');
	}

	onMount(() => {
		loadData();
	});
</script>

<svelte:head>
	<title>Status Pages - WatchDog</title>
</svelte:head>

{#if loading}
	<!-- Skeleton loading state -->
	<div class="animate-fade-in-up space-y-4">
		<!-- Header skeleton -->
		<div class="flex items-center justify-between">
			<div>
				<div class="h-7 w-36 bg-muted/50 rounded animate-pulse"></div>
				<div class="h-3 w-48 bg-muted/30 rounded animate-pulse mt-1.5"></div>
			</div>
			<div class="h-9 w-36 bg-muted/50 rounded-md animate-pulse"></div>
		</div>
		<!-- List skeleton -->
		<div class="bg-card border border-border rounded-lg">
			{#each Array(3) as _}
				<div class="flex items-center px-4 py-4 border-b border-border/20">
					<div class="flex-1 space-y-1.5">
						<div class="flex items-center space-x-2">
							<div class="h-4 w-40 bg-muted/50 rounded animate-pulse"></div>
							<div class="h-4 w-14 bg-muted/30 rounded animate-pulse"></div>
						</div>
						<div class="h-3 w-64 bg-muted/30 rounded animate-pulse"></div>
					</div>
					<div class="flex items-center space-x-2">
						<div class="h-7 w-7 bg-muted/50 rounded animate-pulse"></div>
						<div class="h-7 w-7 bg-muted/50 rounded animate-pulse"></div>
						<div class="h-7 w-7 bg-muted/50 rounded animate-pulse"></div>
					</div>
				</div>
			{/each}
		</div>
	</div>
{:else}
	<div class="animate-fade-in-up">
		<!-- Page header -->
		<div class="flex items-center justify-between mb-5">
			<div>
				<h1 class="text-lg font-semibold text-foreground">Status Pages</h1>
				<p class="text-xs text-muted-foreground mt-0.5">
					{statusPages.length} status page{statusPages.length !== 1 ? 's' : ''} configured
				</p>
			</div>
			<button
				onclick={() => { showCreateModal = true; }}
				class="flex items-center space-x-1.5 px-3 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors"
			>
				<Plus class="w-3.5 h-3.5" />
				<span>New Status Page</span>
			</button>
		</div>

		{#if statusPages.length === 0}
			<!-- Empty state -->
			<div class="bg-card border border-border rounded-lg">
				<div class="p-12 text-center">
					<div class="w-12 h-12 bg-muted/50 rounded-lg flex items-center justify-center mx-auto mb-4">
						<Globe class="w-6 h-6 text-muted-foreground/40" />
					</div>
					<p class="text-sm font-medium text-foreground mb-1">No status pages</p>
					<p class="text-xs text-muted-foreground mb-4">
						Create a public status page to share your service health with your users.
					</p>
					<button
						onclick={() => { showCreateModal = true; }}
						class="inline-flex items-center space-x-1.5 px-4 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors"
					>
						<Plus class="w-3.5 h-3.5" />
						<span>Create Status Page</span>
					</button>
				</div>
			</div>
		{:else}
			<!-- Status pages list -->
			<div class="bg-card border border-border rounded-lg">
				<div class="divide-y divide-border/20">
					{#each statusPages as sp (sp.id)}
						<div class="flex items-center px-4 py-3.5 hover:bg-card-elevated transition-colors group">
							<!-- Info -->
							<div class="flex-1 min-w-0">
								<div class="flex items-center space-x-2 mb-0.5">
									<span class="text-sm text-foreground font-medium truncate">{sp.name}</span>
									{#if sp.is_public}
										<span class="text-[9px] font-medium uppercase px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-400">
											Public
										</span>
									{:else}
										<span class="text-[9px] font-medium uppercase px-1.5 py-0.5 rounded bg-muted/50 text-muted-foreground">
											Private
										</span>
									{/if}
								</div>
								{#if sp.is_public && username}
									<a
										href={publicUrl(sp)}
										target="_blank"
										rel="noopener noreferrer"
										class="text-[10px] text-muted-foreground font-mono hover:text-accent transition-colors truncate block"
									>
										/status/@{username}/{sp.slug}
									</a>
								{:else}
									<p class="text-[10px] text-muted-foreground font-mono truncate">
										/{sp.slug}
									</p>
								{/if}
								{#if sp.description}
									<p class="text-[10px] text-muted-foreground mt-0.5 truncate hidden sm:block">
										{truncate(sp.description, 100)}
									</p>
								{/if}
							</div>

							<!-- Actions -->
							<div class="flex items-center space-x-1 shrink-0 ml-3">
								<!-- View (external link) -->
								{#if sp.is_public && username}
									<a
										href={publicUrl(sp)}
										target="_blank"
										rel="noopener noreferrer"
										class="p-1.5 rounded hover:bg-muted/50 text-muted-foreground/40 hover:text-muted-foreground transition-colors"
										title="View status page"
									>
										<ExternalLink class="w-3.5 h-3.5" />
									</a>
								{/if}

								<!-- Edit -->
								<a
									href="/status-pages/{sp.id}/edit"
									class="p-1.5 rounded hover:bg-muted/50 text-muted-foreground/40 hover:text-muted-foreground transition-colors"
									title="Edit status page"
								>
									<Pencil class="w-3.5 h-3.5" />
								</a>

								<!-- Delete -->
								<button
									onclick={() => handleDelete(sp.id)}
									class="p-1.5 rounded hover:bg-red-500/10 text-muted-foreground/40 hover:text-red-400 transition-colors"
									title="Delete status page"
								>
									<Trash2 class="w-3.5 h-3.5" />
								</button>
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	</div>

	<CreateStatusPageModal
		open={showCreateModal}
		onClose={() => { showCreateModal = false; }}
		onCreated={handleCreated}
	/>

	<ConfirmModal
		open={confirmModal.open}
		title="Delete Status Page"
		message="Are you sure you want to delete &quot;{confirmModal.pageName}&quot;? This cannot be undone."
		confirmLabel="Delete"
		variant="danger"
		loading={deleting}
		onConfirm={executeDelete}
		onCancel={closeConfirmModal}
	/>
{/if}
