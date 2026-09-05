<script lang="ts">
	import { api, type CustomAgent } from '$lib/api';

	let endpoint = $state('');
	let model = $state('');
	let apiKey = $state('');
	let customAgents = $state<CustomAgent[]>([]);
	let busy = $state(false);
	let saved = $state(false);
	let error = $state('');

	let newAgent = $state({ display_name: '', role_summary: '', wave_preference: 1, system_prompt: '' });
	let agentError = $state('');

	$effect(() => {
		void load();
	});

	async function load() {
		try {
			const cfg = await api.getConfig();
			endpoint = cfg['llm_endpoint'] ?? '';
			model = cfg['llm_model'] ?? '';
			apiKey = ''; // stored masked; leave blank to keep the current key
			customAgents = await api.listCustomAgents();
		} catch (e) {
			error = String(e);
		}
	}

	async function save() {
		busy = true;
		saved = false;
		error = '';
		try {
			const values: Record<string, string> = { llm_endpoint: endpoint, llm_model: model };
			if (apiKey.trim() !== '') values['llm_api_key'] = apiKey;
			await api.putConfig(values);
			saved = true;
			apiKey = '';
		} catch (e) {
			error = String(e);
		} finally {
			busy = false;
		}
	}

	async function createAgent() {
		agentError = '';
		if (!newAgent.display_name.trim() || !newAgent.system_prompt.trim()) {
			agentError = 'Name and system prompt are required.';
			return;
		}
		try {
			await api.createCustomAgent(newAgent);
			newAgent = { display_name: '', role_summary: '', wave_preference: 1, system_prompt: '' };
			customAgents = await api.listCustomAgents();
		} catch (e) {
			agentError = String(e);
		}
	}

	async function removeAgent(id: string) {
		try {
			await api.deleteCustomAgent(id);
			customAgents = await api.listCustomAgents();
		} catch (e) {
			agentError = String(e);
		}
	}
</script>

<div class="mx-auto max-w-3xl space-y-6">
	<section class="rounded border border-zinc-300 bg-white">
		<header class="border-b border-zinc-300 px-4 py-3">
			<h2 class="text-sm font-semibold text-zinc-900">Inference provider</h2>
			<p class="mt-1 text-sm text-zinc-600">
				Any OpenAI-compatible endpoint works. Changes take effect on the next LLM call — no restart.
			</p>
		</header>
		<form
			class="space-y-3 px-4 py-4"
			onsubmit={(e) => {
				e.preventDefault();
				void save();
			}}
		>
			<div>
				<label class="block text-sm font-medium text-zinc-800" for="llm-endpoint">Endpoint</label>
				<input
					id="llm-endpoint"
					type="text"
					class="mt-1 w-full rounded border border-zinc-400 px-3 py-2 text-sm focus:border-zinc-900 focus:outline-none"
					bind:value={endpoint}
				/>
			</div>
			<div>
				<label class="block text-sm font-medium text-zinc-800" for="llm-model">Model</label>
				<input
					id="llm-model"
					type="text"
					class="mt-1 w-full rounded border border-zinc-400 px-3 py-2 text-sm focus:border-zinc-900 focus:outline-none"
					bind:value={model}
				/>
			</div>
			<div>
				<label class="block text-sm font-medium text-zinc-800" for="llm-api-key">API key</label>
				<input
					id="llm-api-key"
					type="password"
					class="mt-1 w-full rounded border border-zinc-400 px-3 py-2 text-sm focus:border-zinc-900 focus:outline-none"
					bind:value={apiKey}
					placeholder="Leave blank to keep the current key"
				/>
			</div>
			{#if error}
				<p class="text-sm text-red-800">{error}</p>
			{:else if saved}
				<p class="text-sm text-zinc-600">Saved.</p>
			{/if}
			<button
				type="submit"
				class="rounded bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-700 disabled:opacity-50"
				disabled={busy}
			>
				Save provider settings
			</button>
		</form>
	</section>

	<section class="rounded border border-zinc-300 bg-white">
		<header class="border-b border-zinc-300 px-4 py-3">
			<h2 class="text-sm font-semibold text-zinc-900">Custom agents</h2>
			<p class="mt-1 text-sm text-zinc-600">
				User-defined specialists. Shishō can select them like built-in agents, and you can add them
				to a panel in the builder.
			</p>
		</header>

		{#if customAgents.length > 0}
			<ul class="divide-y divide-zinc-200">
				{#each customAgents as a (a.id)}
					<li class="flex items-start justify-between gap-3 px-4 py-3">
						<span>
							<span class="block text-sm font-medium text-zinc-900">{a.display_name}</span>
							<span class="block text-xs text-zinc-500">{a.role_summary}</span>
						</span>
						<button
							type="button"
							class="text-xs text-zinc-500 underline hover:text-zinc-900"
							onclick={() => removeAgent(a.id)}
						>
							delete
						</button>
					</li>
				{/each}
			</ul>
		{:else}
			<p class="px-4 py-3 text-sm text-zinc-500">No custom agents defined.</p>
		{/if}

		<form
			class="space-y-3 border-t border-zinc-300 px-4 py-4"
			onsubmit={(e) => {
				e.preventDefault();
				void createAgent();
			}}
		>
			<h3 class="text-sm font-medium text-zinc-800">Define a new agent</h3>
			<div class="grid gap-3 sm:grid-cols-2">
				<div>
					<label class="block text-sm font-medium text-zinc-800" for="agent-name">Name</label>
					<input
						id="agent-name"
						type="text"
						class="mt-1 w-full rounded border border-zinc-400 px-3 py-2 text-sm focus:border-zinc-900 focus:outline-none"
						bind:value={newAgent.display_name}
					/>
				</div>
				<div>
					<label class="block text-sm font-medium text-zinc-800" for="agent-wave">Wave preference</label>
					<select
						id="agent-wave"
						class="mt-1 w-full rounded border border-zinc-400 px-3 py-2 text-sm focus:border-zinc-900 focus:outline-none"
						bind:value={newAgent.wave_preference}
					>
						<option value={1}>Wave 1 — independent</option>
						<option value={2}>Wave 2 — reactive</option>
					</select>
				</div>
			</div>
			<div>
				<label class="block text-sm font-medium text-zinc-800" for="agent-role">Role summary</label>
				<input
					id="agent-role"
					type="text"
					class="mt-1 w-full rounded border border-zinc-400 px-3 py-2 text-sm focus:border-zinc-900 focus:outline-none"
					bind:value={newAgent.role_summary}
				/>
			</div>
			<div>
				<label class="block text-sm font-medium text-zinc-800" for="agent-prompt">System prompt</label>
				<textarea
					id="agent-prompt"
					class="mt-1 h-28 w-full rounded border border-zinc-400 px-3 py-2 font-mono text-sm focus:border-zinc-900 focus:outline-none"
					bind:value={newAgent.system_prompt}
				></textarea>
			</div>
			{#if agentError}
				<p class="text-sm text-red-800">{agentError}</p>
			{/if}
			<button
				type="submit"
				class="rounded border border-zinc-400 px-3 py-1.5 text-sm font-medium text-zinc-800 hover:bg-zinc-100"
			>
				Add agent
			</button>
		</form>
	</section>
</div>