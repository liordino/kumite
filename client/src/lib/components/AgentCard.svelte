<script lang="ts">
	import type { AgentOutput } from '$lib/api';

	let {
		output,
		expanded,
		onToggle
	}: {
		output: AgentOutput;
		expanded: boolean;
		onToggle: (agentId: string) => void;
	} = $props();

	// Severity as a colored text marker — the word carries the meaning, the
	// color accelerates scanning. No chips, no backgrounds: scoresheet style.
	const severityClass: Record<string, string> = {
		critical: 'text-red-700',
		high: 'text-orange-700',
		medium: 'text-amber-600',
		low: 'text-stone-500'
	};

	const statusLabel: Record<string, string> = {
		done: 'done',
		partial: 'partial — degraded output, stored',
		error: 'failed — no contribution'
	};
</script>

<article class="border border-stone-300 bg-white">
	<header class="flex flex-wrap items-baseline justify-between gap-2 border-b border-stone-300 px-4 py-2">
		<h3 class="text-sm font-semibold text-stone-900">{output.display_name}</h3>
		<span class="text-[10px] font-semibold tracking-widest text-zinc-500 uppercase">{statusLabel[output.status] ?? output.status}</span>
	</header>

	<div class="px-4 py-3">
		<p class="text-sm leading-relaxed text-stone-800">{output.output.summary}</p>
		<p class="mt-2 text-sm text-stone-700">
			<span class="font-medium">Recommendation:</span> {output.output.recommendation}
		</p>
	</div>

	{#if (output.output.findings ?? []).length > 0}
		<div class="border-t border-stone-300 px-4 py-3">
			<h4 class="text-xs font-medium tracking-wide text-zinc-500 uppercase">Findings</h4>
			<ul class="mt-2 space-y-2">
				{#each output.output.findings ?? [] as f (f.title)}
					<li class="border-l-2 border-stone-300 pl-3">
						<p class="text-sm">
							<span class="font-semibold uppercase {severityClass[f.severity] ?? severityClass.low}">{f.severity}</span>
							<span class="text-stone-400">· {f.type}</span>
							<span class="font-medium text-stone-900">— {f.title}</span>
						</p>
						<p class="mt-1 text-sm text-stone-700">{f.body}</p>
					</li>
				{/each}
			</ul>
		</div>
	{/if}

	{#if (output.output.open_questions ?? []).length > 0}
		<div class="border-t border-stone-300 px-4 py-3">
			<h4 class="text-xs font-medium tracking-wide text-zinc-500 uppercase">Open questions</h4>
			<ul class="mt-2 list-disc space-y-1 pl-5 text-sm text-stone-700">
				{#each output.output.open_questions ?? [] as q}
					<li>{q}</li>
				{/each}
			</ul>
		</div>
	{/if}

	{#if output.thinking}
		<div class="border-t border-stone-300 px-4 py-2">
			<button
				type="button"
				class="text-xs font-medium text-zinc-500 underline hover:text-stone-900"
				aria-expanded={expanded}
				onclick={() => onToggle(output.agent_id)}
			>
				{expanded ? 'Hide' : 'Show'} thinking trace
			</button>
			{#if expanded}
				<pre class="mt-2 max-h-72 overflow-auto whitespace-pre-wrap rounded bg-white p-3 text-xs leading-relaxed text-stone-500">{output.thinking}</pre>
			{/if}
		</div>
	{/if}

	{#if output.error}
		<p class="border-t border-stone-300 px-4 py-2 text-xs text-red-700">{output.error}</p>
	{/if}
</article>