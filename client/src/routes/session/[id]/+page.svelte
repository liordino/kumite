<script lang="ts">
	import { api, streamRun, type AgentNode, type RunEvent, type Session } from '$lib/api';
	import { expandedThinking, toggleThinking } from '$lib/stores';
	import AgentCard from '$lib/components/AgentCard.svelte';
	import IntakeChoice from '$lib/components/IntakeChoice.svelte';
	import PipelineBuilder from '$lib/components/PipelineBuilder.svelte';
	import PsdPanel from '$lib/components/PsdPanel.svelte';
	import ThinkingTrace from '$lib/components/ThinkingTrace.svelte';

	let { params }: { params: { id: string } } = $props();

	let session = $state<Session | null>(null);
	let loadError = $state('');
	let busy = $state(false);
	let runError = $state('');

	// Live-run view state (a run outlives the client; this is only the tail).
	let liveNodes = $state<AgentNode[]>([]);
	let liveThinking = $state<Record<string, string>>({});
	let livePsd = $state('');
	let streaming = $state(false);
	let handoffBundle = $state<{ brief_md: string; context_md: string } | null>(null);
	let handoffBusy = $state(false);

	let attachSource: EventSource | null = null;

	const phase = $derived(session?.phase ?? null);

	$effect(() => {
		const id = params.id;
		reload(id);
		return () => attachSource?.close();
	});

	async function reload(rid = params.id) {
		try {
			session = await api.getSession(rid);
		} catch (e) {
			loadError = String(e);
		}
	}

	function handleRunEvent(ev: RunEvent) {
		const d = ev.data;
		switch (ev.event) {
			case 'pipeline_start':
				streaming = true;
				break;
			case 'agent_start':
				liveNodes = liveNodes.map((n) =>
					n.agent_id === d.agent_id ? { ...n, status: 'running' as const } : n
				);
				break;
			case 'agent_thinking':
				liveThinking = {
					...liveThinking,
					[String(d.agent_id)]: (liveThinking[String(d.agent_id)] ?? '') + String(d.chunk)
				};
				break;
			case 'agent_done':
				liveNodes = liveNodes.map((n) =>
					n.agent_id === d.agent_id ? { ...n, status: 'done' as const } : n
				);
				break;
			case 'agent_error':
				liveNodes = liveNodes.map((n) =>
					n.agent_id === d.agent_id ? { ...n, status: 'error' as const } : n
				);
				break;
			case 'psd_chunk':
				livePsd += String(d.chunk);
				break;
			case 'psd_done':
				livePsd = String(d.psd);
				break;
			case 'pipeline_complete':
				streaming = false;
				void reload();
				break;
			case 'pipeline_error':
				streaming = false;
				runError = String(d.error);
				void reload();
				break;
		}
	}

	async function generateHandoff() {
		if (!session) return;
		handoffBusy = true;
		runError = '';
		try {
			handoffBundle = await api.handoff(session.id);
		} catch (e) {
			runError = String(e);
		} finally {
			handoffBusy = false;
		}
	}

	async function runPhase0() {
		if (!session) return;
		busy = true;
		runError = '';
		try {
			await api.phase0(session.id);
			await reload();
		} catch (e) {
			runError = String(e);
		} finally {
			busy = false;
		}
	}

	async function savePlan(plan: Session['pipeline_plan']) {
		if (!session || !plan) return;
		await api.updatePlan(session.id, plan);
		await reload();
	}

	async function startRun(resume = false) {
		if (!session) return;
		busy = true;
		runError = '';
		liveNodes = allNodes(session);
		livePsd = '';
		liveThinking = {};
		try {
			await streamRun(resume ? api.resume(session.id) : api.run(session.id), handleRunEvent);
		} catch (e) {
			runError = String(e);
		} finally {
			busy = false;
			streaming = false;
			await reload();
		}
	}

	// Attach to a run already in flight: the run outlives the client. Events
	// are not replayed — the session above is the record.
	$effect(() => {
		if ((phase === 'running' || phase === 'synthesis') && !attachSource && session) {
			liveNodes = allNodes(session);
			const es = new EventSource(`/api/pipeline/stream/${session.id}`);
			const close = () => es.close();
			for (const name of [
				'agent_start',
				'agent_done',
				'agent_error',
				'agent_thinking',
				'psd_chunk',
				'psd_done',
				'pipeline_complete',
				'pipeline_error'
			]) {
				es.addEventListener(name, (e) => {
					const data = JSON.parse((e as MessageEvent).data) as Record<string, unknown>;
					handleRunEvent({ event: name, data });
					if (name === 'pipeline_complete' || name === 'pipeline_error') close();
				});
			}
			es.onerror = () => {
				// The endpoint returns 409 once the run ended; stop reconnecting.
				if (es.readyState === EventSource.CLOSED) close();
			};
			attachSource = es;
		}
	});

	function allNodes(s: Session): AgentNode[] {
		const p = s.pipeline_plan?.pipeline;
		if (!p) return [];
		return [...p.wave1, ...p.wave2, ...p.fixed];
	}

	const phaseText: Record<string, string> = {
		intake: 'Intake — decide how the panel reads your material.',
		distilling: 'Distilling…',
		pipeline_review: 'Review the proposed panel, then run it.',
		running: 'Panel running — results appear as each specialist completes.',
		synthesis: 'Shishō is synthesizing the PSD…',
		interrupted: 'Run interrupted. Completed results are preserved; resume from the first incomplete agent.',
		complete: 'Feasibility verdict complete.',
		error: 'Something failed with no recovery path.'
	};
</script>

<div class="mx-auto max-w-4xl space-y-6">
	<p class="text-xs text-zinc-500">
		<a class="underline hover:text-zinc-900" href="/">← All sessions</a>
	</p>

	{#if loadError}
		<p class="rounded border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800">{loadError}</p>
	{:else if !session}
		<p class="text-sm text-zinc-500">Loading…</p>
	{:else}
		<header class="rounded border border-zinc-300 bg-white px-4 py-3">
			<div class="flex flex-wrap items-baseline justify-between gap-2">
				<h1 class="text-lg font-semibold text-zinc-900">{session.project_name}</h1>
				<span class="text-xs text-zinc-500">{session.phase}</span>
			</div>
			<p class="mt-1 text-sm text-zinc-600">{phaseText[session.phase]}</p>
			{#if runError}
				<p class="mt-2 rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800">
					{runError}
				</p>
			{/if}
			{#if phase === 'interrupted'}
				<button
					type="button"
					class="mt-3 rounded bg-zinc-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-zinc-700"
					disabled={busy}
					onclick={() => startRun(true)}
				>
					Resume run
				</button>
			{/if}
		</header>

		{#if session.phase === 'intake'}
			<IntakeChoice
				sessionId={session.id}
				rawInput={session.raw_input}
				rawSource={session.raw_source}
				onChanged={() => reload()}
			/>
			<section class="flex justify-end">
				<button
					type="button"
					class="rounded bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-700 disabled:opacity-50"
					disabled={busy}
					onclick={runPhase0}
				>
					Run Phase 0 — classify and propose panel
				</button>
			</section>
		{:else if session.phase === 'pipeline_review' && session.pipeline_plan}
			{#key session.pipeline_plan}
				<PipelineBuilder
					plan={session.pipeline_plan}
					busy={busy}
					onSave={savePlan}
					onStart={() => startRun(false)}
				/>
			{/key}
		{:else if (session.phase === 'running' || session.phase === 'synthesis') && session.pipeline_plan}
			<section class="space-y-3">
				<h2 class="text-sm font-semibold text-zinc-900">Panel execution</h2>
				{#each liveNodes as node (node.id)}
					<div class="rounded border border-zinc-300 bg-white px-4 py-2">
						<div class="flex items-center justify-between">
							<span class="text-sm font-medium text-zinc-900">{node.display_name}</span>
							<span class="text-xs text-zinc-500">
								{node.status === 'running' ? 'running…' : node.status}
							</span>
						</div>
						{#if node.status === 'running' && liveThinking[node.agent_id]}
							<ThinkingTrace
								agentId={node.agent_id}
								thinking={liveThinking[node.agent_id]}
								expanded={$expandedThinking.has(node.agent_id)}
								onToggle={toggleThinking}
							/>
						{/if}
					</div>
				{/each}
				{#if livePsd}
					<PsdPanel psd={livePsd} />
				{/if}
			</section>
		{:else if session.phase === 'complete'}
			{#if session.psd}
				<PsdPanel psd={session.psd} />
				<section class="rounded border border-zinc-300 bg-white px-4 py-3">
					<div class="flex flex-wrap items-center justify-between gap-3">
						<div>
							<h2 class="text-sm font-semibold text-zinc-900">Dojo handoff bundle</h2>
							<p class="mt-1 text-sm text-zinc-600">
								Reframe the verdict as a Dojo feasibility brief plus a CONTEXT.md seed —
								ready for Dojo&apos;s /hajime.
							</p>
						</div>
						<button
							type="button"
							class="rounded border border-zinc-400 px-3 py-1.5 text-sm font-medium text-zinc-800 hover:bg-zinc-100 disabled:opacity-50"
							disabled={handoffBusy}
							onclick={generateHandoff}
						>
							{handoffBusy ? 'Generating…' : 'Generate bundle'}
						</button>
					</div>
					{#if handoffBundle}
						<div class="mt-3 space-y-3">
							<div>
								<p class="mb-1 text-xs font-medium tracking-wide text-zinc-500 uppercase">BRIEF.md</p>
								<pre class="max-h-96 overflow-auto whitespace-pre-wrap rounded bg-zinc-50 p-3 text-xs leading-relaxed text-zinc-700">{handoffBundle.brief_md}</pre>
							</div>
							<div>
								<p class="mb-1 text-xs font-medium tracking-wide text-zinc-500 uppercase">CONTEXT.md seed</p>
								<pre class="max-h-96 overflow-auto whitespace-pre-wrap rounded bg-zinc-50 p-3 text-xs leading-relaxed text-zinc-700">{handoffBundle.context_md}</pre>
							</div>
						</div>
					{/if}
				</section>
			{/if}
			<section class="space-y-3">
				<h2 class="text-sm font-semibold text-zinc-900">Specialist outputs</h2>
				{#each session.agent_outputs as output (output.agent_id)}
					<AgentCard
						output={output}
						expanded={$expandedThinking.has(output.agent_id)}
						onToggle={toggleThinking}
					/>
				{/each}
			</section>
		{:else if session.phase === 'interrupted' && session.pipeline_plan}
			<section class="space-y-3">
				<h2 class="text-sm font-semibold text-zinc-900">Preserved results</h2>
				{#each session.agent_outputs as output (output.agent_id)}
					<AgentCard
						output={output}
						expanded={$expandedThinking.has(output.agent_id)}
						onToggle={toggleThinking}
					/>
				{/each}
			</section>
		{/if}
	{/if}
</div>