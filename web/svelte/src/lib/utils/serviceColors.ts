// Deterministic service-color assignment. FNV-1a hash of service_name
// modulo 8 picks a slot from a curated muted-tone palette so the same
// service is the same color across reloads, sessions, and views.
//
// Tokens are referenced by index (1..8) and resolved to Tailwind classes
// at consumption time. Palette is muted on purpose — bright colors fight
// the status colors that actually carry meaning (red = error, etc.).

const PALETTE_SIZE = 8;

// Hash → palette slot. fnv1a on UTF-16 code units is plenty for our use
// (we want stability across reloads, not cryptographic strength).
function fnv1a(s: string): number {
	let h = 0x811c9dc5;
	for (let i = 0; i < s.length; i++) {
		h ^= s.charCodeAt(i);
		h = Math.imul(h, 0x01000193);
	}
	return h >>> 0;
}

export function serviceColorIndex(serviceName: string): number {
	if (!serviceName) return 0;
	return fnv1a(serviceName) % PALETTE_SIZE;
}

// Bar/stripe color (the colored band on the waterfall row). 40 alpha
// keeps the bar muted enough that durations and errors still draw the
// eye first.
const BAR_CLASSES = [
	'bg-slate-400/40',
	'bg-teal-400/40',
	'bg-amber-400/40',
	'bg-rose-400/40',
	'bg-lime-400/40',
	'bg-sky-400/40',
	'bg-violet-400/40',
	'bg-orange-400/40'
];

// Service chip background — slightly more opaque so the service name is
// readable in the right rail header.
const CHIP_CLASSES = [
	'bg-slate-400/15 text-slate-300',
	'bg-teal-400/15 text-teal-300',
	'bg-amber-400/15 text-amber-300',
	'bg-rose-400/15 text-rose-300',
	'bg-lime-400/15 text-lime-300',
	'bg-sky-400/15 text-sky-300',
	'bg-violet-400/15 text-violet-300',
	'bg-orange-400/15 text-orange-300'
];

export function serviceBarClass(serviceName: string): string {
	return BAR_CLASSES[serviceColorIndex(serviceName)];
}

export function serviceChipClass(serviceName: string): string {
	return CHIP_CLASSES[serviceColorIndex(serviceName)];
}
