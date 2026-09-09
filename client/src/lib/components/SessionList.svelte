<script lang="ts">
	import type { SessionListItem } from '$lib/api';

	let {
		sessions,
		onDelete
	}: {
		sessions: SessionListItem[];
		onDelete: (id: string) => void;
	} = $props();

	let confirmId = $state<string | null>(null);

	const phaseLabel: Record<string, string> = {
		intake: 'intake',
		distilling: 'distilling',
		pipeline_review: 'builder',
		running: 'running',
		synthesis: 'synthesis',
		interrupted: 'interrupted',
		complete: 'complete',
		error: 'error'
	};

	function severityText(s: SessionListItem['finding_summary']): string {
		const parts: string[] = [];
		if (s.critical) parts.push(`${s.critical} critical`);
		if (s.high) parts.push(`${s.high} high`);
		if (s.medium) parts.push(`${s.medium} medium`);
		if (s.low) parts.push(`${s.low} low`);
		return parts.length ? parts.join(' · ') : 'no findings';
	}
</script>

<div class="overflow-x-auto rounded border border-stone-300 bg-white">
	<table class="w-full text-sm">
		<thead>
			<tr class="border-b border-stone-300 bg-white text-left text-xs font-medium tracking-wide text-zinc-500 uppercase">
				<th class="px-4 py-2">Project</th>
				<th class="px-4 py-2">Phase</th>
				<th class="px-4 py-2">Domain</th>
				<th class="px-4 py-2">Findings</th>
				<th class="px-4 py-2">Updated</th>
				<th class="px-4 py-2"><span class="sr-only">Actions</span></th>
			</tr>
		</thead>
		<tbody>
			{#each sessions as s (s.id)}
				<tr class="border-b border-stone-300 last:border-b-0 hover:bg-stone-100">
					<td class="px-4 py-2">
						<a class="font-medium text-stone-900 underline decoration-zinc-300 hover:decoration-stone-900" href="/session/{s.id}">
							{s.project_name}
						</a>
						{#if s.distilled}<span class="ml-2 text-xs text-zinc-500">distilled</span>{/if}
					</td>
					<td class="px-4 py-2 text-stone-500">{phaseLabel[s.phase] ?? s.phase}</td>
					<td class="px-4 py-2 text-stone-500">{s.domain || '—'}</td>
					<td class="px-4 py-2 text-stone-500">{severityText(s.finding_summary)}</td>
					<td class="px-4 py-2 text-zinc-500">{new Date(s.updated_at).toLocaleString()}</td>
					<td class="px-4 py-2 text-right">
						{#if confirmId === s.id}
							<button
								type="button"
								class="rounded border border-red-300 bg-red-50 px-2 py-1 text-xs font-medium text-red-700 hover:bg-red-100"
								onclick={() => {
									onDelete(s.id);
									confirmId = null;
								}}
							>
								Confirm delete
							</button>
							<button
								type="button"
								class="ml-2 text-xs text-zinc-500 underline"
								onclick={() => (confirmId = null)}
							>
								keep
							</button>
						{:else}
							<button
								type="button"
								class="text-xs text-zinc-500 underline hover:text-stone-900"
								onclick={() => (confirmId = s.id)}
							>
								delete
							</button>
						{/if}
					</td>
				</tr>
			{:else}
				<tr>
					<td class="px-4 py-6 text-center text-zinc-500" colspan="6">
						No sessions yet. Submit a project idea above.
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>