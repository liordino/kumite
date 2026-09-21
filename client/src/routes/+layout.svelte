<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<div class="min-h-screen bg-stone-100 text-stone-900">
	<header class="rule-division border-b border-stone-300 bg-white">
		<div class="mx-auto flex max-w-4xl items-center justify-between px-4 py-3">
			<a class="flex items-center gap-2.5" href="/">
				<span class="seal h-7 w-7 text-sm" title="Kumite — 組手">組</span>
				<span class="display text-sm font-bold tracking-[0.18em] text-stone-900 uppercase">Kumite</span>
			</a>
			<nav class="flex items-center gap-4 text-sm">
				<a class="display text-stone-500 uppercase tracking-widest text-xs hover:text-stone-900" href="/">Sessions</a>
				<a class="display text-stone-500 uppercase tracking-widest text-xs hover:text-stone-900" href="/settings">Settings</a>
			</nav>
		</div>
	</header>
	<main class="mx-auto max-w-4xl px-4 py-6">
		{@render children()}
	</main>
</div>