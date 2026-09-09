<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<div class="min-h-screen bg-white text-stone-900">
	<header class="border-b border-stone-300 bg-white">
		<div class="mx-auto flex max-w-4xl items-center justify-between px-4 py-3">
			<a class="text-sm font-semibold tracking-wide text-stone-900" href="/">KUMITE</a>
			<nav class="flex items-center gap-4 text-sm">
				<a class="text-stone-500 hover:text-stone-900" href="/">Sessions</a>
				<a class="text-stone-500 hover:text-stone-900" href="/settings">Settings</a>
			</nav>
		</div>
	</header>
	<main class="mx-auto max-w-4xl px-4 py-6">
		{@render children()}
	</main>
</div>
