<script lang="ts">
	import { api, type AgentNode, type PipelinePlan, type RepoAgent } from '$lib/api';

	let {
		plan,
		busy = false,
		onSave,
		onStart
	}: {
		plan: PipelinePlan;
		busy?: boolean;
		onSave: (plan: PipelinePlan) => void;
		onStart: () => void;
	} = $props();

	// Local editable copy — nothing is sent until Save. No pipeline logic
	// lives here: ordering validation is server-side (PATCH /plan).
	//
	// The init-capture is intentional: the parent remounts this component
	// via {#key session.pipeline_plan} whenever the persisted plan changes,
	// so capturing the prop's initial value is the contract.
	// svelte-ignore state_referenced_locally
	let working: PipelinePlan = $state($state.snapshot(plan));
	let dirty = $derived(JSON.stringify(working) !== JSON.stringify(plan));
	let error = $state('');
	let repoAgents = $state<RepoAgent[] | null>(null);
	let repoError = $state('');
	let showRepo = $state(false);

	async function toggleRepo() {
		showRepo = !showRepo;
		if (showRepo && repoAgents === null) {
			try {
				repoAgents = await api.repoListing();
			} catch (e) {
				repoError = String(e);
			}
		}
	}

	function addRepoAgent(a: RepoAgent, wave: 1 | 2) {
		const node: AgentNode = {
			id: a.agent_id,
			agent_id: a.agent_id,
			display_name: a.display_name,
			role_summary: '',
			source_url: a.source_url,
			wave,
			status: 'pending',
			enabled: true,
			rationale: 'Added from the full repo.'
		};
		if (wave === 1) working.pipeline.wave1.push(node);
		else working.pipeline.wave2.push(node);
		working = { ...working };
	}

	const allIds = $derived(new Set(working.pipeline.wave1.concat(working.pipeline.wave2).map((n) => n.agent_id)));
	const repoAvailable = $derived(
		(repoAgents ?? []).filter((a) => !allIds.has(a.agent_id))
	);

	const fourBlock = $derived([
		{ label: 'What is wanted', value: working.context.four_block.what_is_wanted },
		{ label: 'How it should be done', value: working.context.four_block.how_it_should_be_done },
		{ label: 'What is NOT wanted', value: working.context.four_block.what_is_not_wanted },
		{ label: 'How success is measured', value: working.context.four_block.how_success_is_measured }
	]);

	function toggle(node: AgentNode) {
		node.enabled = !node.enabled;
		working = { ...working };
	}
	function moveUp(node: AgentNode, list: AgentNode[], index: number) {
		if (index > 0) {
			list.splice(index, 1);
			list.splice(index - 1, 0, node);
			working = { ...working };
		}
	}
	function moveBetween(node: AgentNode, from: AgentNode[], to: AgentNode[]) {
		const i = from.indexOf(node);
		if (i >= 0) {
			from.splice(i, 1);
			node.wave = to === working.pipeline.wave1 ? 1 : 2;
			to.push(node);
			working = { ...working };
		}
	}

	async function save() {
		error = '';
		try {
			await onSave(working);
		} catch (e) {
			error = String(e);
		}
	}
</script>

<section class="rounded border border-zinc-300 bg-white">
	<header class="flex flex-wrap items-center justify-between gap-3 border-b border-zinc-300 px-4 py-3">
		<div>
			<h2 class="text-sm font-semibold text-zinc-900">Proposed panel</h2>
			<p class="mt-0.5 text-xs text-zinc-500">
				{working.project_name} · {working.context.domain} · input: {working.input_type} ·
				maturity: {working.context.maturity}
				{working.context.distilled ? '· distilled input' : ''}
			</p>
		</div>
		<div class="flex items-center gap-3">
			<button
				type="button"
				class="rounded border border-zinc-400 px-3 py-1.5 text-sm font-medium text-zinc-800 hover:bg-zinc-100 disabled:opacity-50"
				disabled={!dirty || busy}
				onclick={save}
			>
				Save changes
			</button>
			<button
				type="button"
				class="rounded bg-zinc-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-zinc-700 disabled:opacity-50"
				disabled={busy}
				onclick={onStart}
			>
				Run panel
			</button>
		</div>
	</header>

	{#if error}
		<p class="border-b border-red-200 bg-red-50 px-4 py-2 text-sm text-red-800">{error}</p>
	{/if}
	{#if dirty}
		<p class="border-b border-amber-200 bg-amber-50 px-4 py-2 text-sm text-amber-900">
			Unsaved changes — save before running.
		</p>
	{/if}

	<div class="border-b border-zinc-200 px-4 py-3">
		<h3 class="text-xs font-medium tracking-wide text-zinc-500 uppercase">Classification</h3>
		<p class="mt-1 text-sm text-zinc-700">{working.context.problem_statement}</p>
		<dl class="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
			{#each fourBlock as block (block.label)}
				<div class="rounded border border-zinc-200 px-2 py-1.5">
					<dt class="text-xs text-zinc-500">{block.label}</dt>
					<dd class="text-sm font-medium text-zinc-800">{block.value}</dd>
				</div>
			{/each}
		</dl>
		{#if working.context.flags.length > 0}
			<ul class="mt-3 space-y-1">
				{#each working.context.flags as flag (flag)}
					<li class="border-l-2 border-amber-400 pl-2 text-sm text-zinc-700">{flag}</li>
				{/each}
			</ul>
		{/if}
	</div>

	{#each [{ title: 'Wave 1 — independent analysis', nodes: working.pipeline.wave1, other: working.pipeline.wave2 }, { title: 'Wave 2 — reactive analysis', nodes: working.pipeline.wave2, other: working.pipeline.wave1 }] as wave (wave.title)}
		<div class="border-b border-zinc-200 px-4 py-3">
			<h3 class="text-xs font-medium tracking-wide text-zinc-500 uppercase">{wave.title}</h3>
			{#if wave.nodes.length === 0}
				<p class="mt-2 text-sm text-zinc-500">No agents in this wave.</p>
			{/if}
			<ul class="mt-2 space-y-2">
				{#each wave.nodes as node, i (node.id)}
					<li class="flex items-start justify-between gap-3 rounded border border-zinc-200 px-3 py-2">
						<label class="flex items-start gap-2">
							<input type="checkbox" class="mt-1" checked={node.enabled} onchange={() => toggle(node)} />
							<span>
								<span class="block text-sm font-medium text-zinc-900">{node.display_name}</span>
								<span class="block text-xs text-zinc-500">{node.rationale}</span>
							</span>
						</label>
						<span class="flex shrink-0 items-center gap-1 text-xs">
							{#if i > 0}
								<button type="button" class="rounded border border-zinc-300 px-1.5 py-0.5 hover:bg-zinc-100" disabled={busy} onclick={() => moveUp(node, wave.nodes, i)}>↑</button>
							{/if}
							<button
								type="button"
								class="rounded border border-zinc-300 px-1.5 py-0.5 hover:bg-zinc-100"
								disabled={busy}
								onclick={() => moveBetween(node, wave.nodes, wave.other)}
							>
								→ {wave.other === working.pipeline.wave1 ? 'W1' : 'W2'}
							</button>
						</span>
					</li>
				{/each}
			</ul>
		</div>
	{/each}

	<div class="border-b border-zinc-200 px-4 py-3">
		<button
			type="button"
			class="text-xs font-medium text-zinc-500 underline hover:text-zinc-900"
			onclick={toggleRepo}
		>
			{showRepo ? 'Hide' : 'Add from'} the full repo (advanced)
		</button>
		{#if showRepo}
			{#if repoError}
				<p class="mt-2 text-sm text-red-800">{repoError}</p>
			{:else if repoAgents === null}
				<p class="mt-2 text-sm text-zinc-500">Loading repo…</p>
			{:else if repoAvailable.length === 0}
				<p class="mt-2 text-sm text-zinc-500">Every repo agent is already in the panel.</p>
			{:else}
				<ul class="mt-2 max-h-64 space-y-1 overflow-auto">
					{#each repoAvailable as a (a.agent_id)}
						<li class="flex items-center justify-between gap-2 rounded border border-zinc-200 px-2 py-1">
							<span class="text-sm text-zinc-800">{a.display_name}</span>
							<span class="flex gap-1 text-xs">
								<button type="button" class="rounded border border-zinc-300 px-1.5 py-0.5 hover:bg-zinc-100" onclick={() => addRepoAgent(a, 1)}>→ W1</button>
								<button type="button" class="rounded border border-zinc-300 px-1.5 py-0.5 hover:bg-zinc-100" onclick={() => addRepoAgent(a, 2)}>→ W2</button>
							</span>
						</li>
					{/each}
				</ul>
			{/if}
		{/if}
	</div>

	<div class="px-4 py-3">
		<h3 class="text-xs font-medium tracking-wide text-zinc-500 uppercase">
			Fixed — challenges every finding (cannot be changed)
		</h3>
		<ul class="mt-2 space-y-2">
			{#each working.pipeline.fixed as node (node.id)}
				<li class="flex items-start justify-between gap-3 rounded border border-zinc-300 bg-zinc-50 px-3 py-2">
					<span>
						<span class="block text-sm font-medium text-zinc-900">{node.display_name}</span>
						<span class="block text-xs text-zinc-500">{node.rationale}</span>
					</span>
					<span class="shrink-0 text-xs text-zinc-500">always last</span>
				</li>
			{/each}
		</ul>
	</div>
</section>