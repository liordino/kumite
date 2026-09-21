<script lang="ts">
	import {
		api,
		streamRun,
		type AgentNode,
		type RunEvent,
		type Session,
		type SessionPhase
	} from '$lib/api';
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
	let pauseNotice = $state('');

	// Live-run view state (a run outlives the client; this is only the tail).
	let liveNodes = $state<AgentNode[]>([]);
	let liveThinking = $state<Record<string, string>>({});
	let livePsd = $state('');
	// The client-side phase during a triggered run: the persisted session
	// phase only updates on completion, so the monitor branch keys off this.
	let livePhase = $state<SessionPhase | ''>('');
	let handoffBundle = $state<{ brief_md: string; context_md: string } | null>(null);
	let handoffBusy = $state(false);

	let attachSource: EventSource | null = null;

	// Display phase: a triggered run drives this live (the persisted session
	// phase only updates on completion, via reload).
	const phase = $derived<SessionPhase>(livePhase || (session?.phase ?? 'intake'));

	const nowRunning = $derived(liveNodes.find((n) => n.status === 'running') ?? null);

	const waveGroups = $derived([
		{
			title: 'Wave 1 — independent analysis',
			nodes: liveNodes.filter((n) => n.wave === 1)
		},
		{
			title: 'Wave 2 — reactive analysis',
			nodes: liveNodes.filter((n) => n.wave === 2)
		},
		{
			title: 'Fixed — Reality Checker',
			nodes: liveNodes.filter((n) => n.wave === 3)
		}
	]);

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
			case 'synthesis_start':
				livePhase = 'synthesis';
				break;
			case 'psd_chunk':
				livePsd += String(d.chunk);
				break;
			case 'psd_done':
				livePsd = String(d.psd);
				break;
			case 'run_paused':
				livePhase = '';
				pauseNotice = 'Run paused — ' + String(d.agent_id) + ' failed: ' + String(d.error) + '. Resume to retry it.';
				void reload();
				break;
			case 'pipeline_complete':
				livePhase = '';
				void reload();
				break;
			case 'pipeline_error':
				runError = String(d.error);
				livePhase = '';
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

	async function startRun(resume = false, pauseOnFail = false) {
		if (!session) return;
		busy = true;
		runError = '';
		pauseNotice = '';
		liveNodes = allNodes(session);
		livePsd = '';
		liveThinking = {};
		livePhase = 'running';
		try {
			await streamRun(
				resume
					? api.resume(session.id)
					: api.run(session.id, { pause_on_fail: pauseOnFail }),
				handleRunEvent
			);
		} catch (e) {
			runError = String(e);
		} finally {
			busy = false;
			await reload();
		}
	}

	// Attach to a run already in flight: the run outlives the client. Events
	// are not replayed — the session above is the record.
	$effect(() => {
		if ((phase === 'running' || phase === 'synthesis') && !attachSource && session) {
			livePhase = session.phase;
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
	{#snippet statusIcon(status: AgentNode['status'])}
		{#if status === 'running'}
			<span class="inline-block h-3 w-3 shrink-0 animate-spin rounded-full border-2 border-stone-300 border-t-stone-400"></span>
		{:else if status === 'done'}
			<span class="shrink-0 text-sm font-medium text-green-700">✓</span>
		{:else if status === 'error'}
			<span class="shrink-0 text-sm font-medium text-red-700">✗</span>
		{:else if status === 'skipped'}
			<span class="shrink-0 text-sm text-stone-500">—</span>
		{:else}
			<span class="inline-block h-2.5 w-2.5 shrink-0 rounded-full border-2 border-stone-300"></span>
		{/if}
	{/snippet}

	{#snippet statusText(status: AgentNode['status'])}
		{#if status === 'running'}
			<span class="text-stone-900">analyzing…</span>
		{:else if status === 'done'}
			<span class="text-zinc-500">done</span>
		{:else if status === 'error'}
			<span class="text-red-700">failed — the run continued</span>
		{:else if status === 'skipped'}
			<span class="text-stone-500">skipped</span>
		{:else}
			<span class="text-stone-500">pending</span>
		{/if}
	{/snippet}

	<p class="text-xs text-stone-500 tabular">
		<a class="underline hover:text-stone-900" href="/">← All sessions</a>
	</p>

	{#if loadError}
		<p class="rounded border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-700">{loadError}</p>
	{:else if !session}
		<p class="text-sm text-zinc-500">Loading…</p>
	{:else}
		<header class="rule-division rounded-none border border-stone-300 bg-white px-4 py-3">
			<div class="flex flex-wrap items-baseline justify-between gap-2">
				<h1 class="display text-xl font-bold text-stone-900">{session.project_name}</h1>
				<span class="display text-xs uppercase tracking-widest text-stone-500">{phase}</span>
			</div>
			<p class="mt-1 text-sm text-stone-500">{phaseText[phase]}</p>
			{#if nowRunning}
				<p class="mt-2 flex items-center gap-2 text-sm font-medium text-stone-900">
					<span class="bout-live inline-block h-3 w-3 animate-spin rounded-full border-2 border-stone-300"></span>
					Now analyzing: {nowRunning.display_name}
				</p>
			{/if}
			{#if runError}
				<p class="mt-2 rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-700">
					{runError}
				</p>
			{/if}
			{#if pauseNotice}
				<p class="mt-2 rounded border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-900">
					{pauseNotice}
				</p>
			{/if}
			{#if phase === 'interrupted'}
				<button
					type="button"
					class="display mt-3 rounded-none bg-orange-700 px-3 py-1.5 text-sm font-bold uppercase tracking-wider text-orange-50 hover:bg-orange-800"
					disabled={busy}
					onclick={() => startRun(true)}
				>
					Resume run
				</button>
			{/if}
		</header>

		{#if phase === 'intake'}
			<IntakeChoice
				sessionId={session.id}
				rawInput={session.raw_input}
				rawSource={session.raw_source}
				onChanged={() => reload()}
			/>
			<section class="flex flex-wrap items-center justify-end gap-3">
				{#if busy}
					<span class="display mr-auto flex items-center gap-2 text-sm text-stone-500 uppercase text-xs tracking-wide">
						<span class="inline-block h-3 w-3 animate-spin rounded-full border-2 border-stone-300 border-t-stone-400"></span>
						Shishō is classifying the input and proposing the panel — a real model call, this can take a minute.
					</span>
				{/if}
				<button
					type="button"
					class="display flex items-center gap-2 rounded-none bg-stone-900 px-4 py-2 text-sm font-bold uppercase tracking-wider text-stone-50 hover:bg-stone-700 disabled:opacity-50"
					disabled={busy}
					onclick={runPhase0}
				>
					{#if busy}
						<span class="inline-block h-3 w-3 animate-spin rounded-full border-2 border-stone-400 border-t-transparent"></span>
						Classifying…
					{:else}
						Run Phase 0 — classify and propose panel
					{/if}
				</button>
			</section>
		{:else if phase === 'pipeline_review' && session.pipeline_plan}
			{#key session.pipeline_plan}
				<PipelineBuilder
					plan={session.pipeline_plan}
					busy={busy}
					onSave={savePlan}
					onStart={(p: boolean) => {
						void startRun(false, p);
					}}
				/>
			{/key}
		{:else if (phase === 'running' || phase === 'synthesis') && session.pipeline_plan}
			<section class="space-y-4">
				{#each waveGroups as wave (wave.title)}
				<div class="rule-division border border-stone-300 bg-white">
					<h3 class="display border-b border-stone-300 px-4 py-2 text-xs font-bold uppercase tracking-widest text-stone-800">{wave.title}</h3>
						{#if wave.nodes.length === 0}
							<p class="px-4 py-3 text-sm text-zinc-500">No agents in this wave.</p>
						{:else}
							<ul class="divide-y divide-stone-200">
								{#each wave.nodes as node (node.id)}
									<li class="flex items-center gap-3 px-4 py-2">
										{@render statusIcon(node.status)}
										<span class="flex-1 text-sm {node.status === 'running' ? 'font-medium text-stone-900' : 'text-stone-700'}">{node.display_name}</span>
										{@render statusText(node.status)}
									</li>
									{#if node.status === 'running' && liveThinking[node.agent_id]}
										<li class="px-4 pb-2 pl-10">
											<ThinkingTrace
												agentId={node.agent_id}
												thinking={liveThinking[node.agent_id]}
												expanded={$expandedThinking.has(node.agent_id)}
												onToggle={toggleThinking}
											/>
										</li>
									{/if}
								{/each}
							</ul>
						{/if}
					</div>
				{/each}

				{#if phase === 'synthesis'}
					<div class="rounded border border-stone-300 bg-white px-4 py-3">
						<p class="flex items-center gap-2 text-sm text-stone-700">
							<span class="inline-block h-3 w-3 animate-spin rounded-full border-2 border-stone-300 border-t-stone-400"></span>
							Shishō is synthesizing the Project Summary Document…
						</p>
					</div>
					{#if livePsd}
						<PsdPanel psd={livePsd} />
					{/if}
				{/if}
			</section>
		{:else if phase === 'complete'}
			{#if session.psd}
				<PsdPanel psd={session.psd} />
				<section class="rounded border border-stone-300 bg-white px-4 py-3">
					<div class="flex flex-wrap items-center justify-between gap-3">
						<div>
							<h2 class="text-sm font-semibold text-stone-900">Dojo handoff bundle</h2>
							<p class="mt-1 text-sm text-stone-500">
								Reframe the verdict as a Dojo feasibility brief plus a CONTEXT.md seed —
								ready for Dojo&apos;s /hajime.
							</p>
						</div>
						<button
							type="button"
							class="display flex items-center gap-2 rounded-none border border-stone-300 px-3 py-1.5 text-sm font-semibold uppercase tracking-wider text-stone-800 hover:bg-stone-100 disabled:opacity-50"
							disabled={handoffBusy}
							onclick={generateHandoff}
						>
							{#if handoffBusy}
								<span class="inline-block h-3 w-3 animate-spin rounded-full border-2 border-stone-400 border-t-transparent"></span>
							{/if}
							{handoffBusy ? 'Generating…' : 'Generate bundle'}
						</button>
					</div>
					{#if handoffBundle}
						<div class="mt-3 space-y-3">
							<div>
								<p class="mb-1 text-xs font-medium tracking-wide text-zinc-500 uppercase">BRIEF.md</p>
								<pre class="max-h-96 overflow-auto whitespace-pre-wrap rounded bg-white p-3 text-xs leading-relaxed text-stone-700">{handoffBundle.brief_md}</pre>
							</div>
							<div>
								<p class="mb-1 text-xs font-medium tracking-wide text-zinc-500 uppercase">CONTEXT.md seed</p>
								<pre class="max-h-96 overflow-auto whitespace-pre-wrap rounded bg-white p-3 text-xs leading-relaxed text-stone-700">{handoffBundle.context_md}</pre>
							</div>
						</div>
					{/if}
				</section>
			{/if}
			<section class="space-y-3">
				<h2 class="text-sm font-semibold text-stone-900">Specialist outputs</h2>
				{#each session.agent_outputs as output (output.agent_id)}
					<AgentCard
						output={output}
						expanded={$expandedThinking.has(output.agent_id)}
						onToggle={toggleThinking}
					/>
				{/each}
			</section>
		{:else if phase === 'interrupted' && session.pipeline_plan}
			<section class="space-y-3">
				<h2 class="text-sm font-semibold text-stone-900">Preserved results</h2>
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