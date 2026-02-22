<script lang="ts">
	import { goto } from '$app/navigation';
	import { base } from '$app/paths';
	import { ShieldCheck, AlertCircle } from 'lucide-svelte';
	import { getAuth } from '$lib/stores/auth';

	const auth = getAuth();

	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');
	let submitting = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!email || !password) {
			error = 'Email and password are required';
			return;
		}
		if (password.length < 8) {
			error = 'Password must be at least 8 characters';
			return;
		}
		if (password !== confirmPassword) {
			error = 'Passwords do not match';
			return;
		}
		error = '';
		submitting = true;
		try {
			await auth.register(email, password, confirmPassword);
			goto(`${base}/login?success=${encodeURIComponent('Account created successfully. Please login.')}`);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Registration failed';
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>Register - WatchDog</title>
</svelte:head>

<div class="flex min-h-screen">
	<!-- Left Panel (same as login) -->
	<div class="hidden lg:flex lg:w-1/2 relative overflow-hidden">
		<div class="absolute inset-0 bg-background">
			<div class="absolute top-0 left-0 w-[500px] h-[500px] rounded-full bg-accent/[0.12] blur-[120px] animate-pulse-slow"></div>
			<div class="absolute bottom-0 right-0 w-[400px] h-[400px] rounded-full bg-blue-500/[0.08] blur-[100px] animate-pulse-slow" style="animation-delay: 2s;"></div>
		</div>
		<div class="relative z-10 flex flex-col justify-between p-12 w-full">
			<a href="/" class="flex items-center space-x-2.5">
				<div class="w-8 h-8 bg-accent rounded-lg flex items-center justify-center">
					<ShieldCheck class="w-4 h-4 text-white" />
				</div>
				<span class="text-base font-semibold text-foreground">WatchDog</span>
			</a>
			<div class="max-w-sm">
				<h2 class="text-2xl font-bold text-foreground leading-tight mb-4">
					Infrastructure monitoring that reaches where others can't.
				</h2>
				<p class="text-sm text-muted-foreground leading-relaxed">
					Deploy agents inside your network to monitor internal services, databases, and APIs behind firewalls and NATs.
				</p>
			</div>
			<p class="text-[11px] text-muted-foreground/50">Open-source infrastructure monitoring</p>
		</div>
	</div>

	<!-- Right Panel: Register Form -->
	<div class="flex-1 flex items-center justify-center p-6 lg:p-12">
		<div class="w-full max-w-sm animate-fade-in-up">
			<div class="lg:hidden text-center mb-8">
				<a href="/">
					<div class="inline-flex items-center justify-center w-10 h-10 bg-accent rounded-lg mb-3">
						<ShieldCheck class="w-5 h-5 text-white" />
					</div>
					<h1 class="text-lg font-semibold text-foreground">WatchDog</h1>
				</a>
			</div>

			<div class="bg-card border border-border/50 rounded-lg p-6">
				<h2 class="text-base font-semibold text-foreground mb-1">Create your account</h2>
				<p class="text-xs text-muted-foreground mb-5">Get started with WatchDog monitoring.</p>

				{#if error}
					<div class="bg-destructive/10 border border-destructive/20 rounded-md px-3 py-2 mb-4 flex items-center space-x-2" role="alert">
						<AlertCircle class="w-3.5 h-3.5 text-destructive flex-shrink-0" />
						<span class="text-xs text-destructive">{error}</span>
					</div>
				{/if}

				<form onsubmit={handleSubmit} class="space-y-4">
					<div>
						<label for="email" class="block text-xs font-medium text-muted-foreground mb-1.5">Email</label>
						<input type="email" id="email" bind:value={email} required autocomplete="email"
							class="w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background"
							placeholder="you@example.com" />
					</div>
					<div>
						<label for="password" class="block text-xs font-medium text-muted-foreground mb-1.5">Password</label>
						<input type="password" id="password" bind:value={password} required minlength="8" autocomplete="new-password"
							class="w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background"
							placeholder="At least 8 characters" />
					</div>
					<div>
						<label for="confirm_password" class="block text-xs font-medium text-muted-foreground mb-1.5">Confirm Password</label>
						<input type="password" id="confirm_password" bind:value={confirmPassword} required minlength="8" autocomplete="new-password"
							class="w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background"
							placeholder="Confirm your password" />
					</div>

					<button type="submit" disabled={submitting}
						class="w-full py-2.5 bg-accent text-white hover:bg-accent/90 text-sm font-medium rounded-md transition-colors disabled:opacity-50">
						{submitting ? 'Creating account...' : 'Create Account'}
					</button>
				</form>

				<p class="text-center text-muted-foreground mt-5 text-xs">
					Already have an account?
					<a href="{base}/login" class="text-foreground hover:text-accent transition-colors">Sign in</a>
				</p>
			</div>

			<p class="text-center mt-6 text-[11px] text-muted-foreground/50">
				<a href="/" class="hover:text-muted-foreground transition-colors">&larr; Back to WatchDog</a>
			</p>
		</div>
	</div>
</div>
