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

	const severityClass: Record<string, string> = {
		critical: 'bg-red-100 text-red-900 border-red-300',
		high: 'bg-orange-100 text-orange-900 border-orange-300',
		medium: 'bg-amber-100 text-amber-900 border-amber-300',
		low: 'bg-zinc-100 text-zinc-700 border-zinc-300'
	};

	const statusLabel: Record<string, string> = {
		done: 'done',
		partial: 'partial — degraded output, stored',
		error: 'failed — no contribution'
	};
</script>

<article class="border border-zinc-300 bg-white">
	<header class="flex flex-wrap items-baseline justify-between gap-2 border-b border-zinc-200 px-4 py-2">
		<h3 class="text-sm font-semibold text-zinc-900">{output.display_name}</h3>
		<span class="text-xs text-zinc-500">{statusLabel[output.status] ?? output.status}</span>
	</header>

	<div class="px-4 py-3">
		<p class="text-sm leading-relaxed text-zinc-800">{output.output.summary}</p>
		<p class="mt-2 text-sm text-zinc-700">
			<span class="font-medium">Recommendation:</span> {output.output.recommendation}
		</p>
	</div>

	{#if output.output.findings.length > 0}
		<div class="border-t border-zinc-200 px-4 py-3">
			<h4 class="text-xs font-medium tracking-wide text-zinc-500 uppercase">Findings</h4>
			<ul class="mt-2 space-y-2">
				{#each output.output.findings as f (f.title)}
					<li class="border {severityClass[f.severity] ?? severityClass.low} rounded px-3 py-2">
						<p class="text-sm font-medium">
							<span class="font-semibold uppercase">{f.severity}</span>
							<span class="text-opacity-70">· {f.type}</span>
							— {f.title}
						</p>
						<p class="mt-1 text-sm">{f.body}</p>
					</li>
				{/each}
			</ul>
		</div>
	{/if}

	{#if output.output.open_questions.length > 0}
		<div class="border-t border-zinc-200 px-4 py-3">
			<h4 class="text-xs font-medium tracking-wide text-zinc-500 uppercase">Open questions</h4>
			<ul class="mt-2 list-disc space-y-1 pl-5 text-sm text-zinc-700">
				{#each output.output.open_questions as q}
					<li>{q}</li>
				{/each}
			</ul>
		</div>
	{/if}

	{#if output.thinking}
		<div class="border-t border-zinc-200 px-4 py-2">
			<button
				type="button"
				class="text-xs font-medium text-zinc-500 underline hover:text-zinc-900"
				aria-expanded={expanded}
				onclick={() => onToggle(output.agent_id)}
			>
				{expanded ? 'Hide' : 'Show'} thinking trace
			</button>
			{#if expanded}
				<pre class="mt-2 max-h-72 overflow-auto whitespace-pre-wrap rounded bg-zinc-50 p-3 text-xs leading-relaxed text-zinc-600">{output.thinking}</pre>
			{/if}
		</div>
	{/if}

	{#if output.error}
		<p class="border-t border-zinc-200 px-4 py-2 text-xs text-red-800">{output.error}</p>
	{/if}
</article>