<script lang="ts">
	import { onMount } from 'svelte';
	import { Plus, ExternalLink, Pencil, Trash2 } from 'lucide-svelte';
	import { Button, Skeleton } from '@sylvester-francis/watchdog-ui';
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
		const sp = statusPages.find((s) => s.id === id);
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

	async function handleCreated() {
		await loadData();
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
	<div class="animate-fade-in-up mx-auto max-w-[1080px] space-y-8 px-4 py-8 sm:px-6 sm:py-10">
		<div class="space-y-2">
			<Skeleton emphasis="tertiary" width="9rem" height="0.75rem" />
			<Skeleton emphasis="secondary" width="16rem" height="2rem" />
			<Skeleton emphasis="tertiary" width="12rem" height="0.875rem" />
		</div>
		<div class="space-y-2">
			{#each Array(3) as _}
				<Skeleton emphasis="tertiary" width="100%" height="3rem" />
			{/each}
		</div>
	</div>
{:else}
	<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-6 sm:px-6 sm:py-10">
		<!-- Page header -->
		<header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between sm:gap-4">
			<div class="min-w-0">
				<div class="flex items-center gap-2 font-mono tabular-nums text-xs text-muted-foreground">
					<span class="uppercase tracking-wider">Status Pages</span>
				</div>
				<h1 class="mt-1.5 text-xl font-medium text-foreground sm:text-2xl md:text-3xl">
					{statusPages.length} status page{statusPages.length !== 1 ? 's' : ''} configured
				</h1>
				<p class="mt-1 text-sm text-muted-foreground">Public-facing service health pages for your customers.</p>
			</div>
			<Button variant="primary" size="sm" onclick={() => { showCreateModal = true; }}>
				<span class="flex items-center gap-1.5">
					<Plus class="h-3.5 w-3.5" />
					<span>New Status Page</span>
				</span>
			</Button>
		</header>

		<div class="mt-8">
			{#if statusPages.length === 0}
				<section>
					<div class="border-b border-border pb-3">
						<h3 class="text-sm font-medium text-foreground">No status pages</h3>
					</div>
					<div class="flex flex-col items-start gap-3 pt-6 sm:flex-row sm:items-center sm:justify-between sm:gap-6">
						<p class="text-xs text-muted-foreground">
							Create a public status page to share your service health with your users.
						</p>
						<Button variant="primary" size="sm" onclick={() => { showCreateModal = true; }}>
							<span class="flex items-center gap-1.5">
								<Plus class="h-3.5 w-3.5" />
								<span>Create Status Page</span>
							</span>
						</Button>
					</div>
				</section>
			{:else}
				<section>
					<div class="flex items-baseline gap-2 border-b border-border pb-3">
						<h3 class="text-sm font-medium text-foreground">All Pages</h3>
						<span class="font-mono tabular-nums text-[11px] text-muted-foreground">{statusPages.length}</span>
					</div>
					<div class="divide-y divide-border/40">
						{#each statusPages as sp (sp.id)}
							<div class="group flex items-start gap-3 py-3 transition-colors hover:bg-muted/30">
								<!-- Info -->
								<div class="min-w-0 flex-1">
									<div class="flex items-baseline gap-2">
										<span class="truncate text-sm font-medium text-foreground">{sp.name}</span>
										<span class="font-mono tabular-nums text-[11px] uppercase tracking-wider {sp.is_public ? 'text-success' : 'text-muted-foreground'}">
											{sp.is_public ? 'Public' : 'Private'}
										</span>
									</div>
									{#if sp.is_public && username}
										<a
											href={publicUrl(sp)}
											target="_blank"
											rel="noopener noreferrer"
											class="mt-0.5 block truncate font-mono tabular-nums text-[11px] text-muted-foreground transition-colors hover:text-accent"
										>
											/status/@{username}/{sp.slug}
										</a>
									{:else}
										<p class="mt-0.5 truncate font-mono tabular-nums text-[11px] text-muted-foreground">
											/{sp.slug}
										</p>
									{/if}
									{#if sp.description}
										<p class="mt-1 hidden truncate text-xs text-muted-foreground sm:block">
											{truncate(sp.description, 100)}
										</p>
									{/if}
								</div>

								<!-- Actions -->
								<div class="flex shrink-0 items-center gap-2 text-xs sm:gap-3">
									{#if sp.is_public && username}
										<a
											href={publicUrl(sp)}
											target="_blank"
											rel="noopener noreferrer"
											class="inline-flex min-h-[36px] items-center gap-1 -my-1.5 px-1 text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline"
											title="View status page"
										>
											<ExternalLink class="h-3 w-3" />
											<span>View</span>
										</a>
									{/if}

									<a
										href="/status-pages/{sp.id}/edit"
										class="inline-flex min-h-[36px] items-center gap-1 -my-1.5 px-1 text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline"
										title="Edit status page"
									>
										<Pencil class="h-3 w-3" />
										<span>Edit</span>
									</a>

									<button
										onclick={() => handleDelete(sp.id)}
										class="inline-flex min-h-[36px] items-center gap-1 -my-1.5 px-1 text-destructive underline-offset-4 transition-colors hover:underline"
										title="Delete status page"
									>
										<Trash2 class="h-3 w-3" />
										<span>Delete</span>
									</button>
								</div>
							</div>
						{/each}
					</div>
				</section>
			{/if}
		</div>
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
