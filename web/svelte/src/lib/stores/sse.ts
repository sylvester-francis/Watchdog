type SSECallback = (event: string, data: unknown) => void;

export function createSSE(onEvent: SSECallback) {
	let eventSource: EventSource | null = null;
	let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
	let retryDelay = 1000;
	const maxDelay = 30000;

	function connect() {
		if (eventSource) return;

		eventSource = new EventSource('/sse/events', { withCredentials: true });

		eventSource.onopen = () => {
			retryDelay = 1000;
		};

		eventSource.addEventListener('agent-status', (e) => {
			try {
				onEvent('agent-status', JSON.parse(e.data));
			} catch { /* ignore parse errors */ }
		});

		eventSource.addEventListener('incident-count', (e) => {
			try {
				onEvent('incident-count', JSON.parse(e.data));
			} catch { /* ignore parse errors */ }
		});

		eventSource.onerror = () => {
			cleanup();
			reconnectTimer = setTimeout(() => {
				retryDelay = Math.min(retryDelay * 2, maxDelay);
				connect();
			}, retryDelay);
		};
	}

	function cleanup() {
		if (eventSource) {
			eventSource.close();
			eventSource = null;
		}
	}

	function disconnect() {
		cleanup();
		if (reconnectTimer) {
			clearTimeout(reconnectTimer);
			reconnectTimer = null;
		}
	}

	return { connect, disconnect };
}
