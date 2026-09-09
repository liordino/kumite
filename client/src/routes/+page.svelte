<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, type SessionListItem } from '$lib/api';
	import SessionList from '$lib/components/SessionList.svelte';

	let projectName = $state('');
	let rawInput = $state('');
	let sessions = $state<SessionListItem[]>([]);
	let busy = $state(false);
	let error = $state('');
	let loaded = $state(false);

	async function load() {
		try {
			sessions = await api.listSessions();
		} catch (e) {
			error = String(e);
		} finally {
			loaded = true;
		}
	}

	$effect(() => {
		load();
	});

	async function create() {
		if (!rawInput.trim()) {
			error = 'Describe the project idea first.';
			return;
		}
		busy = true;
		error = '';
		try {
			const { id } = await api.createSession(projectName.trim(), rawInput);
			await goto(`/session/${id}`);
		} catch (e) {
			error = String(e);
			busy = false;
		}
	}

	async function remove(id: string) {
		error = '';
		try {
			await api.deleteSession(id);
			await load();
		} catch (e) {
			error = String(e);
		}
	}
</script>

<div class="mx-auto max-w-3xl space-y-6">
	<section class="rounded border border-stone-300 bg-white">
		<header class="border-b border-stone-300 px-4 py-3">
			<h2 class="text-sm font-semibold text-stone-900">New evaluation</h2>
			<p class="mt-1 text-sm text-stone-500">
				Submit a project idea — a one-line concept, a written brief, a transcript of conversations
				with AI models, or an existing PSD. The panel decides whether it is worth building.
			</p>
		</header>
		<form
			class="space-y-3 px-4 py-4"
			onsubmit={(e) => {
				e.preventDefault();
				void create();
			}}
		>
			<div>
				<label class="block text-sm font-medium text-stone-800" for="project-name">
					Project name <span class="font-normal text-zinc-500">(optional — a name is derived if empty)</span>
				</label>
				<input
					id="project-name"
					type="text"
					class="mt-1 w-full rounded border border-stone-300 px-3 py-2 text-sm focus:border-stone-300 focus:outline-none"
					bind:value={projectName}
					placeholder="Field Ledger"
				/>
			</div>
			<div>
				<label class="block text-sm font-medium text-stone-800" for="raw-input">Project material</label>
				<textarea
					id="raw-input"
					class="mt-1 h-40 w-full rounded border border-stone-300 px-3 py-2 font-mono text-sm focus:border-stone-300 focus:outline-none"
					bind:value={rawInput}
					placeholder="Paste the brief, transcript, or concept here…"
				></textarea>
			</div>
			{#if error}
				<p class="text-sm text-red-700">{error}</p>
			{/if}
			<button
				type="submit"
				class="rounded bg-stone-900 px-4 py-2 text-sm font-medium text-stone-50 hover:bg-stone-700 disabled:opacity-50"
				disabled={busy}
			>
				Create session
			</button>
		</form>
	</section>

	<section class="space-y-3">
		<h2 class="text-sm font-semibold text-stone-900">Sessions</h2>
		{#if loaded}
			<SessionList sessions={sessions} onDelete={remove} />
		{:else}
			<p class="text-sm text-zinc-500">Loading…</p>
		{/if}
	</section>
</div>