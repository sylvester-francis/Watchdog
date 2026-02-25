export function formatTimeAgo(date: string): string {
	const now = Date.now();
	const then = new Date(date).getTime();
	const seconds = Math.floor((now - then) / 1000);

	if (seconds < 60) return 'just now';
	const minutes = Math.floor(seconds / 60);
	if (minutes < 60) return `${minutes}m ago`;
	const hours = Math.floor(minutes / 60);
	if (hours < 24) return `${hours}h ago`;
	const days = Math.floor(hours / 24);
	return `${days}d ago`;
}

export function formatDuration(startedAt: string): string {
	const now = Date.now();
	const start = new Date(startedAt).getTime();
	const seconds = Math.floor((now - start) / 1000);

	if (seconds < 60) return `${seconds}s`;
	const minutes = Math.floor(seconds / 60);
	const hours = Math.floor(minutes / 60);
	if (hours === 0) return `${minutes}m`;
	return `${hours}h ${minutes % 60}m`;
}

export function formatPercent(value: number): string {
	return value.toFixed(2);
}

export function uptimeColor(percent: number): string {
	if (percent >= 99) return 'text-emerald-400';
	if (percent >= 95) return 'text-amber-400';
	return 'text-red-400';
}

export function isInfraMonitor(type: string): boolean {
	return type === 'docker' || type === 'database' || type === 'system' || type === 'service';
}

export function isNonLatencyMonitor(type: string): boolean {
	return type === 'system' || type === 'docker' || type === 'service';
}
