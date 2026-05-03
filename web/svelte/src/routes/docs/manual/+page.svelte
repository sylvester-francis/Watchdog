<script lang="ts">
	import { onMount } from 'svelte';
	import { ShieldCheck, ExternalLink } from 'lucide-svelte';
	import { getAuth } from '$lib/stores/auth.svelte';

	const auth = getAuth();

	onMount(() => {
		auth.check();
	});
</script>

<svelte:head>
	<title>User Guide - WatchDog</title>
	<meta name="description" content="Get started with WatchDog — sign up, deploy an agent, create monitors, set up alerts, and build status pages in minutes." />
</svelte:head>

<div class="min-h-screen bg-background text-foreground font-sans antialiased">
	<!-- Top bar -->
	<header class="sticky top-0 z-30 border-b border-border bg-background/80 backdrop-blur-sm">
		<div class="max-w-3xl mx-auto flex items-center h-14 px-4 lg:px-8">
			<a href="/" class="flex items-center space-x-2.5 shrink-0">
				<div class="w-7 h-7 bg-accent rounded-lg flex items-center justify-center">
					<ShieldCheck class="w-3.5 h-3.5 text-white" />
				</div>
				<span class="text-sm font-semibold text-foreground">WatchDog</span>
			</a>
			<span class="text-border mx-3">|</span>
			<span class="text-sm text-muted-foreground">User Guide</span>

			<div class="ml-auto flex items-center space-x-3">
				<a href="/docs" class="hidden sm:inline-flex items-center space-x-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors">
					<span>API Reference</span>
					<ExternalLink class="w-3 h-3" />
				</a>
				{#if auth.isAuthenticated}
					<a href="/dashboard" class="text-xs text-accent hover:text-accent/80 transition-colors font-medium">Dashboard</a>
				{:else if !auth.loading}
					<a href="/login" class="text-xs text-accent hover:text-accent/80 transition-colors font-medium">Sign In</a>
				{/if}
			</div>
		</div>
	</header>

	<main class="max-w-3xl mx-auto px-4 lg:px-8">
		<!-- Hero -->
		<section class="pt-10 pb-8 border-b border-border/50">
			<p class="text-xs font-medium text-accent mb-2">User Guide</p>
			<h1 class="text-2xl font-bold text-foreground tracking-tight leading-tight mb-2">
				Get up and running with WatchDog
			</h1>
			<p class="text-sm text-muted-foreground leading-relaxed max-w-xl">
				From sign-up to your first alert in under 10 minutes. This guide walks you through everything you need to start monitoring your infrastructure.
			</p>
		</section>

		<!-- TOC -->
		<section class="py-6 border-b border-border/50">
			<h2 class="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">In this guide</h2>
			<div class="grid sm:grid-cols-2 gap-1.5">
				{#each [
					['#signup', '01', 'Sign up'],
					['#agents', '02', 'Deploy an agent'],
					['#monitors', '03', 'Create monitors'],
					['#alerts', '04', 'Set up alerts'],
					['#status-pages', '05', 'Status pages'],
					['#telemetry', '06', 'Send traces & logs']
				] as [href, num, label]}
					<a {href} class="flex items-center gap-2 text-sm text-foreground hover:text-accent transition-colors py-1">
						<span class="text-accent font-mono text-xs">{num}</span> {label}
					</a>
				{/each}
			</div>
		</section>

		<!-- Step 1: Sign Up -->
		<section id="signup" class="py-10 border-b border-border/50 scroll-mt-16">
			<div class="flex items-center gap-3 mb-4">
				<span class="w-8 h-8 rounded-md bg-accent/10 flex items-center justify-center text-accent font-mono text-sm font-bold">1</span>
				<h2 class="text-lg font-semibold text-foreground">Sign up</h2>
			</div>
			<div class="space-y-4 text-sm text-muted-foreground leading-relaxed">
				<p>
					On a fresh self-hosted install, the first user creates the admin account at <code class="text-foreground font-mono text-xs">/setup</code>. After that, you sign in at <a href="/login" class="text-accent hover:underline">/login</a>. If you're using the hosted version at <a href="https://usewatchdog.dev" class="text-accent hover:underline">usewatchdog.dev</a>, register at <a href="/register" class="text-accent hover:underline">/register</a> with your email and password.
				</p>
				<p>
					Every authenticated user has access to all features — monitors, agents, status pages, settings, audit log. Your data is private to your account; other users on the same hub can't see your monitors, incidents, or telemetry.
				</p>
			</div>
		</section>

		<!-- Step 2: Deploy an Agent -->
		<section id="agents" class="py-10 border-b border-border/50 scroll-mt-16">
			<div class="flex items-center gap-3 mb-4">
				<span class="w-8 h-8 rounded-md bg-accent/10 flex items-center justify-center text-accent font-mono text-sm font-bold">2</span>
				<h2 class="text-lg font-semibold text-foreground">Deploy an agent</h2>
			</div>
			<div class="space-y-4 text-sm text-muted-foreground leading-relaxed">
				<p>
					Agents are lightweight binaries that run on your servers and execute monitoring checks. They connect to the WatchDog hub over WebSocket, so they work behind firewalls and NATs with no inbound ports required.
				</p>
				<div>
					<p class="text-foreground font-medium mb-2">Create an agent</p>
					<ol class="list-decimal list-inside space-y-1.5 ml-1">
						<li>Open the <strong class="text-foreground">Dashboard</strong>.</li>
						<li>Click <strong class="text-foreground">New Agent</strong> and give it a name (e.g., "production-web-1").</li>
						<li><strong class="text-foreground">Copy the API key</strong> &mdash; it's shown only once.</li>
					</ol>
				</div>
				<div>
					<p class="text-foreground font-medium mb-2">Install on your server</p>
					<p class="mb-2">Run the one-liner on any Linux machine:</p>
					<div class="bg-card-elevated border border-border/50 rounded-md p-3 font-mono text-xs overflow-x-auto">
						<pre class="text-muted-foreground">curl -sSL https://usewatchdog.dev/install | sh -s -- --api-key YOUR_API_KEY</pre>
					</div>
					<p class="mt-2 text-xs text-muted-foreground/60">This downloads the agent, installs it as a systemd service, and starts it automatically.</p>
				</div>
				<div>
					<p class="text-foreground font-medium mb-2">Verify</p>
					<p>Go back to the dashboard. Your agent should show a green <strong class="text-foreground">Online</strong> badge within a few seconds.</p>
				</div>
				<div>
					<p class="text-foreground font-medium mb-2">Useful commands</p>
					<div class="bg-card-elevated border border-border/50 rounded-md p-3 font-mono text-xs overflow-x-auto space-y-1">
						<p class="text-muted-foreground">sudo systemctl status watchdog-agent &nbsp; <span class="text-muted-foreground/50"># Check status</span></p>
						<p class="text-muted-foreground">sudo systemctl restart watchdog-agent <span class="text-muted-foreground/50"># Restart</span></p>
						<p class="text-muted-foreground">sudo journalctl -u watchdog-agent -f &nbsp; <span class="text-muted-foreground/50"># View logs</span></p>
					</div>
				</div>
			</div>
		</section>

		<!-- Step 3: Create Monitors -->
		<section id="monitors" class="py-10 border-b border-border/50 scroll-mt-16">
			<div class="flex items-center gap-3 mb-4">
				<span class="w-8 h-8 rounded-md bg-accent/10 flex items-center justify-center text-accent font-mono text-sm font-bold">3</span>
				<h2 class="text-lg font-semibold text-foreground">Create monitors</h2>
			</div>
			<div class="space-y-4 text-sm text-muted-foreground leading-relaxed">
				<p>Monitors define what to check and how often. Each monitor runs on a specific agent.</p>
				<ol class="list-decimal list-inside space-y-1.5 ml-1">
					<li>Go to <strong class="text-foreground">Monitors</strong> and click <strong class="text-foreground">New Monitor</strong>.</li>
					<li>Select the <strong class="text-foreground">Agent</strong> that will run the check.</li>
					<li>Choose a <strong class="text-foreground">Type</strong> and enter the <strong class="text-foreground">Target</strong>.</li>
					<li>Set the check interval, timeout, and failure threshold.</li>
					<li>Click <strong class="text-foreground">Create</strong>.</li>
				</ol>
				<div>
					<p class="text-foreground font-medium mb-2">Monitor types</p>
					<div class="grid sm:grid-cols-2 gap-2">
						{#each [
							['HTTP', 'Check URLs for expected status codes and response times'],
							['TCP', 'Verify that a port is open and accepting connections'],
							['Ping', 'ICMP ping to check host reachability'],
							['DNS', 'Validate DNS records resolve correctly'],
							['TLS/SSL', 'Monitor certificate expiry and validity'],
							['Docker', 'Check container health and running state'],
							['Database', 'Test connectivity to PostgreSQL, MySQL, Redis, etc.'],
							['System', 'Track CPU, memory, and disk on the agent host'],
							['Service', 'Verify a systemd or Windows service is running'],
							['Port Scan', 'Scan multiple ports on a host with service detection'],
							['SNMP', 'Poll v2c/v3 metrics from network devices (Cisco, MikroTik, Ubiquiti, APC, more)']
						] as [name, desc]}
							<div class="flex items-start gap-2 py-1.5">
								<span class="text-accent text-xs mt-0.5">&bull;</span>
								<div>
									<span class="text-foreground text-xs font-medium">{name}</span>
									<span class="text-xs text-muted-foreground/70"> &mdash; {desc}</span>
								</div>
							</div>
						{/each}
					</div>
				</div>
				<div>
					<p class="text-foreground font-medium mb-2">How incidents work</p>
					<p>
						When a monitor fails consecutively (default: 3 times), WatchDog opens an incident and triggers your configured alerts. When the monitor recovers, the incident is automatically resolved and you're notified of the recovery.
					</p>
				</div>
			</div>
		</section>

		<!-- Step 4: Set Up Alerts -->
		<section id="alerts" class="py-10 border-b border-border/50 scroll-mt-16">
			<div class="flex items-center gap-3 mb-4">
				<span class="w-8 h-8 rounded-md bg-accent/10 flex items-center justify-center text-accent font-mono text-sm font-bold">4</span>
				<h2 class="text-lg font-semibold text-foreground">Set up alerts</h2>
			</div>
			<div class="space-y-4 text-sm text-muted-foreground leading-relaxed">
				<p>Alert channels define where notifications go when an incident is triggered or resolved.</p>
				<ol class="list-decimal list-inside space-y-1.5 ml-1">
					<li>Go to <strong class="text-foreground">Settings → Alert Channels</strong> and click <strong class="text-foreground">New Channel</strong>.</li>
					<li>Choose the channel type and enter the required configuration.</li>
					<li>Click <strong class="text-foreground">Create</strong>.</li>
				</ol>
				<div>
					<p class="text-foreground font-medium mb-2">Supported channels</p>
					<div class="grid sm:grid-cols-2 gap-3">
						{#each [
							['Slack', 'Send to a channel via incoming webhook URL'],
							['Discord', 'Post to a channel via webhook URL'],
							['Email', 'Send alerts to one or more email addresses'],
							['Telegram', 'Send messages via bot token + chat ID'],
							['PagerDuty', 'Trigger incidents via integration key'],
							['Webhook', 'POST JSON payloads to any URL']
						] as [name, desc]}
							<div class="bg-card border border-border/50 rounded-md p-3">
								<p class="text-xs font-medium text-foreground">{name}</p>
								<p class="text-xs text-muted-foreground/70 mt-0.5">{desc}</p>
							</div>
						{/each}
					</div>
				</div>
				<p>Each channel has a <strong class="text-foreground">Test</strong> button to send a sample notification before relying on it for production alerts.</p>
			</div>
		</section>

		<!-- Step 5: Status Pages -->
		<section id="status-pages" class="py-10 border-b border-border/50 scroll-mt-16">
			<div class="flex items-center gap-3 mb-4">
				<span class="w-8 h-8 rounded-md bg-accent/10 flex items-center justify-center text-accent font-mono text-sm font-bold">5</span>
				<h2 class="text-lg font-semibold text-foreground">Status pages</h2>
			</div>
			<div class="space-y-4 text-sm text-muted-foreground leading-relaxed">
				<p>Share a public or private status page with your customers so they can see the health of your services at a glance.</p>
				<ol class="list-decimal list-inside space-y-1.5 ml-1">
					<li>Go to <strong class="text-foreground">Status Pages</strong> and click <strong class="text-foreground">New Status Page</strong>.</li>
					<li>Give it a name and optional description.</li>
					<li>Select which monitors to display.</li>
					<li>Choose <strong class="text-foreground">Public</strong> (anyone with the link) or <strong class="text-foreground">Private</strong> (authenticated users only).</li>
					<li>Click <strong class="text-foreground">Create</strong> and share the link.</li>
				</ol>
				<p>Status pages update in real-time and show 90 days of uptime history, an aggregate uptime percentage, and recent incident history per service.</p>
			</div>
		</section>

		<!-- Step 6: Send traces & logs -->
		<section id="telemetry" class="py-10 border-b border-border/50 scroll-mt-16">
			<div class="flex items-center gap-3 mb-4">
				<span class="w-8 h-8 rounded-md bg-accent/10 flex items-center justify-center text-accent font-mono text-sm font-bold">6</span>
				<h2 class="text-lg font-semibold text-foreground">Send traces &amp; logs</h2>
			</div>
			<div class="space-y-4 text-sm text-muted-foreground leading-relaxed">
				<p>WatchDog's hub speaks OTLP/HTTP natively. Any OpenTelemetry collector or SDK can push traces and logs directly — no separate Tempo, Loki, or Jaeger required.</p>

				<ol class="list-decimal list-inside space-y-1.5 ml-1">
					<li>Open <strong class="text-foreground">Settings → API Tokens</strong> and click <strong class="text-foreground">Create</strong>. Pick the <strong class="text-foreground">Telemetry</strong> scope. Copy the token (it's only shown once).</li>
					<li>Point your collector or SDK at <code class="text-foreground font-mono text-xs">https://usewatchdog.dev</code> (or your self-hosted hub URL) with that token as a Bearer header.</li>
					<li>Open <strong class="text-foreground">Traces</strong> in the sidebar — your spans appear within seconds.</li>
				</ol>

				<p class="text-foreground font-medium mt-4">OpenTelemetry Collector example</p>
				<div class="bg-card-elevated border border-border/50 rounded-md p-3 font-mono text-xs overflow-x-auto">
<pre class="text-muted-foreground">exporters:
  otlp_http:
    endpoint: https://usewatchdog.dev
    headers:
      Authorization: "Bearer wd_..."

service:
  pipelines:
    traces: &#123; exporters: [otlp_http] &#125;
    logs:   &#123; exporters: [otlp_http] &#125;</pre>
				</div>

				<p>Endpoints accept <code class="text-foreground font-mono text-xs">Content-Encoding: gzip</code>. The 1 MB request cap and 64 KB per-record cap apply on the decompressed body. Your trace data is private to your account — never visible to other users on the same hub.</p>

				<p class="text-foreground font-medium mt-4">What the trace explorer gives you</p>
				<ul class="list-disc list-inside space-y-1 ml-1">
					<li><strong class="text-foreground">Trace list</strong>: filter by service, time range (1h / 6h / 24h), or errors-only. Auto-refresh every 15s / 30s / 60s.</li>
					<li><strong class="text-foreground">Waterfall</strong>: span timing, critical path highlight, hover crosshair, span attributes panel.</li>
					<li><strong class="text-foreground">Correlated logs</strong>: log records sharing the same <code class="text-foreground font-mono text-xs">trace_id</code> appear under the waterfall — no separate query.</li>
					<li><strong class="text-foreground">Pagination</strong>: a <strong class="text-foreground">Load older traces</strong> button at the bottom keeps loading 200-row pages until you reach the end of the visible time window.</li>
				</ul>
			</div>
		</section>

		<!-- Security -->
		<section class="py-10 border-b border-border/50">
			<h2 class="text-lg font-semibold text-foreground mb-4">Security</h2>
			<div class="space-y-4 text-sm text-muted-foreground leading-relaxed">
				<p>WatchDog ships with these security primitives built in:</p>
				<div class="grid sm:grid-cols-2 gap-3">
					{#each [
						['Audit trail', 'Every login, monitor change, and incident action is logged with user ID, IP, and timestamp'],
						['Account isolation', 'Each user&apos;s monitors, agents, and incidents are scoped to their own account'],
						['Brute-force protection', 'Per-IP and per-email login rate limiting with lockout'],
						['Encrypted connections', 'All agent-to-hub and browser-to-dashboard traffic uses TLS'],
						['Session management', 'Sessions invalidate on password change'],
						['API tokens', 'Scoped tokens for automation (admin, read-only, or telemetry-ingest)']
					] as [title, desc]}
						<div class="bg-card border border-border/50 rounded-md p-3">
							<p class="text-xs font-medium text-foreground">{title}</p>
							<p class="text-xs text-muted-foreground/70 mt-0.5">{@html desc}</p>
						</div>
					{/each}
				</div>
			</div>
		</section>

		<!-- CTA -->
		<section class="py-12 text-center">
			<h2 class="text-lg font-bold text-foreground mb-2">Ready to start monitoring?</h2>
			<p class="text-sm text-muted-foreground mb-5">WatchDog is open source (AGPL-3.0). Self-host with one Docker compose up, or sign up at usewatchdog.dev.</p>
			<div class="flex flex-col sm:flex-row items-center justify-center gap-3">
				<a href="/register" class="rounded-md bg-accent px-6 py-2.5 text-sm font-medium text-white hover:bg-accent/90 transition-colors">
					Get started
				</a>
				<a href="/docs" class="rounded-md border border-border px-6 py-2.5 text-sm font-medium text-muted-foreground hover:text-foreground hover:border-muted-foreground/30 transition-colors">
					API Reference
				</a>
			</div>
		</section>
	</main>
</div>
