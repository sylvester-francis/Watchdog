<script lang="ts">
	import { onMount, onDestroy, tick } from 'svelte';

	import { Database, Layers, Server, Clock, HeartPulse, HardDrive, ArrowUpCircle, ScrollText, Users, KeyRound, Copy, Check, AlertTriangle, Trash2, Activity, Wifi, CircleDot } from 'lucide-svelte';
	import { system as systemApi } from '$lib/api';
	import { getAuth } from '$lib/stores/auth.svelte';
	import { getToasts } from '$lib/stores/toast.svelte';
	import type { SystemInfo, AdminUser, MetricsResponse, MetricsSnapshot } from '$lib/types';
	import ConfirmModal from '$lib/components/ConfirmModal.svelte';
	import {
		Chart,
		LineController,
		LineElement,
		PointElement,
		LinearScale,
		CategoryScale,
		Filler,
		Tooltip,
		Legend
	} from 'chart.js';

	Chart.register(LineController, LineElement, PointElement, LinearScale, CategoryScale, Filler, Tooltip, Legend);

	const auth = getAuth();
	const toast = getToasts();
	const isAdmin = $derived(auth.user?.is_admin === true);

	let data = $state<SystemInfo | null>(null);
	let users = $state<AdminUser[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Hub Metrics state
	let metrics = $state<MetricsResponse | null>(null);
	let metricsInterval: ReturnType<typeof setInterval> | null = null;
	let httpLatencyCanvas = $state<HTMLCanvasElement>(undefined as unknown as HTMLCanvasElement);
	let heartbeatCanvas = $state<HTMLCanvasElement>(undefined as unknown as HTMLCanvasElement);
	let requestRateCanvas = $state<HTMLCanvasElement>(undefined as unknown as HTMLCanvasElement);
	let httpLatencyChart: Chart | null = null;
	let heartbeatChart: Chart | null = null;
	let requestRateChart: Chart | null = null;

	const chartTooltip = {
		backgroundColor: '#18181b',
		borderColor: '#27272a',
		borderWidth: 1,
		titleFont: { family: "'Inter', system-ui, sans-serif" as const, size: 11 },
		bodyFont: { family: "'JetBrains Mono', ui-monospace, monospace" as const, size: 12 },
		padding: 10,
		cornerRadius: 6,
		displayColors: true,
		boxWidth: 8,
		boxPadding: 4
	};

	const chartScaleDefaults = {
		x: {
			grid: { color: '#27272a', lineWidth: 0.5 },
			ticks: { maxTicksLimit: 8, maxRotation: 0, color: '#71717a', font: { size: 10, family: "'JetBrains Mono', ui-monospace, monospace" as const } },
			border: { color: '#27272a' }
		},
		y: {
			grid: { color: '#27272a', lineWidth: 0.5 },
			ticks: { color: '#71717a', font: { size: 10, family: "'JetBrains Mono', ui-monospace, monospace" as const } },
			beginAtZero: true,
			border: { color: '#27272a' }
		}
	};

	function formatChartTime(ts: number): string {
		const d = new Date(ts * 1000);
		return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}

	function buildMetricsCharts() {
		const history = metrics?.history;
		if (!history || history.length < 2) return;

		const labels = history.map(s => formatChartTime(s.timestamp));

		if (httpLatencyCanvas) {
			if (httpLatencyChart) httpLatencyChart.destroy();
			httpLatencyChart = new Chart(httpLatencyCanvas, {
				type: 'line',
				data: {
					labels,
					datasets: [
						{ label: 'p50', data: history.map(s => s.http_latency_p50), borderColor: '#3b82f6', borderWidth: 2, pointRadius: 0, pointHoverRadius: 3, tension: 0.3, fill: false },
						{ label: 'p95', data: history.map(s => s.http_latency_p95), borderColor: '#fbbf24', borderWidth: 1.5, pointRadius: 0, pointHoverRadius: 3, tension: 0.3, fill: false },
						{ label: 'p99', data: history.map(s => s.http_latency_p99), borderColor: '#f87171', borderWidth: 1, borderDash: [3, 3], pointRadius: 0, pointHoverRadius: 3, tension: 0.3, fill: false }
					]
				},
				options: {
					responsive: true, maintainAspectRatio: false,
					interaction: { mode: 'index', intersect: false },
					plugins: {
						legend: { display: true, position: 'top', align: 'end', labels: { boxWidth: 12, padding: 10, font: { size: 10 }, color: '#a1a1aa', usePointStyle: true, pointStyle: 'line' } },
						tooltip: { ...chartTooltip, callbacks: { label: (ctx: any) => ` ${ctx.dataset.label}: ${ctx.parsed.y}ms` } }
					},
					scales: { ...chartScaleDefaults, y: { ...chartScaleDefaults.y, ticks: { ...chartScaleDefaults.y.ticks, callback: (v: any) => `${v}ms` } } },
					animation: { duration: 0 }
				}
			});
		}

		if (heartbeatCanvas) {
			if (heartbeatChart) heartbeatChart.destroy();
			heartbeatChart = new Chart(heartbeatCanvas, {
				type: 'line',
				data: {
					labels,
					datasets: [
						{ label: 'p50', data: history.map(s => s.heartbeat_p50), borderColor: '#3b82f6', borderWidth: 2, pointRadius: 0, pointHoverRadius: 3, tension: 0.3, fill: false },
						{ label: 'p95', data: history.map(s => s.heartbeat_p95), borderColor: '#fbbf24', borderWidth: 1.5, pointRadius: 0, pointHoverRadius: 3, tension: 0.3, fill: false }
					]
				},
				options: {
					responsive: true, maintainAspectRatio: false,
					interaction: { mode: 'index', intersect: false },
					plugins: {
						legend: { display: true, position: 'top', align: 'end', labels: { boxWidth: 12, padding: 10, font: { size: 10 }, color: '#a1a1aa', usePointStyle: true, pointStyle: 'line' } },
						tooltip: { ...chartTooltip, callbacks: { label: (ctx: any) => ` ${ctx.dataset.label}: ${ctx.parsed.y}ms` } }
					},
					scales: { ...chartScaleDefaults, y: { ...chartScaleDefaults.y, ticks: { ...chartScaleDefaults.y.ticks, callback: (v: any) => `${v}ms` } } },
					animation: { duration: 0 }
				}
			});
		}

		if (requestRateCanvas) {
			if (requestRateChart) requestRateChart.destroy();
			requestRateChart = new Chart(requestRateCanvas, {
				type: 'line',
				data: {
					labels,
					datasets: [
						{ label: 'req/s', data: history.map(s => s.http_request_rate), borderColor: '#3b82f6', backgroundColor: 'rgba(59, 130, 246, 0.08)', borderWidth: 2, pointRadius: 0, pointHoverRadius: 3, tension: 0.3, fill: true }
					]
				},
				options: {
					responsive: true, maintainAspectRatio: false,
					interaction: { mode: 'index', intersect: false },
					plugins: {
						legend: { display: false },
						tooltip: { ...chartTooltip, callbacks: { label: (ctx: any) => ` ${ctx.parsed.y} req/s` } }
					},
					scales: { ...chartScaleDefaults, y: { ...chartScaleDefaults.y, ticks: { ...chartScaleDefaults.y.ticks, callback: (v: any) => `${v}/s` } } },
					animation: { duration: 0 }
				}
			});
		}
	}

	async function refreshMetrics() {
		try {
			metrics = await systemApi.getMetrics();
			buildMetricsCharts();
		} catch {
			// Silently fail — metrics are supplemental
		}
	}

	function destroyCharts() {
		httpLatencyChart?.destroy(); httpLatencyChart = null;
		heartbeatChart?.destroy(); heartbeatChart = null;
		requestRateChart?.destroy(); requestRateChart = null;
	}

	// Password reset state
	let resetPassword = $state('');
	let resetUserEmail = $state('');
	let copiedPassword = $state(false);
	let confirmModal = $state<{
		open: boolean;
		title: string;
		message: string;
		confirmLabel: string;
		variant: 'danger' | 'warning';
		loading: boolean;
		action: (() => Promise<void>) | null;
	}>({
		open: false, title: '', message: '', confirmLabel: 'Confirm', variant: 'warning', loading: false, action: null
	});

	function closeConfirmModal() {
		confirmModal = { open: false, title: '', message: '', confirmLabel: 'Confirm', variant: 'warning', loading: false, action: null };
	}

	async function executeConfirm() {
		if (confirmModal.action) await confirmModal.action();
	}

	function handleResetPassword(user: AdminUser) {
		confirmModal = {
			open: true,
			title: 'Reset Password',
			message: `Reset the password for ${user.email}? They will be required to change it on next login.`,
			confirmLabel: 'Reset Password',
			variant: 'warning',
			loading: false,
			action: async () => {
				confirmModal.loading = true;
				try {
					const res = await systemApi.resetUserPassword(user.id);
					resetPassword = res.password;
					resetUserEmail = user.email;
					closeConfirmModal();
				} catch (err) {
					toast.error(err instanceof Error ? err.message : 'Failed to reset password.');
					confirmModal.loading = false;
				}
			}
		};
	}

	function handleDeleteUser(user: AdminUser) {
		confirmModal = {
			open: true,
			title: 'Delete User',
			message: `Permanently delete ${user.email}? This will remove their agents, monitors, and all associated data. This cannot be undone.`,
			confirmLabel: 'Delete User',
			variant: 'danger',
			loading: false,
			action: async () => {
				confirmModal.loading = true;
				try {
					await systemApi.deleteUser(user.id);
					users = users.filter(u => u.id !== user.id);
					toast.success(`User ${user.email} deleted.`);
					closeConfirmModal();
				} catch (err) {
					toast.error(err instanceof Error ? err.message : 'Failed to delete user.');
					confirmModal.loading = false;
				}
			}
		};
	}

	async function copyPasswordToClipboard() {
		await navigator.clipboard.writeText(resetPassword);
		copiedPassword = true;
		setTimeout(() => { copiedPassword = false; }, 2000);
	}

	function dismissResetPassword() {
		resetPassword = '';
		resetUserEmail = '';
		copiedPassword = false;
	}

	function timeAgo(dateStr: string): string {
		const diff = Date.now() - new Date(dateStr).getTime();
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return 'just now';
		if (mins < 60) return `${mins}m ago`;
		const hours = Math.floor(mins / 60);
		if (hours < 24) return `${hours}h ago`;
		const days = Math.floor(hours / 24);
		return `${days}d ago`;
	}

	function actionBadgeClass(action: string): string {
		if (action === 'login_success') return 'bg-green-500/15 text-green-400';
		if (action === 'login_failed') return 'bg-red-500/15 text-red-400';
		if (action.endsWith('_created')) return 'bg-blue-500/15 text-blue-400';
		if (action.endsWith('_updated')) return 'bg-yellow-500/15 text-yellow-400';
		if (action.endsWith('_deleted') || action.endsWith('_revoked')) return 'bg-red-500/15 text-red-400';
		if (action.startsWith('incident_')) return 'bg-orange-500/15 text-orange-400';
		if (action === 'settings_changed') return 'bg-purple-500/15 text-purple-400';
		return 'bg-muted text-muted-foreground';
	}

	onMount(async () => {
		try {
			const [sysResult, usersResult, metricsResult] = await Promise.allSettled([
				systemApi.getSystemInfo(),
				systemApi.listUsers(),
				systemApi.getMetrics()
			]);

			if (sysResult.status === 'fulfilled') {
				data = sysResult.value;
			} else {
				throw sysResult.reason;
			}

			if (usersResult.status === 'fulfilled') {
				users = usersResult.value.data ?? [];
			}

			if (metricsResult.status === 'fulfilled') {
				metrics = metricsResult.value;
			}

			metricsInterval = setInterval(refreshMetrics, 10000);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load system info';
		} finally {
			loading = false;
			await tick();
			buildMetricsCharts();
		}
	});

	onDestroy(() => {
		if (metricsInterval) clearInterval(metricsInterval);
		destroyCharts();
	});
</script>

<svelte:head>
	<title>System - WatchDog</title>
</svelte:head>

{#if loading}
	<div class="animate-fade-in-up space-y-4">
		<div class="h-7 w-24 bg-muted/50 rounded animate-pulse"></div>
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
			{#each Array(4) as _}
				<div class="bg-card border border-border rounded-lg h-24 animate-pulse"></div>
			{/each}
		</div>
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
			{#each Array(2) as _}
				<div class="bg-card border border-border rounded-lg h-48 animate-pulse"></div>
			{/each}
		</div>
		<div class="bg-card border border-border rounded-lg h-64 animate-pulse"></div>
	</div>
{:else if error}
	<div class="animate-fade-in-up">
		<div class="bg-card border border-border rounded-lg p-8 text-center">
			<p class="text-sm text-foreground font-medium mb-1">Failed to load system info</p>
			<p class="text-xs text-muted-foreground">{error}</p>
		</div>
	</div>
{:else if data}
	<div class="animate-fade-in-up">
		<div class="mb-5">
			<h1 class="text-lg font-semibold text-foreground">System</h1>
			<p class="text-xs text-muted-foreground mt-0.5">Server health, performance metrics, and audit log.</p>
		</div>

		<!-- System Health Cards -->
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 mb-6">
			<!-- Database -->
			<div class="bg-card rounded-lg border border-border p-3">
				<div class="flex items-center justify-between mb-2">
					<p class="text-xs font-medium text-muted-foreground">Database</p>
					<div class="w-6 h-6 rounded flex items-center justify-center {data.db.healthy ? 'bg-emerald-500/10' : 'bg-red-500/10'}">
						<Database class="w-3 h-3 {data.db.healthy ? 'text-emerald-400' : 'text-red-400'}" />
					</div>
				</div>
				<p class="text-2xl font-semibold text-foreground font-mono tracking-tight">{data.db.ping_ms.toFixed(0)}ms</p>
				<p class="text-xs text-muted-foreground mt-1">{data.db.healthy ? 'Healthy' : 'Unreachable'} &middot; response time to database</p>
			</div>

			<!-- Connection Pool -->
			<div class="bg-card rounded-lg border border-border p-3">
				<div class="flex items-center justify-between mb-2">
					<p class="text-xs font-medium text-muted-foreground">DB Connections</p>
					<div class="w-6 h-6 bg-muted/50 rounded flex items-center justify-center">
						<Layers class="w-3 h-3 text-muted-foreground" />
					</div>
				</div>
				<p class="text-2xl font-semibold text-foreground font-mono tracking-tight">
					{data.db.pool.acquired} <span class="text-sm font-normal text-muted-foreground">/ {data.db.pool.total}</span>
				</p>
				<p class="text-xs text-muted-foreground mt-1">{data.db.pool.idle} available &middot; {data.db.pool.acquired} in use right now</p>
			</div>

			<!-- Connected Agents -->
			<div class="bg-card rounded-lg border border-border p-3">
				<div class="flex items-center justify-between mb-2">
					<p class="text-xs font-medium text-muted-foreground">Agents Online</p>
					<div class="w-6 h-6 {data.agents_connected > 0 ? 'bg-emerald-500/10' : 'bg-muted/50'} rounded flex items-center justify-center">
						<Server class="w-3 h-3 {data.agents_connected > 0 ? 'text-emerald-400' : 'text-muted-foreground'}" />
					</div>
				</div>
				<p class="text-2xl font-semibold text-foreground font-mono tracking-tight">{data.agents_connected}</p>
				<p class="text-xs text-muted-foreground mt-1">{data.agents_connected > 0 ? 'Reporting health checks' : 'No agents connected'}</p>
			</div>

			<!-- Uptime -->
			<div class="bg-card rounded-lg border border-border p-3">
				<div class="flex items-center justify-between mb-2">
					<p class="text-xs font-medium text-muted-foreground">Uptime</p>
					<div class="w-6 h-6 bg-muted/50 rounded flex items-center justify-center">
						<Clock class="w-3 h-3 text-muted-foreground" />
					</div>
				</div>
				<p class="text-2xl font-semibold text-foreground font-mono tracking-tight">{data.runtime.uptime_formatted}</p>
				<p class="text-xs text-muted-foreground mt-1">Time since server was started</p>
			</div>
		</div>

		<!-- Hub Metrics (Prometheus) -->
		<div class="mb-6">
			<div class="flex items-center space-x-2 mb-1">
				<div class="w-6 h-6 bg-muted/50 rounded flex items-center justify-center">
					<Activity class="w-3 h-3 text-muted-foreground" />
				</div>
				<h2 class="text-sm font-medium text-foreground">Hub Metrics</h2>
				<span class="text-[10px] text-muted-foreground">Auto-refreshes every 10s</span>
			</div>
			<p class="text-[10px] text-muted-foreground mb-3 ml-8">Live performance data from the hub server. Use these to spot bottlenecks before they become outages.</p>

			{#if metrics}
				{@const cur = metrics.current}
				<!-- Live Gauges -->
				<div class="grid grid-cols-3 gap-3 mb-4">
					<div class="bg-card border border-border rounded-lg px-4 py-3">
						<div class="flex items-center space-x-2 mb-1">
							<Wifi class="w-3 h-3 text-muted-foreground" />
							<span class="text-[10px] text-muted-foreground uppercase tracking-wider">WS Connections</span>
						</div>
						<span class="text-xl font-semibold font-mono text-foreground">{cur.ws_connections}</span>
						<p class="text-[9px] text-muted-foreground/60 mt-0.5">Agents connected right now</p>
					</div>
					<div class="bg-card border border-border rounded-lg px-4 py-3">
						<div class="flex items-center space-x-2 mb-1">
							<Database class="w-3 h-3 text-muted-foreground" />
							<span class="text-[10px] text-muted-foreground uppercase tracking-wider">DB Pool Active</span>
						</div>
						<span class="text-xl font-semibold font-mono text-foreground">{cur.db_pool_active}</span>
						<p class="text-[9px] text-muted-foreground/60 mt-0.5">Database connections in use</p>
					</div>
					<div class="bg-card border border-border rounded-lg px-4 py-3">
						<div class="flex items-center space-x-2 mb-1">
							<CircleDot class="w-3 h-3 text-muted-foreground" />
							<span class="text-[10px] text-muted-foreground uppercase tracking-wider">Active Incidents</span>
						</div>
						<div class="flex items-baseline space-x-2">
							<span class="text-xl font-semibold font-mono {cur.incidents_open > 0 ? 'text-red-400' : 'text-foreground'}">{cur.incidents_open}</span>
							{#if cur.incidents_acked > 0}
								<span class="text-xs text-muted-foreground font-mono">+{cur.incidents_acked} ack'd</span>
							{/if}
						</div>
						<p class="text-[9px] text-muted-foreground/60 mt-0.5">Unresolved problems</p>
					</div>
				</div>

				<!-- Time-series Charts -->
				{#if metrics.history && metrics.history.length >= 2}
					<div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
						<div class="bg-card border border-border rounded-lg">
							<div class="px-4 py-2.5 border-b border-border">
								<span class="text-xs font-medium text-foreground">HTTP Latency</span>
								<p class="text-[10px] text-muted-foreground mt-0.5">How fast the hub responds to API requests. Lower is better.</p>
							</div>
							<div class="px-4 py-3 h-[200px]">
								<canvas bind:this={httpLatencyCanvas}></canvas>
							</div>
						</div>
						<div class="bg-card border border-border rounded-lg">
							<div class="px-4 py-2.5 border-b border-border">
								<span class="text-xs font-medium text-foreground">Heartbeat Processing</span>
								<p class="text-[10px] text-muted-foreground mt-0.5">Time to process each health check from agents. Spikes mean the hub is overloaded.</p>
							</div>
							<div class="px-4 py-3 h-[200px]">
								<canvas bind:this={heartbeatCanvas}></canvas>
							</div>
						</div>
						<div class="bg-card border border-border rounded-lg lg:col-span-2">
							<div class="px-4 py-2.5 border-b border-border">
								<span class="text-xs font-medium text-foreground">Request Rate</span>
								<p class="text-[10px] text-muted-foreground mt-0.5">Total API requests per second hitting the hub. Helps size capacity and spot traffic anomalies.</p>
							</div>
							<div class="px-4 py-3 h-[180px]">
								<canvas bind:this={requestRateCanvas}></canvas>
							</div>
						</div>
					</div>
				{:else}
					<div class="bg-card border border-border rounded-lg p-6 text-center">
						<p class="text-xs text-muted-foreground">Collecting data... charts will appear after ~20s of history.</p>
					</div>
				{/if}
			{/if}
		</div>

		<!-- Operational Metrics -->
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6">
			<!-- Heartbeat Throughput -->
			<div class="bg-card rounded-lg border border-border">
				<div class="px-4 py-3 border-b border-border flex items-center space-x-2">
					<div class="w-6 h-6 bg-muted/50 rounded flex items-center justify-center">
						<HeartPulse class="w-3 h-3 text-muted-foreground" />
					</div>
					<h2 class="text-sm font-medium text-foreground">Heartbeat Throughput</h2>
				</div>
				<div class="px-4 py-3">
					<div class="flex items-center justify-between py-2">
						<span class="text-xs text-muted-foreground">Checks per minute</span>
						<span class="text-sm font-medium text-foreground font-mono">{data.heartbeats.per_minute.toFixed(1)}/min</span>
					</div>
					<div class="border-t border-border/30"></div>
					<div class="flex items-center justify-between py-2">
						<span class="text-xs text-muted-foreground">Total checks in last hour</span>
						<span class="text-sm font-medium text-foreground font-mono">{data.heartbeats.total_last_hour}</span>
					</div>
					<div class="border-t border-border/30"></div>
					<div class="flex items-center justify-between py-2">
						<span class="text-xs text-muted-foreground">Failed checks in last hour</span>
						<span class="text-sm font-medium font-mono {data.heartbeats.errors_last_hour > 0 ? 'text-red-400' : 'text-green-400'}">
							{data.heartbeats.errors_last_hour}
						</span>
					</div>
				</div>
			</div>

			<!-- Storage & Runtime -->
			<div class="bg-card rounded-lg border border-border">
				<div class="px-4 py-3 border-b border-border flex items-center space-x-2">
					<div class="w-6 h-6 bg-muted/50 rounded flex items-center justify-center">
						<HardDrive class="w-3 h-3 text-muted-foreground" />
					</div>
					<h2 class="text-sm font-medium text-foreground">Storage & Runtime</h2>
				</div>
				<div class="px-4 py-3">
					<div class="flex items-center justify-between py-2">
						<span class="text-xs text-muted-foreground">Total disk used by database</span>
						<span class="text-sm font-medium text-foreground font-mono">{data.db.size}</span>
					</div>
					<div class="border-t border-border/30"></div>
					<div class="flex items-center justify-between py-2">
						<span class="text-xs text-muted-foreground">Active background tasks</span>
						<span class="text-sm font-medium text-foreground font-mono">{data.runtime.goroutines}</span>
					</div>
					<div class="border-t border-border/30"></div>
					<div class="flex items-center justify-between py-2">
						<span class="text-xs text-muted-foreground">Memory in use</span>
						<span class="text-sm font-medium text-foreground font-mono">{data.runtime.heap_mb} MB</span>
					</div>
					<div class="border-t border-border/30"></div>
					<div class="flex items-center justify-between py-2">
						<span class="text-xs text-muted-foreground">Last garbage collection pause</span>
						<span class="text-sm font-medium text-foreground font-mono">{data.runtime.gc_pause_ms} ms</span>
					</div>
				</div>
			</div>
		</div>

		<!-- Table Sizes -->
		{#if data.db.table_sizes.length > 0}
			<div class="bg-card rounded-lg border border-border mb-6">
				<div class="px-4 py-3 border-b border-border flex items-center justify-between">
					<div class="flex items-center space-x-2">
						<div class="w-6 h-6 bg-muted/50 rounded flex items-center justify-center">
							<Database class="w-3 h-3 text-muted-foreground" />
						</div>
						<h2 class="text-sm font-medium text-foreground">Table Sizes</h2>
					</div>
					<span class="text-[10px] text-muted-foreground font-mono">Largest 5 tables by disk space</span>
				</div>
				<div class="px-4 py-2">
					{#each data.db.table_sizes as table, i}
						{#if i > 0}
							<div class="border-t border-border/30"></div>
						{/if}
						<div class="flex items-center justify-between py-2">
							<span class="text-xs text-muted-foreground font-mono">{table.name}</span>
							<span class="text-xs font-medium text-foreground font-mono">{table.size}</span>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Migration Status -->
		<div class="bg-card rounded-lg border border-border mb-6">
			<div class="px-4 py-3 border-b border-border flex items-center space-x-2">
				<div class="w-6 h-6 bg-muted/50 rounded flex items-center justify-center">
					<ArrowUpCircle class="w-3 h-3 text-muted-foreground" />
				</div>
				<h2 class="text-sm font-medium text-foreground">Migration Status</h2>
			</div>
			<div class="px-4 py-3 flex items-center space-x-4">
				<div class="flex items-center space-x-2">
					<span class="text-xs text-muted-foreground">Schema version</span>
					<span class="text-sm font-medium text-foreground font-mono">{data.db.migration.version}</span>
				</div>
				<div class="w-px h-4 bg-border/50"></div>
				<div class="flex items-center space-x-2">
					<span class="text-xs text-muted-foreground">Failed migration</span>
					{#if data.db.migration.dirty}
						<span class="px-2 py-0.5 text-[10px] font-medium rounded bg-red-500/15 text-red-400">Yes</span>
					{:else}
						<span class="px-2 py-0.5 text-[10px] font-medium rounded bg-green-500/15 text-green-400">No</span>
					{/if}
				</div>
			</div>
		</div>

		{#if isAdmin}
		<!-- Reset Password Banner (shown after admin reset) -->
		{#if resetPassword}
			<div class="bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-4 mb-6">
				<div class="flex items-start justify-between mb-2">
					<div class="flex items-center space-x-2">
						<KeyRound class="w-4 h-4 text-yellow-400" />
						<span class="text-sm font-medium text-foreground">Password Reset</span>
					</div>
					<button
						onclick={dismissResetPassword}
						class="text-muted-foreground hover:text-foreground transition-colors text-xs"
					>
						Dismiss
					</button>
				</div>
				<p class="text-xs text-muted-foreground mb-1">Temporary password for <span class="text-foreground font-medium">{resetUserEmail}</span>:</p>
				<p class="text-xs text-muted-foreground mb-2">Copy this password now. You won't be able to see it again. The user will be required to change it on next login.</p>
				<div class="flex items-center space-x-2">
					<code class="flex-1 text-xs font-mono bg-card border border-border rounded px-3 py-2 text-foreground break-all select-all">{resetPassword}</code>
					<button
						onclick={copyPasswordToClipboard}
						class="p-2 rounded-md hover:bg-muted/50 text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
						aria-label="Copy password"
					>
						{#if copiedPassword}
							<Check class="w-4 h-4 text-emerald-400" />
						{:else}
							<Copy class="w-4 h-4" />
						{/if}
					</button>
				</div>
			</div>
		{/if}

		<!-- Users -->
		<div class="bg-card rounded-lg border border-border mb-6">
			<div class="px-4 py-3 border-b border-border flex items-center justify-between">
				<div class="flex items-center space-x-2">
					<div class="w-6 h-6 bg-muted/50 rounded flex items-center justify-center">
						<Users class="w-3 h-3 text-muted-foreground" />
					</div>
					<h2 class="text-sm font-medium text-foreground">Users</h2>
					{#if users.length > 0}
						<span class="text-[10px] font-mono text-muted-foreground bg-muted/50 px-1.5 py-0.5 rounded">{users.length}</span>
					{/if}
				</div>
			</div>

			{#if users.length > 0}
				<div class="overflow-x-auto">
					<table class="w-full">
						<thead>
							<tr class="border-b border-border">
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Email</th>
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider hidden sm:table-cell">Username</th>
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider hidden md:table-cell">Plan</th>
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider hidden lg:table-cell">Agents</th>
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider hidden lg:table-cell">Monitors</th>
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider hidden md:table-cell">Joined</th>
								<th class="px-4 py-2.5 text-right text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Actions</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border/30">
							{#each users as u (u.id)}
								<tr class="hover:bg-muted/20 transition-colors">
									<td class="px-4 py-2.5 text-xs text-foreground">
										<div class="flex items-center space-x-2">
											<span>{u.email}</span>
											{#if u.is_admin}
												<span class="px-1.5 py-0.5 text-[9px] font-medium rounded bg-yellow-500/15 text-yellow-400 uppercase">Admin</span>
											{/if}
										</div>
									</td>
									<td class="px-4 py-2.5 text-xs text-muted-foreground hidden sm:table-cell font-mono">{u.username}</td>
									<td class="px-4 py-2.5 text-xs text-muted-foreground hidden md:table-cell capitalize">{u.plan}</td>
									<td class="px-4 py-2.5 text-xs text-muted-foreground font-mono hidden lg:table-cell">{u.agent_count}</td>
									<td class="px-4 py-2.5 text-xs text-muted-foreground font-mono hidden lg:table-cell">{u.monitor_count}</td>
									<td class="px-4 py-2.5 text-xs text-muted-foreground hidden md:table-cell">{timeAgo(u.created_at)}</td>
									<td class="px-4 py-2.5 text-right">
										{#if u.id !== auth.user?.id}
											<div class="flex items-center justify-end space-x-1.5">
												<button
													onclick={() => handleResetPassword(u)}
													class="px-2.5 py-1.5 text-[10px] font-medium text-muted-foreground hover:text-foreground bg-muted/50 hover:bg-muted rounded-md transition-colors"
												>
													Reset Password
												</button>
												<button
													onclick={() => handleDeleteUser(u)}
													class="inline-flex items-center space-x-1 px-2.5 py-1.5 text-[10px] font-medium text-muted-foreground hover:text-destructive bg-muted/50 hover:bg-destructive/10 rounded-md transition-colors"
												>
													<Trash2 class="w-3 h-3" />
													<span>Delete</span>
												</button>
											</div>
										{:else}
											<span class="text-[10px] text-muted-foreground/40">You</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<div class="text-center py-12">
					<div class="w-12 h-12 rounded-full bg-muted/50 flex items-center justify-center mx-auto mb-3">
						<Users class="w-6 h-6 text-muted-foreground/40" />
					</div>
					<p class="text-sm text-muted-foreground font-medium">No users found</p>
				</div>
			{/if}
		</div>
		{/if}

		<!-- Audit Log -->
		<div class="bg-card rounded-lg border border-border">
			<div class="px-4 py-3 border-b border-border flex items-center justify-between">
				<div class="flex items-center space-x-2">
					<div class="w-6 h-6 bg-muted/50 rounded flex items-center justify-center">
						<ScrollText class="w-3 h-3 text-muted-foreground" />
					</div>
					<h2 class="text-sm font-medium text-foreground">Audit Log</h2>
				</div>
				<span class="text-[10px] text-muted-foreground font-mono">Last 50 events</span>
			</div>

			{#if data.audit_logs.length > 0}
				<div class="overflow-x-auto max-h-[32rem] overflow-y-auto">
					<table class="w-full">
						<thead class="sticky top-0 bg-card z-10">
							<tr class="border-b border-border">
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider w-24">Time</th>
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Action</th>
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">User</th>
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider hidden sm:table-cell w-32">IP Address</th>
								<th class="px-4 py-2.5 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider hidden md:table-cell">Details</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border/30">
							{#each data.audit_logs as log}
								<tr class="hover:bg-muted/20 transition-colors">
									<td class="px-4 py-2.5 text-xs text-muted-foreground whitespace-nowrap font-mono">{timeAgo(log.created_at)}</td>
									<td class="px-4 py-2.5">
										<span class="px-2 py-0.5 text-[10px] font-medium rounded whitespace-nowrap {actionBadgeClass(log.action)}">
											{log.action}
										</span>
									</td>
									<td class="px-4 py-2.5 text-xs text-foreground">
										{#if log.user_email}
											{log.user_email}
										{:else}
											<span class="text-muted-foreground/40">&mdash;</span>
										{/if}
									</td>
									<td class="px-4 py-2.5 text-xs text-muted-foreground font-mono hidden sm:table-cell">
										{#if log.ip_address}
											{log.ip_address}
										{:else}
											<span class="text-muted-foreground/40">&mdash;</span>
										{/if}
									</td>
									<td class="px-4 py-2.5 text-xs text-muted-foreground hidden md:table-cell max-w-xs truncate">
										{#if log.metadata && Object.keys(log.metadata).length > 0}
											{#each Object.entries(log.metadata) as [k, v]}
												<span class="inline-block mr-2"><span class="text-muted-foreground/50">{k}:</span> {v}</span>
											{/each}
										{:else}
											<span class="text-muted-foreground/40">&mdash;</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<div class="text-center py-12">
					<div class="w-12 h-12 rounded-full bg-muted/50 flex items-center justify-center mx-auto mb-3">
						<ScrollText class="w-6 h-6 text-muted-foreground/40" />
					</div>
					<p class="text-sm text-muted-foreground font-medium">No audit log entries yet</p>
					<p class="text-xs text-muted-foreground/60 mt-1">Events will appear here as users interact with the system.</p>
				</div>
			{/if}
		</div>
	</div>

	<ConfirmModal
		open={confirmModal.open}
		title={confirmModal.title}
		message={confirmModal.message}
		confirmLabel={confirmModal.confirmLabel}
		variant={confirmModal.variant}
		loading={confirmModal.loading}
		onConfirm={executeConfirm}
		onCancel={closeConfirmModal}
	/>
{/if}
