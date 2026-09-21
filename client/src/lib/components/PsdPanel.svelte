<script lang="ts">
	import {
		parsePsd,
		parseBlocks,
		downloadMarkdown,
		type MdBlock,
		type MdInline
	} from '$lib/markdown';

	let { psd, projectName }: { psd: string; projectName?: string } = $props();

	const parsed = $derived(parsePsd(psd));

	const renderedSections = $derived(
		parsed.sections.map((s) => ({
			...splitSectionTitle(s.title),
			blocks: parseBlocks(s.body)
		}))
	);

	// "1. Overview" → 1 + "Overview"; unnumbered titles pass through whole.
	function splitSectionTitle(title: string): { num: string; text: string } {
		const m = /^(\d+)[.)]\s+(.*)$/.exec(title);
		return m ? { num: m[1], text: m[2] } : { num: '', text: title };
	}

	// Attribution line from the frontmatter: the panel roster that produced
	// this document, styled as a byline row.
	const panelByline = $derived.by(() => {
		const m = /^panel:\s*\[(.*)\]\s*$/m.exec(parsed.frontmatter);
		if (!m) return [];
		return m[1]
			.split(',')
			.map((s) => s.trim().replace(/^["']|["']$/g, ''))
			.filter((s) => s.length > 0);
	});

	function download() {
		const name = (parsed.title || projectName || 'psd')
			.replace(/[^a-z0-9]+/gi, '-')
			.toLowerCase();
		downloadMarkdown(`${name}.md`, psd);
	}
</script>

{#snippet renderInlines(inlines: MdInline[])}
	{#each inlines as seg}
		{#if seg.kind === 'strong'}
			<strong class="font-semibold text-stone-900">{seg.text}</strong>
		{:else if seg.kind === 'em'}
			<em class="italic text-stone-800">{seg.text}</em>
		{:else if seg.kind === 'code'}
			<code class="rounded bg-stone-200 px-1 py-0.5 text-[0.85em] text-stone-800">{seg.text}</code>
		{:else}
			{seg.text}
		{/if}
	{/each}
{/snippet}

{#snippet renderBlocks(blocks: MdBlock[])}
	{#each blocks as block}
		{#if block.kind === 'heading'}
			<h4 class="mt-4 mb-1 text-sm font-semibold tracking-wide text-stone-900">
				{@render renderInlines(block.inlines)}
			</h4>
		{:else if block.kind === 'paragraph'}
			<p class="my-2 text-sm leading-relaxed text-stone-700">
				{@render renderInlines(block.inlines)}
			</p>
		{:else if block.kind === 'list'}
			<ul class="my-2 list-disc space-y-1 pl-5">
				{#each block.items as itemInlines}
					<li class="text-sm leading-relaxed text-stone-700">
						{@render renderInlines(itemInlines)}
					</li>
				{/each}
			</ul>
		{:else if block.kind === 'blank'}
			<div class="h-1"></div>
		{/if}
	{/each}
{/snippet}

<article class="overflow-hidden border border-stone-300 bg-white">
	<header class="rule-division flex flex-wrap items-center justify-between gap-3 border-b border-stone-300 px-6 py-4">
		<div>
			<div class="flex items-center gap-3">
				<span class="seal h-9 w-9 text-lg" title="Official verdict">判</span>
				<div>
					<p class="display text-xs font-bold uppercase tracking-[0.2em] text-stone-900">Project Summary Document</p>
					{#if parsed.title}
						<h2 class="display mt-0.5 text-lg font-bold text-stone-900">{parsed.title}</h2>
					{/if}
				</div>
			</div>
			{#if panelByline.length > 0}
				<p class="display mt-1.5 text-[10px] font-medium uppercase tracking-widest text-stone-500">
					Panel — {panelByline.join(' · ')}
				</p>
			{/if}
		</div>
		<button
			type="button"
			class="display rounded-none border border-stone-300 px-3 py-1.5 text-xs font-semibold uppercase tracking-wider text-stone-700 hover:bg-stone-100"
			onclick={() => download()}
		>
			Download .md
		</button>
	</header>

	{#if parsed.frontmatter}
		<details class="border-b border-stone-300">
			<summary class="cursor-pointer px-6 py-2 text-xs text-zinc-500 hover:text-stone-700">
				frontmatter
			</summary>
			<pre class="overflow-auto px-6 pb-3 text-xs leading-relaxed text-zinc-500">{parsed.frontmatter}</pre>
		</details>
	{/if}

	{#each renderedSections as section, i (section.text)}
		<details class="border-b border-stone-300 last:border-b-0" open={i < 2}>
			<summary class="flex cursor-pointer items-center gap-3 px-6 py-3 text-sm select-none">
				<span class="display flex h-7 w-7 shrink-0 items-center justify-center border border-stone-300 bg-stone-100 text-sm font-bold tabular text-stone-800">
					{section.num || '·'}
				</span>
				<span class="display font-semibold text-stone-900">{section.text}</span>
			</summary>
			<div class="settle px-6 pb-5 pl-16">
				{#if section.blocks.length === 0}
					<p class="text-sm text-zinc-500 italic">_No panel contribution for this section._</p>
				{:else}
					{@render renderBlocks(section.blocks)}
				{/if}
			</div>
		</details>
	{/each}
</article>