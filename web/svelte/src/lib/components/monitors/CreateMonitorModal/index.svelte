<script lang="ts">
	import { X, AlertCircle } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import type { Agent, MonitorType, DeviceTemplate } from '$lib/types';
	import { onMount } from 'svelte';
	import { Modal } from '@sylvester-francis/watchdog-ui';
	import { Button } from '@sylvester-francis/watchdog-ui';
	import SharedFields from './SharedFields.svelte';
	import HTTPSection from './HTTPSection.svelte';
	import DatabaseSection from './DatabaseSection.svelte';
	import SystemSection from './SystemSection.svelte';
	import PortScanSection from './PortScanSection.svelte';
	import SNMPSection from './SNMPSection.svelte';

	interface Props {
		open: boolean;
		agents: Agent[];
		onClose: () => void;
		onCreated: () => void;
	}

	let { open = $bindable(), agents, onClose, onCreated }: Props = $props();

	let name = $state('');
	let type = $state<MonitorType>('http');
	let target = $state('');
	let agentId = $state('');
	let intervalSeconds = $state(30);
	let timeoutSeconds = $state(10);
	let failureThreshold = $state(3);

	let expectedStatus = $state(200);

	let dbType = $state('postgres');
	let dbConnectionString = $state('');
	let dbPassword = $state('');

	let systemMetric = $state('cpu');
	let systemThreshold = $state(80);

	let portScanPorts = $state('');
	let portScanRange = $state('');
	let portScanExpectedOpen = $state('');
	let bannerGrab = $state(true);

	let snmpVersion = $state<'2c' | '3'>('2c');
	let snmpCommunity = $state('public');
	let snmpOid = $state('');
	let snmpOids = $state('');
	let snmpOperation = $state<'get' | 'walk' | 'bulk'>('get');
	let snmpPort = $state(161);
	let snmpSecurityLevel = $state<'noAuthNoPriv' | 'authNoPriv' | 'authPriv'>('authNoPriv');
	let snmpUsername = $state('');
	let snmpAuthProtocol = $state('SHA');
	let snmpAuthPassword = $state('');
	let snmpPrivacyProtocol = $state('AES');
	let snmpPrivacyPassword = $state('');

	let deviceTemplates = $state<DeviceTemplate[]>([]);
	let selectedTemplateId = $state('');

	onMount(async () => {
		try {
			const res = await monitorsApi.listDeviceTemplates();
			deviceTemplates = res.data ?? [];
		} catch {
			// silent
		}
	});

	async function applyTemplate(templateId: string) {
		if (!templateId) {
			selectedTemplateId = '';
			return;
		}
		try {
			const res = await monitorsApi.getDeviceTemplate(templateId);
			const t = res.data;
			if (!t?.oids) return;

			selectedTemplateId = templateId;

			const scalarOids = t.oids.filter(o => o.category === 'system' || o.category === 'cpu' || o.category === 'memory' || o.category === 'battery' || o.category === 'output' || o.category === 'input');
			const walkOids = t.oids.filter(o => o.category === 'interface' || o.category === 'storage');

			if (walkOids.length > 0 && scalarOids.length > 0) {
				snmpOperation = 'bulk';
				const allOids = t.oids.map(o => o.oid);
				const seen = new Set<string>();
				const deduped: string[] = [];
				for (const oid of allOids) {
					const parts = oid.split('.');
					if (parts.length > 8) {
						const prefix = parts.slice(0, 8).join('.');
						if (!seen.has(prefix)) {
							seen.add(prefix);
							deduped.push(oid);
						}
					} else {
						deduped.push(oid);
					}
				}
				snmpOids = deduped.join(',');
				snmpOid = '';
			} else if (scalarOids.length > 0) {
				snmpOperation = 'get';
				if (scalarOids.length === 1) {
					snmpOid = scalarOids[0].oid;
					snmpOids = '';
				} else {
					snmpOid = scalarOids[0].oid;
					snmpOids = scalarOids.slice(1).map(o => o.oid).join(',');
				}
			}

			const counterOids = t.oids.filter(o => o.is_counter).map(o => o.oid);

			if (!name || name === deviceTemplates.find(d => d.id === selectedTemplateId)?.model) {
				name = `${t.vendor} ${t.model}`;
			}
			intervalSeconds = t.default_interval;

			snmpTemplateId = templateId;
			snmpRateOids = counterOids.join(',');
		} catch {
			// silent
		}
	}

	let snmpTemplateId = $state('');
	let snmpRateOids = $state('');

	const snmpPresets: { label: string; group: string; oid: string; op: 'get' | 'walk' }[] = [
		{ label: 'System Description', group: 'System', oid: '1.3.6.1.2.1.1.1.0', op: 'get' },
		{ label: 'System Uptime', group: 'System', oid: '1.3.6.1.2.1.1.3.0', op: 'get' },
		{ label: 'Hostname', group: 'System', oid: '1.3.6.1.2.1.1.5.0', op: 'get' },
		{ label: 'CPU User %', group: 'CPU', oid: '1.3.6.1.4.1.2021.11.9.0', op: 'get' },
		{ label: 'CPU System %', group: 'CPU', oid: '1.3.6.1.4.1.2021.11.10.0', op: 'get' },
		{ label: 'CPU Idle %', group: 'CPU', oid: '1.3.6.1.4.1.2021.11.11.0', op: 'get' },
		{ label: 'Total RAM (KB)', group: 'Memory', oid: '1.3.6.1.4.1.2021.4.5.0', op: 'get' },
		{ label: 'Available RAM (KB)', group: 'Memory', oid: '1.3.6.1.4.1.2021.4.6.0', op: 'get' },
		{ label: 'Total Free Memory', group: 'Memory', oid: '1.3.6.1.4.1.2021.4.11.0', op: 'get' },
		{ label: 'Load Avg (1m)', group: 'Load', oid: '1.3.6.1.4.1.2021.10.1.3.1', op: 'get' },
		{ label: 'Load Avg (5m)', group: 'Load', oid: '1.3.6.1.4.1.2021.10.1.3.2', op: 'get' },
		{ label: 'Load Avg (15m)', group: 'Load', oid: '1.3.6.1.4.1.2021.10.1.3.3', op: 'get' },
		{ label: 'Interface Count', group: 'Network', oid: '1.3.6.1.2.1.2.1.0', op: 'get' },
		{ label: 'All Interfaces', group: 'Network', oid: '1.3.6.1.2.1.2.2', op: 'walk' },
		{ label: 'All Storage', group: 'Disk', oid: '1.3.6.1.2.1.25.2', op: 'walk' },
		{ label: 'All System Info', group: 'System', oid: '1.3.6.1.2.1.1', op: 'walk' },
	];

	function applyPreset(preset: typeof snmpPresets[0]) {
		snmpOperation = preset.op;
		if (preset.op === 'get') {
			snmpOid = preset.oid;
		} else {
			snmpOid = preset.oid;
		}
		if (!name) {
			name = preset.label;
		}
	}

	let loading = $state(false);
	let error = $state('');

	const monitorTypes: { value: MonitorType; label: string }[] = [
		{ value: 'http', label: 'HTTP' },
		{ value: 'tcp', label: 'TCP' },
		{ value: 'ping', label: 'Ping' },
		{ value: 'dns', label: 'DNS' },
		{ value: 'tls', label: 'TLS' },
		{ value: 'docker', label: 'Docker' },
		{ value: 'database', label: 'Database' },
		{ value: 'system', label: 'System' },
		{ value: 'service', label: 'Service' },
		{ value: 'port_scan', label: 'Port Scan' },
		{ value: 'snmp', label: 'SNMP' }
	];

	const targetPlaceholders: Record<MonitorType, string> = {
		http: 'https://example.com',
		tcp: 'host:port',
		ping: '192.168.1.1',
		dns: 'example.com',
		tls: 'example.com:443',
		docker: 'container_name',
		database: 'localhost:5432',
		system: 'localhost',
		service: 'nginx',
		port_scan: '192.168.1.1 or hostname',
		snmp: '192.168.1.1 or switch.local'
	};

	function buildMetadata(): Record<string, string> | undefined {
		const meta: Record<string, string> = {};

		if (type === 'http' && expectedStatus !== 200) {
			meta.expected_status = String(expectedStatus);
		} else if (type === 'http') {
			meta.expected_status = String(expectedStatus);
		}

		if (type === 'database') {
			meta.db_type = dbType;
			if (dbConnectionString) meta.connection_string = dbConnectionString;
			if (dbPassword) meta.password = dbPassword;
		}

		if (type === 'system') {
			meta.metric = systemMetric;
			meta.threshold = String(systemThreshold);
		}

		if (type === 'port_scan') {
			if (portScanPorts.trim()) meta.ports = portScanPorts.trim();
			if (portScanRange.trim()) meta.port_range = portScanRange.trim();
			if (portScanExpectedOpen.trim()) meta.expected_open = portScanExpectedOpen.trim();
			if (bannerGrab) meta.banner_grab = 'true';
		}

		if (type === 'snmp') {
			meta.version = snmpVersion;
			if (snmpOid.trim()) meta.oid = snmpOid.trim();
			if (snmpOids.trim()) meta.oids = snmpOids.trim();
			if (snmpOperation !== 'get') meta.operation = snmpOperation;
			if (snmpPort !== 161) meta.port = String(snmpPort);
			if (snmpTemplateId) meta.template_id = snmpTemplateId;
			if (snmpRateOids) meta.rate_oids = snmpRateOids;

			if (snmpVersion === '2c') {
				meta.community = snmpCommunity || 'public';
			} else {
				meta.username = snmpUsername;
				meta.security_level = snmpSecurityLevel;
				if (snmpSecurityLevel !== 'noAuthNoPriv') {
					meta.auth_protocol = snmpAuthProtocol;
					meta.auth_password = snmpAuthPassword;
				}
				if (snmpSecurityLevel === 'authPriv') {
					meta.privacy_protocol = snmpPrivacyProtocol;
					meta.privacy_password = snmpPrivacyPassword;
				}
			}
		}

		return Object.keys(meta).length > 0 ? meta : undefined;
	}

	function resetForm() {
		name = '';
		type = 'http';
		target = '';
		agentId = '';
		intervalSeconds = 30;
		timeoutSeconds = 10;
		failureThreshold = 3;
		expectedStatus = 200;
		dbType = 'postgres';
		dbConnectionString = '';
		dbPassword = '';
		systemMetric = 'cpu';
		systemThreshold = 80;
		portScanPorts = '';
		portScanRange = '';
		portScanExpectedOpen = '';
		bannerGrab = true;
		snmpVersion = '2c';
		snmpCommunity = 'public';
		snmpOid = '';
		snmpOids = '';
		snmpOperation = 'get';
		snmpPort = 161;
		snmpSecurityLevel = 'authNoPriv';
		snmpUsername = '';
		snmpAuthProtocol = 'SHA';
		snmpAuthPassword = '';
		snmpPrivacyProtocol = 'AES';
		snmpPrivacyPassword = '';
		snmpTemplateId = '';
		snmpRateOids = '';
		selectedTemplateId = '';
		error = '';
		loading = false;
	}

	function handleClose() {
		resetForm();
		onClose();
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();

		if (!name.trim()) {
			error = 'Name is required';
			return;
		}
		if (!target.trim()) {
			error = 'Target is required';
			return;
		}
		if (!agentId) {
			error = 'Please select an agent';
			return;
		}

		if (type === 'port_scan') {
			const portPattern = /^[\d,\s-]*$/;
			if (portScanPorts.trim() && !portPattern.test(portScanPorts)) {
				error = 'Ports must be comma-separated numbers (e.g. 80,443)';
				return;
			}
			if (portScanExpectedOpen.trim() && !portPattern.test(portScanExpectedOpen)) {
				error = 'Expected open ports must be comma-separated numbers';
				return;
			}
			if (!portScanPorts.trim() && !portScanRange.trim()) {
				error = 'Please specify ports or a port range';
				return;
			}
		}

		if (type === 'snmp') {
			if (!snmpOid.trim() && !snmpOids.trim()) {
				error = 'At least one OID is required';
				return;
			}
			const oidPattern = /^\d+(\.\d+)+$/;
			if (snmpOid.trim() && !oidPattern.test(snmpOid.trim())) {
				error = 'OID must be dotted-decimal format (e.g. 1.3.6.1.2.1.1.1.0)';
				return;
			}
			if (snmpVersion === '3' && !snmpUsername.trim()) {
				error = 'SNMPv3 requires a username';
				return;
			}
			if (snmpVersion === '3' && snmpSecurityLevel !== 'noAuthNoPriv' && !snmpAuthPassword) {
				error = 'Auth password is required for this security level';
				return;
			}
			if (snmpVersion === '3' && snmpSecurityLevel === 'authPriv' && !snmpPrivacyPassword) {
				error = 'Privacy password is required for authPriv security level';
				return;
			}
		}

		loading = true;
		error = '';

		try {
			await monitorsApi.createMonitor({
				name: name.trim(),
				type,
				target: target.trim(),
				agent_id: agentId,
				interval_seconds: intervalSeconds,
				timeout_seconds: timeoutSeconds,
				failure_threshold: failureThreshold,
				metadata: buildMetadata()
			});
			onCreated();
			handleClose();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create monitor';
		} finally {
			loading = false;
		}
	}
</script>

<Modal bind:open onclose={handleClose} size="lg">
	<div class="flex items-center justify-between mb-4">
		<h3 class="text-sm font-medium text-foreground">Create Monitor</h3>
		<button
			type="button"
			onclick={handleClose}
			class="text-muted-foreground hover:text-foreground transition-colors"
			aria-label="Close"
		>
			<X class="w-4 h-4" />
		</button>
	</div>

	<form onsubmit={handleSubmit}>
		<div class="space-y-4">
			{#if error}
				<div class="bg-destructive/10 border border-destructive/20 rounded-md px-3 py-2 flex items-center space-x-2" role="alert">
					<AlertCircle class="w-3.5 h-3.5 text-destructive flex-shrink-0" />
					<span class="text-xs text-destructive">{error}</span>
				</div>
			{/if}

			<SharedFields
				bind:name
				bind:type
				bind:agentId
				bind:target
				bind:intervalSeconds
				bind:timeoutSeconds
				bind:failureThreshold
				{agents}
				{monitorTypes}
				targetPlaceholder={targetPlaceholders[type]}
			/>

			{#if type === 'http'}
				<HTTPSection bind:expectedStatus />
			{:else if type === 'database'}
				<DatabaseSection bind:dbType bind:dbConnectionString bind:dbPassword />
			{:else if type === 'system'}
				<SystemSection bind:systemMetric bind:systemThreshold />
			{:else if type === 'port_scan'}
				<PortScanSection
					bind:portScanPorts
					bind:portScanRange
					bind:portScanExpectedOpen
					bind:bannerGrab
				/>
			{:else if type === 'snmp'}
				<SNMPSection
					bind:snmpVersion
					bind:snmpCommunity
					bind:snmpOid
					bind:snmpOids
					bind:snmpOperation
					bind:snmpPort
					bind:snmpSecurityLevel
					bind:snmpUsername
					bind:snmpAuthProtocol
					bind:snmpAuthPassword
					bind:snmpPrivacyProtocol
					bind:snmpPrivacyPassword
					bind:selectedTemplateId
					{deviceTemplates}
					{snmpPresets}
					onApplyTemplate={applyTemplate}
					onApplyPreset={applyPreset}
				/>
			{/if}
		</div>

		<div class="mt-4 pt-4 border-t border-border flex justify-end space-x-2">
			<Button variant="outline" size="sm" onclick={handleClose}>Cancel</Button>
			<Button variant="primary" size="sm" type="submit" disabled={loading}>
				{loading ? 'Creating...' : 'Create Monitor'}
			</Button>
		</div>
	</form>
</Modal>
