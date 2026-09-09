<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<div class="min-h-screen bg-zinc-900 text-zinc-100">
	<header class="border-b border-zinc-800 bg-zinc-900">
		<div class="mx-auto flex max-w-4xl items-center justify-between px-4 py-3">
			<a class="text-sm font-semibold tracking-wide text-zinc-100" href="/">KUMITE</a>
			<nav class="flex items-center gap-4 text-sm">
				<a class="text-zinc-400 hover:text-zinc-100" href="/">Sessions</a>
				<a class="text-zinc-400 hover:text-zinc-100" href="/settings">Settings</a>
			</nav>
		</div>
	</header>
	<main class="mx-auto max-w-4xl px-4 py-6">
		{@render children()}
	</main>
</div>
