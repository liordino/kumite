<script lang="ts">
	// Renders the PSD markdown without a markdown dependency and without any
	// innerHTML: line-classified plain text, styled by line kind. Sections are
	// always shown in their fixed order; empty sections show their placeholder
	// exactly as Shishō wrote it.
	let { psd }: { psd: string } = $props();

	type Line = { kind: 'frontmatter' | 'h1' | 'h2' | 'item' | 'text' | 'blank'; text: string };

	const lines: Line[] = $derived.by(() => {
		const out: Line[] = [];
		let inFrontmatter = false;
		let frontmatterDone = false;
		for (const raw of psd.split('\n')) {
			const line = raw.trimEnd();
			if (!frontmatterDone) {
				if (line === '---') {
					if (!inFrontmatter) {
						inFrontmatter = true;
						out.push({ kind: 'blank', text: '' });
						continue;
					}
					frontmatterDone = true;
					out.push({ kind: 'blank', text: '' });
					continue;
				}
				if (inFrontmatter) {
					out.push({ kind: 'frontmatter', text: line });
					continue;
				}
			}
			if (line === '---') {
				out.push({ kind: 'blank', text: '' });
				continue;
			}
			if (line.startsWith('# ')) out.push({ kind: 'h1', text: line.slice(2) });
			else if (line.startsWith('## ')) out.push({ kind: 'h2', text: line.slice(3) });
			else if (line.startsWith('- ')) out.push({ kind: 'item', text: line.slice(2) });
			else if (line === '') out.push({ kind: 'blank', text: '' });
			else out.push({ kind: 'text', text: line });
		}
		return out;
	});
</script>

<article class="rounded border border-zinc-300 bg-white">
	<header class="border-b border-zinc-300 px-6 py-4">
		<h2 class="text-sm font-semibold tracking-wide text-zinc-500 uppercase">Project Summary Document</h2>
	</header>
	<div class="px-6 py-5">
		{#each lines as line, i (i)}
			{#if line.kind === 'h1'}
				<h3 class="mt-6 mb-2 text-lg font-semibold text-zinc-900 first:mt-0">{line.text}</h3>
			{:else if line.kind === 'h2'}
				<h4 class="mt-6 mb-2 border-b border-zinc-200 pb-1 text-sm font-semibold tracking-wide text-zinc-900 uppercase">
					{line.text}
				</h4>
			{:else if line.kind === 'item'}
				<p class="border-l-2 border-zinc-300 py-0.5 pl-3 text-sm leading-relaxed text-zinc-800">
					{line.text}
				</p>
			{:else if line.kind === 'frontmatter'}
				<p class="font-mono text-xs text-zinc-500">{line.text}</p>
			{:else if line.kind === 'text'}
				<p class="my-2 text-sm leading-relaxed text-zinc-800">{line.text}</p>
			{:else}
				<div class="h-2"></div>
			{/if}
		{/each}
	</div>
</article>