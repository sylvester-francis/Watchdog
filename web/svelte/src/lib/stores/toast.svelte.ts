type ToastType = 'success' | 'error' | 'warning';

interface Toast {
	id: number;
	type: ToastType;
	message: string;
}

let toasts = $state<Toast[]>([]);
let nextId = 0;

export function getToasts() {
	function add(type: ToastType, message: string, duration = 4000) {
		const id = nextId++;
		toasts = [...toasts, { id, type, message }];
		if (duration > 0) {
			setTimeout(() => remove(id), duration);
		}
	}

	function remove(id: number) {
		toasts = toasts.filter((t) => t.id !== id);
	}

	function success(message: string) { add('success', message); }
	function error(message: string) { add('error', message); }
	function warning(message: string) { add('warning', message); }

	return {
		get items() { return toasts; },
		add,
		remove,
		success,
		error,
		warning
	};
}
