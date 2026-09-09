<script lang="ts">
	import {
		parsePsd,
		parseBlocks,
		parseInline,
		downloadMarkdown,
		type MdBlock,
		type MdInline
	} from '$lib/markdown';

	let { psd, projectName }: { psd: string; projectName?: string } = $props();

	const parsed = $derived(parsePsd(psd));

	const renderedSections = $derived(
		parsed.sections.map((s) => ({ title: s.title, blocks: parseBlocks(s.body) }))
	);

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
			<strong class="font-semibold text-zinc-100">{seg.text}</strong>
		{:else if seg.kind === 'em'}
			<em class="italic text-zinc-200">{seg.text}</em>
		{:else if seg.kind === 'code'}
			<code class="rounded bg-zinc-800 px-1 py-0.5 text-[0.85em] text-zinc-200">{seg.text}</code>
		{:else}
			{seg.text}
		{/if}
	{/each}
{/snippet}

{#snippet renderBlocks(blocks: MdBlock[])}
	{#each blocks as block}
		{#if block.kind === 'heading'}
			<h4 class="mt-4 mb-1 text-sm font-semibold tracking-wide text-zinc-100">
				{@render renderInlines(block.inlines)}
			</h4>
		{:else if block.kind === 'paragraph'}
			<p class="my-2 text-sm leading-relaxed text-zinc-300">
				{@render renderInlines(block.inlines)}
			</p>
		{:else if block.kind === 'list'}
			<ul class="my-2 list-disc space-y-1 pl-5">
				{#each block.items as itemInlines}
					<li class="text-sm leading-relaxed text-zinc-300">
						{@render renderInlines(itemInlines)}
					</li>
				{/each}
			</ul>
		{:else if block.kind === 'blank'}
			<div class="h-1"></div>
		{/if}
	{/each}
{/snippet}

<article class="overflow-hidden rounded border border-zinc-800 bg-zinc-900">
	<header class="flex flex-wrap items-center justify-between gap-3 border-b border-zinc-800 px-6 py-4">
		<div>
			<p class="text-xs font-medium tracking-wide text-zinc-500 uppercase">Project Summary Document</p>
			{#if parsed.title}
				<h2 class="mt-0.5 text-base font-semibold text-zinc-100">{parsed.title}</h2>
			{/if}
		</div>
		<button
			type="button"
			class="rounded border border-zinc-700 px-3 py-1.5 text-xs font-medium text-zinc-300 hover:bg-zinc-800 hover:text-zinc-100"
			onclick={() => download()}
		>
			Download .md
		</button>
	</header>

	{#if parsed.frontmatter}
		<details class="border-b border-zinc-800">
			<summary class="cursor-pointer px-6 py-2 text-xs text-zinc-500 hover:text-zinc-300">
				frontmatter
			</summary>
			<pre class="overflow-auto px-6 pb-3 text-xs leading-relaxed text-zinc-500">{parsed.frontmatter}</pre>
		</details>
	{/if}

	{#each renderedSections as section, i (section.title)}
		<details class="border-b border-zinc-800 last:border-b-0" open={i < 2}>
			<summary class="cursor-pointer px-6 py-3 text-sm font-semibold text-zinc-200 select-none hover:text-zinc-100">
				{section.title}
			</summary>
			<div class="px-6 pb-5">
				{#if section.blocks.length === 0}
					<p class="text-sm text-zinc-500 italic">_No panel contribution for this section._</p>
				{:else}
					{#each section.blocks as block}
						{#if block.kind === 'heading'}
							<h4 class="mt-4 mb-1 text-sm font-semibold tracking-wide text-zinc-100">
								{@render renderInlines(block.inlines)}
							</h4>
						{:else if block.kind === 'paragraph'}
							<p class="my-2 text-sm leading-relaxed text-zinc-300">
								{@render renderInlines(block.inlines)}
							</p>
						{:else if block.kind === 'list'}
							<ul class="my-2 list-disc space-y-1 pl-5">
								{#each block.items as itemInlines}
									<li class="text-sm leading-relaxed text-zinc-300">
										{@render renderInlines(itemInlines)}
									</li>
								{/each}
							</ul>
						{:else if block.kind === 'blank'}
							<div class="h-1"></div>
						{/if}
					{/each}
				{/if}
			</div>
		</details>
	{/each}
</article>