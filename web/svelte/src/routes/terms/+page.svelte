<svelte:head>
	<title>Terms of Service - WatchDog</title>
</svelte:head>

<div class="bg-background text-foreground font-sans min-h-screen antialiased">
	<div class="max-w-2xl mx-auto px-6 py-12">
		<a href="/" class="inline-flex items-center space-x-1 text-xs text-muted-foreground hover:text-foreground transition-colors mb-8">
			<span>&larr;</span>
			<span>Back to WatchDog</span>
		</a>

		<h1 class="text-2xl font-bold text-foreground mb-2">Terms of Service</h1>
		<p class="text-xs text-muted-foreground mb-8">Last updated: February 16, 2026</p>

		<div class="space-y-6 text-sm text-muted-foreground leading-relaxed">

			<section>
				<h2 class="text-base font-semibold text-foreground mb-2">1. Beta Disclaimer</h2>
				<p>WatchDog is currently in <strong class="text-foreground">open beta</strong>. The software is provided "as-is" without warranties of any kind, either express or implied, including but not limited to warranties of merchantability, fitness for a particular purpose, or non-infringement. Features may change, data may be reset, and the service may experience downtime without prior notice.</p>
			</section>

			<section>
				<h2 class="text-base font-semibold text-foreground mb-2">2. Service Description</h2>
				<p>WatchDog is an open-source, agent-based infrastructure monitoring platform. You deploy a lightweight agent binary inside your network. The agent performs outbound-only network checks (HTTP, TCP, ping, DNS, TLS) against targets you configure and reports results to the WatchDog hub over WebSocket. The agent does not access file systems, credentials, or user data on the machines where it runs.</p>
			</section>

			<section>
				<h2 class="text-base font-semibold text-foreground mb-2">3. Agent Software</h2>
				<p>By installing the WatchDog agent on your infrastructure, you accept responsibility for its deployment and operation. The agent binary is open-source (licensed under AGPL-3.0) and fully auditable. It performs only the network checks you configure through the hub dashboard or API. You are responsible for ensuring the agent's use complies with your organization's policies and applicable laws.</p>
			</section>

			<section>
				<h2 class="text-base font-semibold text-foreground mb-2">4. Limitation of Liability</h2>
				<p>To the maximum extent permitted by law, WatchDog and its developer shall not be liable for any indirect, incidental, special, consequential, or punitive damages, including but not limited to loss of profits, data, or business opportunities, arising out of or related to your use of the service. This includes, without limitation:</p>
				<ul class="list-disc list-inside mt-2 space-y-1 ml-2">
					<li>Service downtime or unavailability</li>
					<li>Data loss or corruption</li>
					<li>Security incidents affecting your infrastructure</li>
					<li>Missed or delayed alerts</li>
					<li>Any actions taken or not taken based on monitoring data</li>
				</ul>
				<p class="mt-2">Total aggregate liability is limited to the amount of fees you have paid for the service.</p>
			</section>

			<section>
				<h2 class="text-base font-semibold text-foreground mb-2">5. Service Availability</h2>
				<p>WatchDog provides no uptime guarantees. The service may be modified, suspended, or discontinued at any time with reasonable notice. During the beta period, features may change and data may be wiped.</p>
			</section>

			<section>
				<h2 class="text-base font-semibold text-foreground mb-2">6. Account Termination</h2>
				<p>Either party may terminate the relationship at any time. You may delete your account by contacting us. Upon termination, your data will be deleted within 30 days. We may terminate or suspend accounts that violate these terms.</p>
			</section>

			<section>
				<h2 class="text-base font-semibold text-foreground mb-2">7. Acceptable Use</h2>
				<p>You agree not to:</p>
				<ul class="list-disc list-inside mt-2 space-y-1 ml-2">
					<li>Use the service to attack, disrupt, or perform unauthorized monitoring of systems you do not own or have permission to monitor</li>
					<li>Reverse engineer the service for malicious purposes</li>
					<li>Attempt to gain unauthorized access to other users' accounts or data</li>
					<li>Use the service in violation of any applicable law</li>
				</ul>
			</section>

			<section>
				<h2 class="text-base font-semibold text-foreground mb-2">8. Data</h2>
				<p>Your use of WatchDog is also governed by our <a href="/privacy" class="text-accent hover:underline">Privacy Policy</a>, which describes how we collect, use, and protect your data.</p>
			</section>

			<section>
				<h2 class="text-base font-semibold text-foreground mb-2">9. Open Source License</h2>
				<p>The WatchDog source code is separately licensed under the <strong class="text-foreground">GNU Affero General Public License v3.0 (AGPL-3.0)</strong>. These Terms of Service govern your use of the hosted service at usewatchdog.dev and are separate from the open-source license.</p>
			</section>

			<section>
				<h2 class="text-base font-semibold text-foreground mb-2">10. Governing Law</h2>
				<p>These terms are governed by the laws of the <strong class="text-foreground">Province of Ontario, Canada</strong>. Any disputes arising from these terms shall be resolved in the courts of Ontario, Canada. Service infrastructure is hosted in the United Kingdom.</p>
			</section>

			<section>
				<h2 class="text-base font-semibold text-foreground mb-2">11. Contact</h2>
				<p>For questions about these terms, contact us at <a href="mailto:techwithsyl@gmail.com" class="text-accent hover:underline">techwithsyl@gmail.com</a>.</p>
			</section>

		</div>

		<div class="mt-10 pt-6 border-t border-border text-center">
			<p class="text-[11px] text-muted-foreground/60">WatchDog &mdash; Open-source infrastructure monitoring</p>
		</div>
	</div>
</div>
