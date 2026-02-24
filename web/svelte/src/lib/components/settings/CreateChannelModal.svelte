<script lang="ts">
	import { X, AlertCircle } from 'lucide-svelte';
	import { settings as settingsApi } from '$lib/api';
	import type { AlertChannelType } from '$lib/types';

	interface Props {
		open: boolean;
		onClose: () => void;
		onCreated: () => void;
	}

	let { open = $bindable(), onClose, onCreated }: Props = $props();

	// Form state
	let channelType = $state<AlertChannelType>('discord');
	let name = $state('');
	let loading = $state(false);
	let error = $state('');

	// Discord / Slack
	let webhookUrl = $state('');

	// Email
	let emailHost = $state('');
	let emailPort = $state('587');
	let emailUsername = $state('');
	let emailPassword = $state('');
	let emailFrom = $state('');
	let emailTo = $state('');

	// Telegram
	let telegramBotToken = $state('');
	let telegramChatId = $state('');

	// PagerDuty
	let pagerdutyRoutingKey = $state('');

	// Webhook
	let webhookCustomUrl = $state('');

	const typeOptions: { value: AlertChannelType; label: string }[] = [
		{ value: 'discord', label: 'Discord' },
		{ value: 'slack', label: 'Slack' },
		{ value: 'email', label: 'Email' },
		{ value: 'telegram', label: 'Telegram' },
		{ value: 'pagerduty', label: 'PagerDuty' },
		{ value: 'webhook', label: 'Webhook' }
	];

	const inputClass = 'w-full px-3 py-2 bg-card-elevated border border-border rounded-md text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background';
	const labelClass = 'block text-xs font-medium text-muted-foreground mb-1.5';

	function buildConfig(): Record<string, string> {
		switch (channelType) {
			case 'discord':
				return { webhook_url: webhookUrl };
			case 'slack':
				return { webhook_url: webhookUrl };
			case 'email':
				return {
					host: emailHost,
					port: emailPort,
					username: emailUsername,
					password: emailPassword,
					from: emailFrom,
					to: emailTo
				};
			case 'telegram':
				return { bot_token: telegramBotToken, chat_id: telegramChatId };
			case 'pagerduty':
				return { routing_key: pagerdutyRoutingKey };
			case 'webhook':
				return { url: webhookCustomUrl };
			default:
				return {};
		}
	}

	function resetForm() {
		channelType = 'discord';
		name = '';
		error = '';
		loading = false;
		webhookUrl = '';
		emailHost = '';
		emailPort = '587';
		emailUsername = '';
		emailPassword = '';
		emailFrom = '';
		emailTo = '';
		telegramBotToken = '';
		telegramChatId = '';
		pagerdutyRoutingKey = '';
		webhookCustomUrl = '';
	}

	function handleClose() {
		resetForm();
		onClose();
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();

		if (!name.trim()) {
			error = 'Name is required.';
			return;
		}

		loading = true;
		error = '';

		try {
			await settingsApi.createChannel({
				type: channelType,
				name: name.trim(),
				config: buildConfig()
			});
			onCreated();
			handleClose();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create channel.';
		} finally {
			loading = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') handleClose();
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm animate-fade-in"
		onkeydown={handleKeydown}
	>
		<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
		<div
			class="bg-card border border-border rounded-lg shadow-lg w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto animate-fade-in-up"
			onclick={(e) => e.stopPropagation()}
			role="dialog"
			aria-label="Add alert channel"
		>
			<!-- Header -->
			<div class="px-5 py-3.5 border-b border-border flex items-center justify-between">
				<h3 class="text-sm font-medium text-foreground">Add Alert Channel</h3>
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

					<!-- Type + Name row -->
					<div class="grid grid-cols-2 gap-3">
						<div>
							<label for="channel-type" class={labelClass}>Type</label>
							<select
								id="channel-type"
								bind:value={channelType}
								class={inputClass}
							>
								{#each typeOptions as opt}
									<option value={opt.value}>{opt.label}</option>
								{/each}
							</select>
						</div>
						<div>
							<label for="channel-name" class={labelClass}>Name</label>
							<input
								id="channel-name"
								type="text"
								bind:value={name}
								required
								placeholder="e.g. Production Alerts"
								class={inputClass}
							/>
						</div>
					</div>

					<!-- Dynamic config fields -->
					{#if channelType === 'discord' || channelType === 'slack'}
						<div>
							<label for="channel-webhook-url" class={labelClass}>Webhook URL</label>
							<input
								id="channel-webhook-url"
								type="url"
								bind:value={webhookUrl}
								required
								placeholder={channelType === 'discord'
									? 'https://discord.com/api/webhooks/...'
									: 'https://hooks.slack.com/services/...'}
								class={inputClass}
							/>
						</div>
					{/if}

					{#if channelType === 'email'}
						<div class="space-y-3 pt-1">
							<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">SMTP Settings</div>
							<div class="grid grid-cols-2 gap-3">
								<div>
									<label for="channel-email-host" class={labelClass}>Host</label>
									<input
										id="channel-email-host"
										type="text"
										bind:value={emailHost}
										required
										placeholder="smtp.gmail.com"
										class={inputClass}
									/>
								</div>
								<div>
									<label for="channel-email-port" class={labelClass}>Port</label>
									<input
										id="channel-email-port"
										type="text"
										bind:value={emailPort}
										required
										placeholder="587"
										class={inputClass}
									/>
								</div>
							</div>
							<div class="grid grid-cols-2 gap-3">
								<div>
									<label for="channel-email-username" class={labelClass}>Username</label>
									<input
										id="channel-email-username"
										type="text"
										bind:value={emailUsername}
										required
										placeholder="user@gmail.com"
										class={inputClass}
									/>
								</div>
								<div>
									<label for="channel-email-password" class={labelClass}>Password</label>
									<input
										id="channel-email-password"
										type="password"
										bind:value={emailPassword}
										required
										placeholder="App password"
										class={inputClass}
									/>
								</div>
							</div>
							<div class="grid grid-cols-2 gap-3">
								<div>
									<label for="channel-email-from" class={labelClass}>From</label>
									<input
										id="channel-email-from"
										type="email"
										bind:value={emailFrom}
										required
										placeholder="alerts@example.com"
										class={inputClass}
									/>
								</div>
								<div>
									<label for="channel-email-to" class={labelClass}>To</label>
									<input
										id="channel-email-to"
										type="email"
										bind:value={emailTo}
										required
										placeholder="oncall@example.com"
										class={inputClass}
									/>
								</div>
							</div>
						</div>
					{/if}

					{#if channelType === 'telegram'}
						<div class="space-y-3 pt-1">
							<div class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">Telegram Settings</div>
							<div>
								<label for="channel-telegram-token" class={labelClass}>Bot Token</label>
								<input
									id="channel-telegram-token"
									type="text"
									bind:value={telegramBotToken}
									required
									placeholder="123456789:ABCdef..."
									class={inputClass}
								/>
							</div>
							<div>
								<label for="channel-telegram-chat" class={labelClass}>Chat ID</label>
								<input
									id="channel-telegram-chat"
									type="text"
									bind:value={telegramChatId}
									required
									placeholder="-1001234567890"
									class={inputClass}
								/>
							</div>
						</div>
					{/if}

					{#if channelType === 'pagerduty'}
						<div>
							<label for="channel-pagerduty-key" class={labelClass}>Routing Key</label>
							<input
								id="channel-pagerduty-key"
								type="text"
								bind:value={pagerdutyRoutingKey}
								required
								placeholder="e93facc04764012d7bfb002500d5d1a6"
								class={inputClass}
							/>
						</div>
					{/if}

					{#if channelType === 'webhook'}
						<div>
							<label for="channel-webhook-custom-url" class={labelClass}>URL</label>
							<input
								id="channel-webhook-custom-url"
								type="url"
								bind:value={webhookCustomUrl}
								required
								placeholder="https://api.example.com/webhooks/alerts"
								class={inputClass}
							/>
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
						{loading ? 'Creating...' : 'Create Channel'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
