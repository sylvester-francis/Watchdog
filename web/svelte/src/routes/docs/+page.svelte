<script lang="ts">
	import { onMount } from 'svelte';

	onMount(() => {
		// Load Swagger UI CSS
		const link = document.createElement('link');
		link.rel = 'stylesheet';
		link.href = 'https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.18.2/swagger-ui.css';
		document.head.appendChild(link);

		// Load Swagger UI JS
		const bundleScript = document.createElement('script');
		bundleScript.src = 'https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.18.2/swagger-ui-bundle.js';
		bundleScript.onload = () => {
			const presetScript = document.createElement('script');
			presetScript.src = 'https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.18.2/swagger-ui-standalone-preset.js';
			presetScript.onload = () => {
				// @ts-ignore - SwaggerUIBundle loaded via CDN
				window.SwaggerUIBundle({
					url: '/openapi.json',
					dom_id: '#swagger-ui',
					deepLinking: true,
					presets: [
						// @ts-ignore
						window.SwaggerUIBundle.presets.apis,
					],
					defaultModelsExpandDepth: 1,
					defaultModelExpandDepth: 2,
					docExpansion: 'list',
					tryItOutEnabled: true,
					persistAuthorization: true,
					validatorUrl: null,
				});
			};
			document.head.appendChild(presetScript);
		};
		document.head.appendChild(bundleScript);

		return () => {
			link.remove();
			bundleScript.remove();
		};
	});
</script>

<svelte:head>
	<title>API Docs - WatchDog</title>
</svelte:head>

<div id="swagger-ui" class="swagger-wrapper"></div>

<style>
	.swagger-wrapper {
		background: #ffffff;
		color: #3b4151;
		min-height: 100vh;
	}
</style>
