<script lang="ts">
	// Collapsed by default; omitted entirely when the model emitted no trace.
	let { agentId, thinking, expanded, onToggle }: {
		agentId: string;
		thinking: string;
		expanded: boolean;
		onToggle: (agentId: string) => void;
	} = $props();
</script>

{#if thinking}
	<button
		type="button"
		class="text-xs text-zinc-500 underline hover:text-stone-900"
		aria-expanded={expanded}
		onclick={() => onToggle(agentId)}
	>
		{expanded ? 'Hide' : 'Show'} thinking
	</button>
	{#if expanded}
		<pre class="mt-1 max-h-64 overflow-auto whitespace-pre-wrap rounded bg-white p-2 text-xs text-stone-500">{thinking}</pre>
	{/if}
{/if}