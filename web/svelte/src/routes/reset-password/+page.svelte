<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { ShieldCheck, ArrowLeft, AlertCircle, CheckCircle2, KeyRound } from 'lucide-svelte';
	import { completePasswordReset } from '$lib/api/auth';

	let token = $derived($page.url.searchParams.get('token') ?? '');
	let password = $state('');
	let confirm = $state('');
	let submitting = $state(false);
	let message = $state('');
	let error = $state('');

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		message = '';
		if (!token) {
			error = 'Reset link is invalid — please request a new one from /forgot-password.';
			return;
		}
		if (password.length < 8) {
			error = 'Password must be at least 8 characters.';
			return;
		}
		if (password !== confirm) {
			error = 'Passwords do not match.';
			return;
		}
		submitting = true;
		try {
			const res = await completePasswordReset(token, password);
			message = res.message;
			// Brief success delay then redirect — gives user time to read confirmation.
			setTimeout(() => goto('/login?reset=1'), 1200);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Could not reset password — please request a new link.';
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>Reset Password · WatchDog</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center p-6">
	<div class="w-full max-w-sm">
		<div class="text-center mb-8">
			<div class="inline-flex items-center justify-center w-10 h-10 bg-accent rounded-lg mb-3">
				<ShieldCheck class="w-5 h-5 text-white" />
			</div>
			<h1 class="text-lg font-semibold text-foreground">WatchDog</h1>
		</div>

		<div class="bg-card border border-border/50 rounded-lg p-6">
			<div class="text-center mb-5">
				<div class="inline-flex items-center justify-center w-10 h-10 bg-muted/50 rounded-full mb-4">
					<KeyRound class="w-5 h-5 text-muted-foreground" />
				</div>
				<h2 class="text-base font-semibold text-foreground mb-1">Set a new password</h2>
				<p class="text-xs text-muted-foreground">Choose a strong password. Existing sessions on other devices will be signed out.</p>
			</div>

			{#if message}
				<div class="bg-emerald-500/10 border border-emerald-500/20 rounded-md px-3 py-2 mb-4 flex items-start space-x-2" role="status">
					<CheckCircle2 class="w-3.5 h-3.5 text-emerald-400 flex-shrink-0 mt-0.5" />
					<span class="text-xs text-emerald-400">{message}</span>
				</div>
			{/if}

			{#if error}
				<div class="bg-destructive/10 border border-destructive/20 rounded-md px-3 py-2 mb-4 flex items-center space-x-2" role="alert">
					<AlertCircle class="w-3.5 h-3.5 text-destructive flex-shrink-0" />
					<span class="text-xs text-destructive">{error}</span>
				</div>
			{/if}

			{#if !message}
				<form onsubmit={handleSubmit} class="space-y-4">
					<div>
						<label for="password" class="block text-xs font-medium text-muted-foreground mb-1.5">New password</label>
						<input
							type="password"
							id="password"
							bind:value={password}
							required
							autocomplete="new-password"
							minlength="8"
							class="w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground placeholder-muted-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
							placeholder="At least 8 characters"
						/>
					</div>

					<div>
						<label for="confirm" class="block text-xs font-medium text-muted-foreground mb-1.5">Confirm new password</label>
						<input
							type="password"
							id="confirm"
							bind:value={confirm}
							required
							autocomplete="new-password"
							minlength="8"
							class="w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground placeholder-muted-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
							placeholder="Re-enter password"
						/>
					</div>

					<button
						type="submit"
						disabled={submitting}
						class="w-full py-2.5 bg-accent text-white hover:bg-accent/90 text-sm font-medium rounded-md transition-colors disabled:opacity-50"
					>
						{submitting ? 'Updating…' : 'Update password'}
					</button>
				</form>
			{/if}

			<p class="text-center text-muted-foreground mt-5 text-xs">
				<a href="/login" class="inline-flex items-center justify-center space-x-1 text-foreground hover:text-accent transition-colors">
					<ArrowLeft class="w-3 h-3" />
					<span>Back to login</span>
				</a>
			</p>
		</div>
	</div>
</div>
