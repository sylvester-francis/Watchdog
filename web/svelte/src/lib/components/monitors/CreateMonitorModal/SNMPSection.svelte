<script lang="ts">
	import type { DeviceTemplate } from '$lib/types';
	import { FormField } from '@sylvester-francis/watchdog-ui';
	import { Input } from '@sylvester-francis/watchdog-ui';
	import { Select } from '@sylvester-francis/watchdog-ui';

	type SNMPPreset = { label: string; group: string; oid: string; op: 'get' | 'walk' };

	interface Props {
		snmpVersion: '2c' | '3';
		snmpCommunity: string;
		snmpOid: string;
		snmpOids: string;
		snmpOperation: 'get' | 'walk' | 'bulk';
		snmpPort: number;
		snmpSecurityLevel: 'noAuthNoPriv' | 'authNoPriv' | 'authPriv';
		snmpUsername: string;
		snmpAuthProtocol: string;
		snmpAuthPassword: string;
		snmpPrivacyProtocol: string;
		snmpPrivacyPassword: string;
		selectedTemplateId: string;
		deviceTemplates: DeviceTemplate[];
		snmpPresets: SNMPPreset[];
		onApplyTemplate: (id: string) => void;
		onApplyPreset: (preset: SNMPPreset) => void;
	}

	let {
		snmpVersion = $bindable(),
		snmpCommunity = $bindable(),
		snmpOid = $bindable(),
		snmpOids = $bindable(),
		snmpOperation = $bindable(),
		snmpPort = $bindable(),
		snmpSecurityLevel = $bindable(),
		snmpUsername = $bindable(),
		snmpAuthProtocol = $bindable(),
		snmpAuthPassword = $bindable(),
		snmpPrivacyProtocol = $bindable(),
		snmpPrivacyPassword = $bindable(),
		selectedTemplateId = $bindable(),
		deviceTemplates,
		snmpPresets,
		onApplyTemplate,
		onApplyPreset,
	}: Props = $props();

	const plainSelectClass = 'w-full bg-card-elevated border border-border rounded px-3 py-2 text-foreground transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-1 focus:ring-offset-background';
</script>

<div class="space-y-3 pt-1">
	<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">SNMP Settings</div>

	{#if deviceTemplates.length > 0}
		<FormField label="Device Template" htmlFor="monitor-snmp-template">
			<select
				id="monitor-snmp-template"
				bind:value={selectedTemplateId}
				onchange={(e) => onApplyTemplate((e.target as HTMLSelectElement).value)}
				class={plainSelectClass}
			>
				<option value="">Custom (manual OID entry)</option>
				{#each deviceTemplates as tmpl}
					<option value={tmpl.id}>{tmpl.vendor} {tmpl.model} ({tmpl.oid_count} OIDs)</option>
				{/each}
			</select>
			{#if selectedTemplateId}
				<p class="text-[10px] text-muted-foreground mt-1">OIDs auto-filled from template. You can still edit them below.</p>
			{/if}
		</FormField>
	{/if}

	<FormField label="Quick Setup (Preset)" htmlFor="monitor-snmp-preset">
		<select
			id="monitor-snmp-preset"
			onchange={(e) => {
				const idx = parseInt((e.target as HTMLSelectElement).value);
				if (idx >= 0) onApplyPreset(snmpPresets[idx]);
				(e.target as HTMLSelectElement).value = '-1';
			}}
			class={plainSelectClass}
		>
			<option value="-1">Select a common OID...</option>
			{#each ['System', 'CPU', 'Memory', 'Load', 'Network', 'Disk'] as group}
				<optgroup label={group}>
					{#each snmpPresets as preset, i}
						{#if preset.group === group}
							<option value={i}>{preset.label} ({preset.op})</option>
						{/if}
					{/each}
				</optgroup>
			{/each}
		</select>
	</FormField>

	<div class="grid grid-cols-3 gap-3">
		<FormField label="Version" htmlFor="monitor-snmp-version">
			<Select id="monitor-snmp-version" bind:value={snmpVersion}>
				<option value="2c">v2c</option>
				<option value="3">v3</option>
			</Select>
		</FormField>
		<FormField label="Operation" htmlFor="monitor-snmp-operation">
			<Select id="monitor-snmp-operation" bind:value={snmpOperation}>
				<option value="get">GET</option>
				<option value="walk">Walk</option>
				<option value="bulk">Bulk GET</option>
			</Select>
		</FormField>
		<FormField label="Port" htmlFor="monitor-snmp-port">
			<Input id="monitor-snmp-port" type="number" bind:value={snmpPort} min={1} max={65535} />
		</FormField>
	</div>

	<FormField label="OID" htmlFor="monitor-snmp-oid">
		<Input id="monitor-snmp-oid" type="text" bind:value={snmpOid} placeholder="1.3.6.1.2.1.1.1.0 (sysDescr)" />
	</FormField>

	<FormField label="Additional OIDs (optional, comma-separated)" htmlFor="monitor-snmp-oids">
		<Input id="monitor-snmp-oids" type="text" bind:value={snmpOids} placeholder="1.3.6.1.2.1.1.3.0, 1.3.6.1.2.1.1.5.0" />
	</FormField>

	{#if snmpVersion === '2c'}
		<FormField label="Community String" htmlFor="monitor-snmp-community">
			<Input id="monitor-snmp-community" type="text" bind:value={snmpCommunity} placeholder="public" />
		</FormField>
	{/if}

	{#if snmpVersion === '3'}
		<div class="space-y-3 pt-1 border-t border-border/50">
			<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium pt-2">SNMPv3 Authentication</div>

			<div class="grid grid-cols-2 gap-3">
				<FormField label="Username" htmlFor="monitor-snmp-username">
					<Input id="monitor-snmp-username" type="text" bind:value={snmpUsername} placeholder="snmpuser" />
				</FormField>
				<FormField label="Security Level" htmlFor="monitor-snmp-seclevel">
					<Select id="monitor-snmp-seclevel" bind:value={snmpSecurityLevel}>
						<option value="noAuthNoPriv">No Auth, No Privacy</option>
						<option value="authNoPriv">Auth, No Privacy</option>
						<option value="authPriv">Auth + Privacy</option>
					</Select>
				</FormField>
			</div>

			{#if snmpSecurityLevel !== 'noAuthNoPriv'}
				<div class="grid grid-cols-2 gap-3">
					<FormField label="Auth Protocol" htmlFor="monitor-snmp-authproto">
						<Select id="monitor-snmp-authproto" bind:value={snmpAuthProtocol}>
							<option value="MD5">MD5</option>
							<option value="SHA">SHA</option>
							<option value="SHA224">SHA-224</option>
							<option value="SHA256">SHA-256</option>
							<option value="SHA384">SHA-384</option>
							<option value="SHA512">SHA-512</option>
						</Select>
					</FormField>
					<FormField label="Auth Password" htmlFor="monitor-snmp-authpass">
						<Input id="monitor-snmp-authpass" type="password" bind:value={snmpAuthPassword} placeholder="Auth passphrase" />
					</FormField>
				</div>
			{/if}

			{#if snmpSecurityLevel === 'authPriv'}
				<div class="grid grid-cols-2 gap-3">
					<FormField label="Privacy Protocol" htmlFor="monitor-snmp-privproto">
						<Select id="monitor-snmp-privproto" bind:value={snmpPrivacyProtocol}>
							<option value="DES">DES</option>
							<option value="AES">AES</option>
							<option value="AES192">AES-192</option>
							<option value="AES256">AES-256</option>
						</Select>
					</FormField>
					<FormField label="Privacy Password" htmlFor="monitor-snmp-privpass">
						<Input id="monitor-snmp-privpass" type="password" bind:value={snmpPrivacyPassword} placeholder="Privacy passphrase" />
					</FormField>
				</div>
			{/if}
		</div>
	{/if}
</div>
