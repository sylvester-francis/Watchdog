<script lang="ts">
	import { X, AlertCircle } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import type { Agent, MonitorType } from '$lib/types';

	interface Props {
		open: boolean;
		agents: Agent[];
		onClose: () => void;
		onCreated: () => void;
	}

	let { open = $bindable(), agents, onClose, onCreated }: Props = $props();

	// Form state
	let name = $state('');
	let type = $state<MonitorType>('http');
	let target = $state('');
	let agentId = $state('');
	let intervalSeconds = $state(30);
	let timeoutSeconds = $state(10);
	let failureThreshold = $state(3);

	// HTTP-specific
	let expectedStatus = $state(200);

	// Database-specific
	let dbType = $state('postgres');
	let dbConnectionString = $state('');
	let dbPassword = $state('');

	// System-specific
	let systemMetric = $state('cpu');
	let systemThreshold = $state(80);

	// Port scan-specific
	let portScanPorts = $state('');
	let portScanRange = $state('');
	let portScanExpectedOpen = $state('');
	let bannerGrab = $state(true);

	// SNMP-specific
	let snmpVersion = $state<'2c' | '3'>('2c');
	let snmpCommunity = $state('public');
	let snmpOid = $state('');
	let snmpOids = $state('');
	let snmpOperation = $state<'get' | 'walk' | 'bulk'>('get');
	let snmpPort = $state(161);
	// SNMPv3
	let snmpSecurityLevel = $state<'noAuthNoPriv' | 'authNoPriv' | 'authPriv'>('authNoPriv');
	let snmpUsername = $state('');
	let snmpAuthProtocol = $state('SHA');
	let snmpAuthPassword = $state('');
	let snmpPrivacyProtocol = $state('AES');
	let snmpPrivacyPassword = $state('');

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

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') handleClose();
	}

	const inputClass = 'w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background';
	const labelClass = 'block text-xs font-medium text-muted-foreground mb-1.5';
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm animate-fade-in"
		onkeydown={handleKeydown}
	>
		<div
			class="bg-card border border-border rounded-lg shadow-lg w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto animate-fade-in-up"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
			role="dialog"
			aria-label="Create monitor"
			tabindex="-1"
		>
			<!-- Header -->
			<div class="px-5 py-3.5 border-b border-border flex items-center justify-between">
				<h3 class="text-sm font-medium text-foreground">Create Monitor</h3>
				<button
					onclick={handleClose}
					class="text-muted-foreground hover:text-foreground transition-colors"
					aria-label="Close"
				>
					<X class="w-4 h-4" />
				</button>
			</div>

			<!-- Form -->
			<form onsubmit={handleSubmit}>
				<div class="p-5 space-y-4">
					{#if error}
						<div class="bg-destructive/10 border border-destructive/20 rounded-md px-3 py-2 flex items-center space-x-2" role="alert">
							<AlertCircle class="w-3.5 h-3.5 text-destructive flex-shrink-0" />
							<span class="text-xs text-destructive">{error}</span>
						</div>
					{/if}

					<!-- Name -->
					<div>
						<label for="monitor-name" class={labelClass}>Name</label>
						<input
							id="monitor-name"
							type="text"
							bind:value={name}
							required
							placeholder="My API Server"
							class={inputClass}
						/>
					</div>

					<!-- Type + Agent row -->
					<div class="grid grid-cols-2 gap-3">
						<div>
							<label for="monitor-type" class={labelClass}>Type</label>
							<select
								id="monitor-type"
								bind:value={type}
								class={inputClass}
							>
								{#each monitorTypes as mt}
									<option value={mt.value}>{mt.label}</option>
								{/each}
							</select>
						</div>
						<div>
							<label for="monitor-agent" class={labelClass}>Agent</label>
							<select
								id="monitor-agent"
								bind:value={agentId}
								required
								class={inputClass}
							>
								<option value="" disabled>Select agent</option>
								{#each agents as agent}
									<option value={agent.id}>{agent.name}</option>
								{/each}
							</select>
						</div>
					</div>

					<!-- Target -->
					<div>
						<label for="monitor-target" class={labelClass}>Target</label>
						<input
							id="monitor-target"
							type="text"
							bind:value={target}
							required
							placeholder={targetPlaceholders[type]}
							class={inputClass}
						/>
					</div>

					<!-- Interval / Timeout / Failure Threshold -->
					<div class="grid grid-cols-3 gap-3">
						<div>
							<label for="monitor-interval" class={labelClass}>Interval (s)</label>
							<input
								id="monitor-interval"
								type="number"
								bind:value={intervalSeconds}
								min="5"
								max="3600"
								class={inputClass}
							/>
						</div>
						<div>
							<label for="monitor-timeout" class={labelClass}>Timeout (s)</label>
							<input
								id="monitor-timeout"
								type="number"
								bind:value={timeoutSeconds}
								min="1"
								max="300"
								class={inputClass}
							/>
						</div>
						<div>
							<label for="monitor-threshold" class={labelClass}>Fail Threshold</label>
							<input
								id="monitor-threshold"
								type="number"
								bind:value={failureThreshold}
								min="1"
								max="10"
								class={inputClass}
							/>
						</div>
					</div>

					<!-- Type-specific fields -->
					{#if type === 'http'}
						<div>
							<label for="monitor-expected-status" class={labelClass}>Expected Status Code</label>
							<input
								id="monitor-expected-status"
								type="number"
								bind:value={expectedStatus}
								min="100"
								max="599"
								class={inputClass}
							/>
						</div>
					{/if}

					{#if type === 'database'}
						<div class="space-y-3 pt-1">
							<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">Database Settings</div>
							<div>
								<label for="monitor-db-type" class={labelClass}>DB Type</label>
								<select
									id="monitor-db-type"
									bind:value={dbType}
									class={inputClass}
								>
									<option value="postgres">PostgreSQL</option>
									<option value="mysql">MySQL</option>
									<option value="redis">Redis</option>
									<option value="mongodb">MongoDB</option>
								</select>
							</div>
							<div>
								<label for="monitor-db-conn" class={labelClass}>Connection String</label>
								<input
									id="monitor-db-conn"
									type="text"
									bind:value={dbConnectionString}
									placeholder="host=localhost port=5432 dbname=mydb user=postgres"
									class={inputClass}
								/>
							</div>
							<div>
								<label for="monitor-db-pass" class={labelClass}>Password</label>
								<input
									id="monitor-db-pass"
									type="password"
									bind:value={dbPassword}
									placeholder="Database password"
									class={inputClass}
								/>
							</div>
						</div>
					{/if}

					{#if type === 'system'}
						<div class="space-y-3 pt-1">
							<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">System Metric</div>
							<div class="grid grid-cols-2 gap-3">
								<div>
									<label for="monitor-sys-metric" class={labelClass}>Metric</label>
									<select
										id="monitor-sys-metric"
										bind:value={systemMetric}
										class={inputClass}
									>
										<option value="cpu">CPU</option>
										<option value="memory">Memory</option>
										<option value="disk">Disk</option>
									</select>
								</div>
								<div>
									<label for="monitor-sys-threshold" class={labelClass}>Threshold %</label>
									<input
										id="monitor-sys-threshold"
										type="number"
										bind:value={systemThreshold}
										min="1"
										max="100"
										class={inputClass}
									/>
								</div>
							</div>
						</div>
					{/if}

					{#if type === 'port_scan'}
						<div class="space-y-3 pt-1">
							<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">Port Scan Settings</div>
							<div>
								<label for="monitor-ps-ports" class={labelClass}>Ports (comma-separated)</label>
								<input
									id="monitor-ps-ports"
									type="text"
									bind:value={portScanPorts}
									placeholder="22,80,443,3306,8080"
									class={inputClass}
								/>
								<p class="text-[10px] text-muted-foreground mt-1">Supports ranges: 8000-9000</p>
							</div>
							<div>
								<label for="monitor-ps-range" class={labelClass}>Port Range (alternative)</label>
								<input
									id="monitor-ps-range"
									type="text"
									bind:value={portScanRange}
									placeholder="1-1024"
									class={inputClass}
								/>
							</div>
							<div>
								<label for="monitor-ps-expected" class={labelClass}>Expected Open Ports (optional)</label>
								<input
									id="monitor-ps-expected"
									type="text"
									bind:value={portScanExpectedOpen}
									placeholder="22,80,443"
									class={inputClass}
								/>
								<p class="text-[10px] text-muted-foreground mt-1">If set, alerts when expected ports close or unexpected ports open</p>
							</div>
							<div class="flex items-center justify-between pt-1">
								<div>
									<label for="monitor-banner-grab" class={labelClass}>Service Detection</label>
									<p class="text-[10px] text-muted-foreground">Identify services and versions on open ports</p>
								</div>
								<label class="relative inline-flex items-center cursor-pointer">
									<input
										id="monitor-banner-grab"
										type="checkbox"
										bind:checked={bannerGrab}
										class="sr-only peer"
									/>
									<div class="w-9 h-5 bg-muted rounded-full peer peer-checked:bg-accent transition-colors after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-4"></div>
								</label>
							</div>
						</div>
					{/if}

					{#if type === 'snmp'}
						<div class="space-y-3 pt-1">
							<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">SNMP Settings</div>

							<!-- Version + Operation row -->
							<div class="grid grid-cols-3 gap-3">
								<div>
									<label for="monitor-snmp-version" class={labelClass}>Version</label>
									<select
										id="monitor-snmp-version"
										bind:value={snmpVersion}
										class={inputClass}
									>
										<option value="2c">v2c</option>
										<option value="3">v3</option>
									</select>
								</div>
								<div>
									<label for="monitor-snmp-operation" class={labelClass}>Operation</label>
									<select
										id="monitor-snmp-operation"
										bind:value={snmpOperation}
										class={inputClass}
									>
										<option value="get">GET</option>
										<option value="walk">Walk</option>
										<option value="bulk">Bulk GET</option>
									</select>
								</div>
								<div>
									<label for="monitor-snmp-port" class={labelClass}>Port</label>
									<input
										id="monitor-snmp-port"
										type="number"
										bind:value={snmpPort}
										min="1"
										max="65535"
										class={inputClass}
									/>
								</div>
							</div>

							<!-- OID -->
							<div>
								<label for="monitor-snmp-oid" class={labelClass}>OID</label>
								<input
									id="monitor-snmp-oid"
									type="text"
									bind:value={snmpOid}
									placeholder="1.3.6.1.2.1.1.1.0 (sysDescr)"
									class={inputClass}
								/>
							</div>

							<!-- Multiple OIDs (optional) -->
							<div>
								<label for="monitor-snmp-oids" class={labelClass}>Additional OIDs (optional, comma-separated)</label>
								<input
									id="monitor-snmp-oids"
									type="text"
									bind:value={snmpOids}
									placeholder="1.3.6.1.2.1.1.3.0, 1.3.6.1.2.1.1.5.0"
									class={inputClass}
								/>
							</div>

							<!-- v2c: Community string -->
							{#if snmpVersion === '2c'}
								<div>
									<label for="monitor-snmp-community" class={labelClass}>Community String</label>
									<input
										id="monitor-snmp-community"
										type="text"
										bind:value={snmpCommunity}
										placeholder="public"
										class={inputClass}
									/>
								</div>
							{/if}

							<!-- v3: Auth settings -->
							{#if snmpVersion === '3'}
								<div class="space-y-3 pt-1 border-t border-border/50">
									<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium pt-2">SNMPv3 Authentication</div>

									<div class="grid grid-cols-2 gap-3">
										<div>
											<label for="monitor-snmp-username" class={labelClass}>Username</label>
											<input
												id="monitor-snmp-username"
												type="text"
												bind:value={snmpUsername}
												placeholder="snmpuser"
												class={inputClass}
											/>
										</div>
										<div>
											<label for="monitor-snmp-seclevel" class={labelClass}>Security Level</label>
											<select
												id="monitor-snmp-seclevel"
												bind:value={snmpSecurityLevel}
												class={inputClass}
											>
												<option value="noAuthNoPriv">No Auth, No Privacy</option>
												<option value="authNoPriv">Auth, No Privacy</option>
												<option value="authPriv">Auth + Privacy</option>
											</select>
										</div>
									</div>

									{#if snmpSecurityLevel !== 'noAuthNoPriv'}
										<div class="grid grid-cols-2 gap-3">
											<div>
												<label for="monitor-snmp-authproto" class={labelClass}>Auth Protocol</label>
												<select
													id="monitor-snmp-authproto"
													bind:value={snmpAuthProtocol}
													class={inputClass}
												>
													<option value="MD5">MD5</option>
													<option value="SHA">SHA</option>
													<option value="SHA224">SHA-224</option>
													<option value="SHA256">SHA-256</option>
													<option value="SHA384">SHA-384</option>
													<option value="SHA512">SHA-512</option>
												</select>
											</div>
											<div>
												<label for="monitor-snmp-authpass" class={labelClass}>Auth Password</label>
												<input
													id="monitor-snmp-authpass"
													type="password"
													bind:value={snmpAuthPassword}
													placeholder="Auth passphrase"
													class={inputClass}
												/>
											</div>
										</div>
									{/if}

									{#if snmpSecurityLevel === 'authPriv'}
										<div class="grid grid-cols-2 gap-3">
											<div>
												<label for="monitor-snmp-privproto" class={labelClass}>Privacy Protocol</label>
												<select
													id="monitor-snmp-privproto"
													bind:value={snmpPrivacyProtocol}
													class={inputClass}
												>
													<option value="DES">DES</option>
													<option value="AES">AES</option>
													<option value="AES192">AES-192</option>
													<option value="AES256">AES-256</option>
												</select>
											</div>
											<div>
												<label for="monitor-snmp-privpass" class={labelClass}>Privacy Password</label>
												<input
													id="monitor-snmp-privpass"
													type="password"
													bind:value={snmpPrivacyPassword}
													placeholder="Privacy passphrase"
													class={inputClass}
												/>
											</div>
										</div>
									{/if}
								</div>
							{/if}
						</div>
					{/if}
				</div>

				<!-- Footer -->
				<div class="px-5 py-3.5 border-t border-border flex justify-end space-x-2">
					<button
						type="button"
						onclick={handleClose}
						class="px-4 py-2 bg-muted text-muted-foreground hover:bg-muted/80 text-xs font-medium rounded-md transition-colors"
					>
						Cancel
					</button>
					<button
						type="submit"
						disabled={loading}
						class="px-4 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors disabled:opacity-50"
					>
						{loading ? 'Creating...' : 'Create Monitor'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
