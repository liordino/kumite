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

<article class="overflow-hidden rounded border border-stone-300 bg-white">
	<header class="flex flex-wrap items-center justify-between gap-3 border-b border-stone-300 px-6 py-4">
		<div>
			<p class="text-[10px] font-semibold tracking-[0.2em] text-zinc-500 uppercase">Project Summary Document</p>
			{#if parsed.title}
				<h2 class="mt-0.5 text-base font-semibold text-stone-900">{parsed.title}</h2>
			{/if}
			{#if panelByline.length > 0}
				<p class="mt-1.5 text-[10px] font-medium tracking-widest text-zinc-500 uppercase">
					Panel — {panelByline.join(' · ')}
				</p>
			{/if}
		</div>
		<button
			type="button"
			class="rounded border border-stone-300 px-3 py-1.5 text-xs font-medium text-stone-700 hover:bg-stone-100"
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
				<span class="flex h-7 w-7 shrink-0 items-center justify-center border border-stone-300 text-xs font-semibold text-stone-700">
					{section.num || '·'}
				</span>
				<span class="font-semibold text-stone-900">{section.text}</span>
			</summary>
			<div class="px-6 pb-5 pl-16">
				{#if section.blocks.length === 0}
					<p class="text-sm text-zinc-500 italic">_No panel contribution for this section._</p>
				{:else}
					{@render renderBlocks(section.blocks)}
				{/if}
			</div>
		</details>
	{/each}
</article>