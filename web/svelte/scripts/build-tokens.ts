import { writeFileSync, mkdirSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { tokensToCSS } from '../src/lib/tokens';

const HERE = dirname(fileURLToPath(import.meta.url));
const OUT = resolve(HERE, '../src/styles/theme-watchdog.css');

mkdirSync(dirname(OUT), { recursive: true });
writeFileSync(OUT, tokensToCSS(), 'utf-8');

console.log(`wrote ${OUT}`);
