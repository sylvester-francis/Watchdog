<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { getAuth } from '$lib/stores/auth';
	import {
		ShieldCheck, Menu, X, EyeOff, Eye, Download, Radar, Bell, Server,
		Activity, BellRing, Terminal, Zap, Flame, Globe, Code, ScrollText,
		Database, Radio, LayoutDashboard, ShieldOff, Scan, Lock, Copy, Check
	} from 'lucide-svelte';

	const auth = getAuth();
	let checking = $state(true);

	// Mobile nav state
	let mobileOpen = $state(false);

	// Copy install command
	let copied = $state(false);

	// Dashboard mockup animation
	let services = $state([
		{ name: 'API Gateway', type: 'HTTP', status: 'up', latency: '12ms' },
		{ name: 'PostgreSQL', type: 'TCP', status: 'up', latency: '2ms' },
		{ name: 'Redis Cache', type: 'TCP', status: 'up', latency: '1ms' },
		{ name: 'Auth Service', type: 'HTTP', status: 'up', latency: '8ms' },
		{ name: 'Vault', type: 'HTTP', status: 'down', latency: 'timeout' },
	]);

	let bars = $state(generateBars());

	function generateBars() {
		return Array.from({ length: 24 }, () => {
			const h = 20 + Math.random() * 80;
			const isHigh = h > 70;
			return {
				style: `height: ${h}%`,
				cls: isHigh ? 'bg-amber-500/60' : 'bg-accent/40',
			};
		});
	}

	function statusDotClass(status: string) {
		return status === 'up' ? 'bg-emerald-400' : status === 'down' ? 'bg-red-400' : 'bg-zinc-500';
	}

	function latencyClass(latency: string) {
		if (latency === 'timeout') return 'text-red-400';
		const ms = parseInt(latency);
		if (ms < 10) return 'text-emerald-400';
		return 'text-muted-foreground';
	}

	function copyInstall() {
		navigator.clipboard.writeText('curl -sSL https://usewatchdog.dev/install | sh');
		copied = true;
		setTimeout(() => { copied = false; }, 2000);
	}

	let animationInterval: ReturnType<typeof setInterval>;

	onMount(async () => {
		// Redirect authenticated users to dashboard
		const user = await auth.check();
		if (user) {
			goto('/dashboard');
			return;
		}
		checking = false;

		// Animate dashboard mockup
		animationInterval = setInterval(() => {
			services = services.map(s => {
				if (s.name === 'Vault') {
					const flip = Math.random() > 0.7;
					return flip
						? { ...s, status: s.status === 'up' ? 'down' : 'up', latency: s.status === 'up' ? 'timeout' : '45ms' }
						: s;
				}
				const newLatency = Math.max(1, parseInt(s.latency) + Math.floor(Math.random() * 5 - 2));
				return { ...s, latency: `${newLatency}ms` };
			});
			bars = generateBars();
		}, 3000);

		// Scroll reveal
		const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
		const reveals = document.querySelectorAll('.reveal');
		if ('IntersectionObserver' in window && !prefersReducedMotion) {
			const observer = new IntersectionObserver((entries) => {
				entries.forEach((entry) => {
					if (entry.isIntersecting) {
						entry.target.classList.add('visible');
						observer.unobserve(entry.target);
					}
				});
			}, { threshold: 0.1 });
			reveals.forEach((el) => observer.observe(el));
		} else {
			reveals.forEach((el) => el.classList.add('visible'));
		}
	});

	onDestroy(() => {
		clearInterval(animationInterval);
	});

	const currentYear = new Date().getFullYear();

	const features = [
		{ icon: Server, title: 'Agent-based monitoring', desc: "The agent runs inside your network, so it can reach things that external monitoring tools can't \u2014 internal databases, Docker containers, private APIs, services behind a firewall or NAT." },
		{ icon: Activity, title: 'Live dashboard', desc: 'Real-time status via SSE. Latency sparklines, uptime bars, system metric charts, and incident timeline. No page refresh.' },
		{ icon: BellRing, title: 'Smart alerts', desc: 'Configurable failure threshold before firing. Supports Discord, Slack, Email, Telegram, PagerDuty, and webhooks.' },
		{ icon: Terminal, title: 'Eight check types', desc: 'HTTP with content validation, TCP, Ping, DNS, TLS certificate expiry tracking, Docker containers, database connectivity (PostgreSQL, MySQL, Redis), and system metrics with threshold alerts.' },
		{ icon: Zap, title: 'Zero config agent', desc: "The agent takes a hub URL and an API key. That's it. Monitor configs are pushed from the hub over WebSocket \u2014 no config files, no restarts." },
		{ icon: Flame, title: 'Incident tracking', desc: 'Incidents are created automatically on failure. Acknowledge, resolve, and track time-to-resolution. Full history per monitor.' },
		{ icon: Globe, title: 'Public status pages', desc: 'Share real-time service health with your team or customers. 90-day uptime history, incident timeline, and custom URLs. No authentication required to view.' },
		{ icon: Code, title: 'REST API', desc: 'Token-authenticated API for programmatic access. Manage monitors, agents, and incidents from your own scripts or CI/CD pipelines.' },
		{ icon: ScrollText, title: 'Audit trail', desc: 'Every login, monitor change, and incident action is logged with timestamps, user IDs, and IP addresses. Full event history for compliance and debugging.' },
	];

	const comparisons = [
		{ label: 'Architecture', before: 'Single external server', after: 'Distributed agents inside your network' },
		{ label: 'Internal services', before: 'Requires VPN or tunnels', after: 'Direct access \u2014 agent runs locally' },
		{ label: 'Inbound ports', before: 'Required for checks', after: 'None \u2014 agent connects outbound' },
		{ label: 'Configuration', before: 'Config files per check', after: 'Zero-config \u2014 hub pushes over WebSocket' },
		{ label: 'Alert verification', before: 'Often single-check', after: 'Configurable threshold \u2014 no false alarms' },
		{ label: 'Network exposure', before: 'Open inbound ports for probes', after: 'Zero inbound ports \u2014 outbound-only' },
		{ label: 'Data path', before: 'Credentials sent to third-party SaaS', after: 'Self-hosted \u2014 credentials never leave your infra' },
	];
</script>

<svelte:head>
	<title>WatchDog - Monitor Services Behind Your Firewall</title>
	<meta name="description" content="Deploy lightweight agents inside your network to monitor internal services, databases, and APIs. Real-time dashboard, instant alerts, zero configuration.">
</svelte:head>

{#if checking}
	<div class="flex min-h-screen items-center justify-center">
		<div class="flex flex-col items-center space-y-3">
			<div class="w-8 h-8 rounded-full bg-muted/50 animate-pulse"></div>
			<p class="text-xs text-muted-foreground">Loading...</p>
		</div>
	</div>
{:else}
	<!-- Skip to content -->
	<a href="#main-content" class="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-[100] focus:px-4 focus:py-2 focus:bg-accent focus:text-white focus:rounded-md focus:text-sm">
		Skip to main content
	</a>

	<!-- Nav -->
	<nav class="sticky top-0 z-50 bg-background/80 backdrop-blur-sm border-b border-border">
		<div class="max-w-5xl mx-auto px-4 sm:px-6 h-14 flex items-center justify-between">
			<a href="/" class="flex items-center space-x-2.5">
				<div class="w-7 h-7 bg-accent rounded-md flex items-center justify-center">
					<ShieldCheck class="w-3.5 h-3.5 text-white" />
				</div>
				<span class="text-sm font-semibold text-foreground">WatchDog</span>
			</a>
			<div class="hidden sm:flex items-center space-x-5">
				<a href="#features" class="text-sm text-muted-foreground hover:text-foreground transition-colors">Features</a>
				<a href="/login" class="text-sm text-muted-foreground hover:text-foreground transition-colors">Login</a>
				<a href="/register" class="px-3.5 py-1.5 bg-accent text-accent-foreground hover:bg-accent/90 text-sm font-medium rounded-md transition-colors">Deploy Free</a>
			</div>
			<button onclick={() => { mobileOpen = !mobileOpen; }} class="sm:hidden p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors" aria-label="Toggle navigation menu">
				{#if mobileOpen}
					<X class="w-5 h-5" />
				{:else}
					<Menu class="w-5 h-5" />
				{/if}
			</button>
		</div>
		{#if mobileOpen}
			<div class="sm:hidden border-t border-border bg-background/95 backdrop-blur-sm">
				<div class="max-w-5xl mx-auto px-4 sm:px-6 py-4 space-y-3">
					<a href="#features" onclick={() => { mobileOpen = false; }} class="block text-sm text-muted-foreground hover:text-foreground transition-colors">Features</a>
					<a href="/login" class="block text-sm text-muted-foreground hover:text-foreground transition-colors">Login</a>
					<a href="/register" class="block w-full text-center px-3.5 py-2 bg-accent text-accent-foreground hover:bg-accent/90 text-sm font-medium rounded-md transition-colors">Deploy Free</a>
				</div>
			</div>
		{/if}
	</nav>

	<!-- Hero -->
	<section id="main-content" class="relative">
		<div class="absolute inset-0 bg-[radial-gradient(ellipse_60%_40%_at_50%_-10%,rgba(59,130,246,0.06),transparent_70%)]"></div>
		<div class="max-w-5xl mx-auto px-4 sm:px-6 pt-12 pb-10 sm:pt-16 sm:pb-14 md:pt-24 md:pb-20 lg:pt-28 lg:pb-24 relative">
			<div class="grid lg:grid-cols-2 gap-8 md:gap-12 lg:gap-16 items-center">

				<!-- Left: Copy -->
				<div>
					<div class="inline-flex items-center space-x-2 px-3 py-1 rounded-full bg-accent/10 border border-accent/20 mb-4 sm:mb-6">
						<div class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse-dot"></div>
						<span class="text-xs font-medium text-accent">Open source &middot; Self-hosted &middot; Zero-trust</span>
					</div>

					<h1 class="text-2xl sm:text-3xl md:text-4xl font-bold text-foreground leading-[1.15] tracking-tight mb-3 sm:mb-4">
						Your firewall shouldn't<br>
						be a blind spot.
					</h1>
					<p class="text-sm sm:text-base text-muted-foreground max-w-lg leading-relaxed mb-6 md:mb-8">
						WatchDog deploys lightweight agents inside your network to monitor internal services, databases, and APIs that external tools can't reach. Real-time dashboard. Instant alerts. Zero configuration.
					</p>

					<div class="flex flex-col sm:flex-row gap-3 mb-6 md:mb-8">
						<a href="/register" class="w-full sm:w-auto px-5 py-2.5 bg-accent text-accent-foreground hover:bg-accent/90 text-sm font-medium rounded-md transition-colors text-center">
							Deploy Free
						</a>
						<a href="#how-it-works" class="w-full sm:w-auto px-5 py-2.5 bg-card border border-border text-foreground hover:bg-card-elevated text-sm font-medium rounded-md transition-colors text-center">
							See How It Works
						</a>
					</div>

					<!-- Install snippet -->
					<div class="bg-card border border-border rounded-lg p-3 sm:p-4 max-w-md text-left">
						<div class="flex items-center justify-between mb-2.5">
							<div class="flex items-center space-x-2">
								<div class="w-2 h-2 rounded-full bg-red-500/60"></div>
								<div class="w-2 h-2 rounded-full bg-yellow-500/60"></div>
								<div class="w-2 h-2 rounded-full bg-emerald-500/60"></div>
								<span class="text-[10px] text-muted-foreground ml-1 font-mono">terminal</span>
							</div>
							<button onclick={copyInstall} class="text-muted-foreground hover:text-foreground transition-colors p-1 rounded" aria-label="Copy install command">
								{#if copied}
									<Check class="w-3.5 h-3.5 text-emerald-400" />
								{:else}
									<Copy class="w-3.5 h-3.5" />
								{/if}
							</button>
						</div>
						<div class="font-mono text-xs sm:text-sm text-muted-foreground leading-relaxed overflow-x-auto">
							<div class="whitespace-nowrap"><span class="text-muted-foreground/50">$</span> <span class="text-foreground">curl -sSL https://usewatchdog.dev/install | sh</span></div>
							<div class="mt-1 whitespace-nowrap text-muted-foreground/70">  API Key: <span class="text-foreground">&#x2022;&#x2022;&#x2022;&#x2022;&#x2022;&#x2022;&#x2022;&#x2022;&#x2022;&#x2022;&#x2022;&#x2022;</span></div>
							<div class="mt-1.5 text-emerald-400">  Done! Your agent will appear in the dashboard shortly.</div>
						</div>
					</div>
				</div>

				<!-- Right: Animated Dashboard Mockup -->
				<div class="hidden lg:block">
					<div class="bg-card border border-border rounded-xl overflow-hidden shadow-2xl shadow-black/20">
						<div class="flex items-center space-x-2 px-4 py-2.5 border-b border-border bg-card-elevated">
							<div class="w-2.5 h-2.5 rounded-full bg-red-500/50"></div>
							<div class="w-2.5 h-2.5 rounded-full bg-yellow-500/50"></div>
							<div class="w-2.5 h-2.5 rounded-full bg-emerald-500/50"></div>
							<span class="text-[10px] text-muted-foreground font-mono ml-2">dashboard</span>
						</div>

						<div class="p-4 space-y-3">
							<!-- Mini stat cards -->
							<div class="grid grid-cols-3 gap-2">
								<div class="bg-card-elevated rounded-md p-2.5 border border-border/50">
									<p class="text-[9px] text-muted-foreground uppercase tracking-wider mb-1">Monitors</p>
									<p class="text-lg font-bold text-foreground font-mono">12</p>
								</div>
								<div class="bg-card-elevated rounded-md p-2.5 border border-border/50">
									<p class="text-[9px] text-muted-foreground uppercase tracking-wider mb-1">Healthy</p>
									<p class="text-lg font-bold text-emerald-400 font-mono">{services.filter(s => s.status === 'up').length + 7}</p>
								</div>
								<div class="bg-card-elevated rounded-md p-2.5 border border-border/50">
									<p class="text-[9px] text-muted-foreground uppercase tracking-wider mb-1">Uptime</p>
									<p class="text-lg font-bold text-foreground font-mono">99.9<span class="text-xs text-muted-foreground">%</span></p>
								</div>
							</div>

							<!-- Service status list -->
							<div class="bg-card-elevated rounded-md border border-border/50">
								<div class="px-3 py-2 border-b border-border/30">
									<p class="text-[9px] text-muted-foreground uppercase tracking-wider">Service Status</p>
								</div>
								<div class="divide-y divide-border/20">
									{#each services as service}
										<div class="flex items-center justify-between px-3 py-2">
											<div class="flex items-center space-x-2">
												<div class="w-1.5 h-1.5 rounded-full {statusDotClass(service.status)}"></div>
												<span class="text-xs text-foreground">{service.name}</span>
												<span class="text-[9px] text-muted-foreground font-mono uppercase">{service.type}</span>
											</div>
											<span class="text-[10px] font-mono {latencyClass(service.latency)}">{service.latency}</span>
										</div>
									{/each}
								</div>
							</div>

							<!-- Mini bar chart -->
							<div class="bg-card-elevated rounded-md p-3 border border-border/50">
								<p class="text-[9px] text-muted-foreground uppercase tracking-wider mb-2">Response Time (ms)</p>
								<div class="flex items-end space-x-1 h-16">
									{#each bars as bar}
										<div class="flex-1 rounded-sm transition-all duration-700 {bar.cls}" style={bar.style}></div>
									{/each}
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	</section>

	<!-- Problem Section -->
	<section class="py-12 md:py-16 lg:py-20 border-t border-border reveal">
		<div class="max-w-5xl mx-auto px-4 sm:px-6">
			<h2 class="text-xl md:text-2xl font-bold text-foreground mb-6 md:mb-10 text-center">The blind spot in your monitoring</h2>
			<div class="grid md:grid-cols-2 gap-6 md:gap-12">
				<div>
					<div class="w-10 h-10 bg-red-500/10 rounded-lg flex items-center justify-center mb-3 md:mb-4">
						<EyeOff class="w-5 h-5 text-red-400" />
					</div>
					<h3 class="text-sm font-semibold text-foreground mb-2">The problem</h3>
					<p class="text-sm text-muted-foreground leading-relaxed">
						Most monitoring tools run checks from a single external server. If your service is behind a firewall, NAT, or VPC &mdash; they can't see it. You're flying blind on the infrastructure that matters most.
					</p>
				</div>
				<div>
					<div class="w-10 h-10 bg-emerald-500/10 rounded-lg flex items-center justify-center mb-3 md:mb-4">
						<Eye class="w-5 h-5 text-emerald-400" />
					</div>
					<h3 class="text-sm font-semibold text-foreground mb-2">The WatchDog approach</h3>
					<p class="text-sm text-muted-foreground leading-relaxed">
						WatchDog flips the model. A lightweight agent runs inside your network, checking services locally and reporting back to a central hub. No inbound ports. No VPN tunnels. No blind spots.
					</p>
				</div>
			</div>
		</div>
	</section>

	<!-- How It Works -->
	<section class="py-12 md:py-16 lg:py-20 border-t border-border reveal" id="how-it-works">
		<div class="max-w-5xl mx-auto px-4 sm:px-6">
			<h2 class="text-xl md:text-2xl font-bold text-foreground mb-6 md:mb-10 text-center">How it works</h2>
			<div class="grid md:grid-cols-3 gap-6 md:gap-8 relative">
				<div class="hidden md:block absolute top-5 left-[calc(33.33%+0.5rem)] right-[calc(33.33%+0.5rem)] h-px bg-border"></div>

				<div class="text-center relative">
					<div class="w-10 h-10 bg-muted/50 rounded-lg flex items-center justify-center mx-auto mb-3 relative z-10">
						<Download class="w-5 h-5 text-muted-foreground" />
					</div>
					<h3 class="text-sm font-semibold text-foreground mb-1.5">1. Deploy an agent</h3>
					<p class="text-sm text-muted-foreground leading-relaxed">Single Go binary. Run it inside your network &mdash; a VPC, a homelab, behind a NAT. No config file needed.</p>
				</div>
				<div class="text-center relative">
					<div class="w-10 h-10 bg-muted/50 rounded-lg flex items-center justify-center mx-auto mb-3 relative z-10">
						<Radar class="w-5 h-5 text-muted-foreground" />
					</div>
					<h3 class="text-sm font-semibold text-foreground mb-1.5">2. Agent runs checks</h3>
					<p class="text-sm text-muted-foreground leading-relaxed">HTTP, TCP, DNS, TLS, Docker, database, and system metrics. The hub pushes monitor configs to the agent over WebSocket. The agent reports back.</p>
				</div>
				<div class="text-center relative">
					<div class="w-10 h-10 bg-muted/50 rounded-lg flex items-center justify-center mx-auto mb-3 relative z-10">
						<Bell class="w-5 h-5 text-muted-foreground" />
					</div>
					<h3 class="text-sm font-semibold text-foreground mb-1.5">3. Get alerted</h3>
					<p class="text-sm text-muted-foreground leading-relaxed">Configurable failure threshold (1-10 consecutive failures) before alerting. Get notified via Discord, Slack, Email, Telegram, PagerDuty, or webhooks.</p>
				</div>
			</div>
		</div>
	</section>

	<!-- Features -->
	<section class="py-12 md:py-16 lg:py-20 border-t border-border reveal" id="features">
		<div class="max-w-5xl mx-auto px-4 sm:px-6">
			<h2 class="text-xl md:text-2xl font-bold text-foreground mb-2 sm:mb-3 text-center">Features</h2>
			<p class="text-sm text-muted-foreground text-center mb-6 md:mb-10">What you get out of the box.</p>

			<div class="grid sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
				{#each features as f}
					<div class="bg-card border border-border rounded-xl p-4 sm:p-6 card-hover min-w-0 overflow-hidden">
						<div class="w-10 h-10 bg-muted/50 rounded-lg flex items-center justify-center mb-3">
							<f.icon class="w-5 h-5 text-muted-foreground" />
						</div>
						<h3 class="text-sm font-semibold text-foreground mb-1.5">{f.title}</h3>
						<p class="text-sm text-muted-foreground leading-relaxed break-words">{f.desc}</p>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<!-- Why WatchDog -->
	<section class="py-12 md:py-16 lg:py-20 border-t border-border reveal">
		<div class="max-w-5xl mx-auto px-4 sm:px-6">
			<h2 class="text-xl md:text-2xl font-bold text-foreground mb-2 sm:mb-3 text-center">Why WatchDog</h2>
			<p class="text-sm text-muted-foreground text-center mb-6 md:mb-10">How agent-based monitoring compares to traditional tools.</p>

			<!-- Mobile: stacked cards -->
			<div class="md:hidden space-y-3">
				{#each comparisons as c}
					<div class="bg-card border border-border rounded-lg p-4">
						<p class="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">{c.label}</p>
						<div class="flex items-start space-x-2 mb-1.5">
							<span class="text-muted-foreground/50 text-xs mt-0.5 shrink-0">Before:</span>
							<span class="text-xs text-muted-foreground">{c.before}</span>
						</div>
						<div class="flex items-start space-x-2">
							<span class="text-emerald-400/70 text-xs mt-0.5 shrink-0">WatchDog:</span>
							<span class="text-xs text-foreground">{c.after}</span>
						</div>
					</div>
				{/each}
			</div>

			<!-- Desktop: table -->
			<div class="hidden md:block overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-border">
							<th class="text-left py-3 px-4 text-muted-foreground font-medium"></th>
							<th class="text-left py-3 px-4 text-muted-foreground font-medium">Traditional Monitoring</th>
							<th class="text-left py-3 px-4 text-foreground font-semibold">WatchDog</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border/50">
						{#each comparisons as c}
							<tr>
								<td class="py-3 px-4 text-muted-foreground font-medium">{c.label}</td>
								<td class="py-3 px-4 text-muted-foreground">{c.before}</td>
								<td class="py-3 px-4 text-foreground">{c.after}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	</section>

	<!-- Architecture -->
	<section class="py-12 md:py-16 lg:py-20 border-t border-border reveal">
		<div class="max-w-5xl mx-auto px-4 sm:px-6">
			<h2 class="text-xl md:text-2xl font-bold text-foreground mb-2 sm:mb-3 text-center">Architecture</h2>
			<p class="text-sm text-muted-foreground text-center mb-6 md:mb-10">Three components. One binary each.</p>

			<div class="grid sm:grid-cols-2 md:grid-cols-3 gap-4 md:gap-6">
				<div class="bg-card border border-border rounded-xl p-4 sm:p-6">
					<div class="w-10 h-10 bg-muted/50 rounded-lg flex items-center justify-center mb-3">
						<Database class="w-5 h-5 text-muted-foreground" />
					</div>
					<h3 class="text-sm font-semibold text-foreground mb-1.5">Hub</h3>
					<p class="text-sm text-muted-foreground leading-relaxed">Central server. Stores configs, processes heartbeats, serves the dashboard, fires alerts. Go + PostgreSQL + TimescaleDB.</p>
				</div>
				<div class="bg-card border border-border rounded-xl p-4 sm:p-6">
					<div class="w-10 h-10 bg-muted/50 rounded-lg flex items-center justify-center mb-3">
						<Radio class="w-5 h-5 text-muted-foreground" />
					</div>
					<h3 class="text-sm font-semibold text-foreground mb-1.5">Agent</h3>
					<p class="text-sm text-muted-foreground leading-relaxed">Lightweight Go binary that runs inside your network. Connects to the hub over WebSocket. Receives check configs, reports results.</p>
				</div>
				<div class="bg-card border border-border rounded-xl p-4 sm:p-6">
					<div class="w-10 h-10 bg-muted/50 rounded-lg flex items-center justify-center mb-3">
						<LayoutDashboard class="w-5 h-5 text-muted-foreground" />
					</div>
					<h3 class="text-sm font-semibold text-foreground mb-1.5">Dashboard</h3>
					<p class="text-sm text-muted-foreground leading-relaxed">Real-time web UI served by the hub. Live updates via SSE. Public status pages. SvelteKit SPA.</p>
				</div>
			</div>

			<div class="mt-6 md:mt-8 flex flex-wrap items-center justify-center gap-2 sm:gap-3">
				<span class="text-xs text-muted-foreground">Built with:</span>
				{#each ['Go', 'PostgreSQL', 'TimescaleDB', 'WebSocket', 'SvelteKit'] as tech}
					<span class="px-2.5 py-1 rounded-md bg-card border border-border text-xs text-muted-foreground font-mono">{tech}</span>
				{/each}
			</div>
		</div>
	</section>

	<!-- Security -->
	<section class="py-12 md:py-16 lg:py-20 border-t border-border reveal" id="security">
		<div class="max-w-5xl mx-auto px-4 sm:px-6">
			<h2 class="text-xl md:text-2xl font-bold text-foreground mb-2 sm:mb-3 text-center">Security by design</h2>
			<p class="text-sm text-muted-foreground text-center mb-6 md:mb-10">Your infrastructure credentials never leave your network.</p>

			<div class="grid sm:grid-cols-2 md:grid-cols-3 gap-4 md:gap-6">
				<div class="bg-card border border-border rounded-xl p-4 sm:p-6 card-hover">
					<div class="w-10 h-10 bg-muted/50 rounded-lg flex items-center justify-center mb-3">
						<ShieldOff class="w-5 h-5 text-muted-foreground" />
					</div>
					<h3 class="text-sm font-semibold text-foreground mb-1.5">Zero inbound ports</h3>
					<p class="text-sm text-muted-foreground leading-relaxed">The agent connects outbound to the hub over WebSocket. No listening ports, no attack surface on your network.</p>
				</div>
				<div class="bg-card border border-border rounded-xl p-4 sm:p-6 card-hover">
					<div class="w-10 h-10 bg-muted/50 rounded-lg flex items-center justify-center mb-3">
						<Scan class="w-5 h-5 text-muted-foreground" />
					</div>
					<h3 class="text-sm font-semibold text-foreground mb-1.5">Minimal attack surface</h3>
					<p class="text-sm text-muted-foreground leading-relaxed">Single static Go binary. No runtime dependencies, no interpreters, no package managers. Agent fingerprinting detects impersonation.</p>
				</div>
				<div class="bg-card border border-border rounded-xl p-4 sm:p-6 card-hover">
					<div class="w-10 h-10 bg-muted/50 rounded-lg flex items-center justify-center mb-3">
						<Lock class="w-5 h-5 text-muted-foreground" />
					</div>
					<h3 class="text-sm font-semibold text-foreground mb-1.5">Credentials stay local</h3>
					<p class="text-sm text-muted-foreground leading-relaxed">Self-hosted hub means your database passwords, API keys, and service endpoints never leave your infrastructure.</p>
				</div>
			</div>
		</div>
	</section>

	<!-- CTA -->
	<section class="py-12 md:py-16 lg:py-20 border-t border-border reveal">
		<div class="max-w-xl mx-auto px-4 sm:px-6 text-center">
			<h2 class="text-xl md:text-2xl font-bold text-foreground mb-2 sm:mb-3">Deploy your first agent</h2>
			<p class="text-sm text-muted-foreground mb-5 sm:mb-6">Sign up, deploy an agent, and start monitoring in a few minutes.</p>
			<div class="flex flex-col sm:flex-row items-center justify-center gap-3">
				<a href="/register" class="w-full sm:w-auto px-5 py-2.5 bg-accent text-accent-foreground hover:bg-accent/90 text-sm font-medium rounded-md transition-colors text-center">
					Deploy Free
				</a>
				<a href="#how-it-works" class="w-full sm:w-auto px-5 py-2.5 bg-card border border-border text-foreground hover:bg-card-elevated text-sm font-medium rounded-md transition-colors text-center">
					See How It Works
				</a>
			</div>
		</div>
	</section>

	<!-- Footer -->
	<footer class="border-t border-border py-8 md:py-10">
		<div class="max-w-5xl mx-auto px-4 sm:px-6">
			<div class="grid grid-cols-2 sm:grid-cols-2 lg:grid-cols-4 gap-6 sm:gap-8">
				<div class="col-span-2 sm:col-span-1">
					<a href="/" class="flex items-center space-x-2 mb-3">
						<div class="w-6 h-6 bg-accent rounded-md flex items-center justify-center">
							<ShieldCheck class="w-3 h-3 text-white" />
						</div>
						<span class="text-sm font-semibold text-foreground">WatchDog</span>
					</a>
					<p class="text-xs text-muted-foreground leading-relaxed">Open-source infrastructure monitoring for services behind your firewall.</p>
				</div>
				<div>
					<h4 class="text-xs font-semibold text-foreground uppercase tracking-wider mb-3">Product</h4>
					<ul class="space-y-2">
						<li><a href="#features" class="text-xs text-muted-foreground hover:text-foreground transition-colors">Features</a></li>
						<li><a href="/docs" class="text-xs text-muted-foreground hover:text-foreground transition-colors">API Docs</a></li>
					</ul>
				</div>
				<div>
					<h4 class="text-xs font-semibold text-foreground uppercase tracking-wider mb-3">Community</h4>
					<ul class="space-y-2">
						<li><a href="https://github.com/sylvester-francis/Watchdog" target="_blank" rel="noopener noreferrer" class="text-xs text-muted-foreground hover:text-foreground transition-colors">GitHub</a></li>
						<li><a href="https://discord.gg/PPPjZDVS" target="_blank" rel="noopener noreferrer" class="text-xs text-muted-foreground hover:text-foreground transition-colors">Discord</a></li>
					</ul>
				</div>
				<div>
					<h4 class="text-xs font-semibold text-foreground uppercase tracking-wider mb-3">Legal</h4>
					<ul class="space-y-2">
						<li><a href="/terms" class="text-xs text-muted-foreground hover:text-foreground transition-colors">Terms</a></li>
						<li><a href="/privacy" class="text-xs text-muted-foreground hover:text-foreground transition-colors">Privacy</a></li>
					</ul>
				</div>
			</div>

			<div class="mt-6 sm:mt-8 pt-5 sm:pt-6 border-t border-border/50 flex flex-col sm:flex-row items-center justify-between gap-2">
				<p class="text-[11px] text-muted-foreground/60">&copy; {currentYear} Sylvester Ranjith Francis</p>
				<p class="text-[11px] text-muted-foreground/60">Open-source infrastructure monitoring &middot; <a href="https://www.gnu.org/licenses/agpl-3.0.html" target="_blank" rel="noopener noreferrer" class="hover:text-muted-foreground transition-colors">AGPL-3.0</a></p>
			</div>
		</div>
	</footer>
{/if}
