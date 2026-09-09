<script lang="ts">
	import { api, type IntakeExplanation } from '$lib/api';

	let {
		sessionId,
		rawInput,
		rawSource,
		onChanged
	}: {
		sessionId: string;
		rawInput: string;
		rawSource: string;
		onChanged: () => void;
	} = $props();

	let explanation = $state<IntakeExplanation | null>(null);
	let busy = $state(false);
	let error = $state('');
	let showRevert = $state(false);

	$effect(() => {
		api.intakeExplain().then((e) => (explanation = e)).catch(() => (explanation = null));
	});

	const distilled = $derived(rawSource !== '');

	async function distill() {
		busy = true;
		error = '';
		try {
			await api.runIntake(sessionId);
			onChanged();
		} catch (e) {
			error = String(e);
		} finally {
			busy = false;
		}
	}

	async function revert() {
		busy = true;
		error = '';
		try {
			await api.revertIntake(sessionId);
			showRevert = false;
			onChanged();
		} catch (e) {
			error = String(e);
		} finally {
			busy = false;
		}
	}
</script>

<section class="rounded border border-stone-300 bg-white">
	<header class="border-b border-stone-300 px-4 py-3">
		<h2 class="text-sm font-semibold text-stone-900">Intake — Uchikomi</h2>
		<p class="mt-1 text-sm text-stone-500">
			Decide how the panel reads your material. This choice is yours; the system only explains
			the trade-off.
		</p>
	</header>

	{#if error}
		<p class="border-b border-red-200 bg-red-50 px-4 py-2 text-sm text-red-700">{error}</p>
	{/if}

	<div class="grid gap-px bg-zinc-300 sm:grid-cols-2">
		<div class="bg-white p-4">
			<h3 class="text-sm font-semibold text-stone-900">
				{explanation?.distill.label ?? 'Distill first'}
			</h3>
			<p class="mt-2 text-sm text-stone-500">{explanation?.distill.when ?? ''}</p>
			<p class="mt-2 text-sm text-stone-500">{explanation?.distill.why ?? ''}</p>
			<p class="mt-2 text-xs text-zinc-500">{explanation?.distill.cost ?? ''}</p>
			<button
				type="button"
				class="mt-3 rounded border border-stone-300 bg-white px-3 py-1.5 text-sm font-medium text-stone-900 hover:bg-stone-100 disabled:opacity-50"
				disabled={busy || distilled}
				onclick={distill}
			>
				{distilled ? 'Distilled' : 'Distill first'}
			</button>
		</div>
		<div class="bg-white p-4">
			<h3 class="text-sm font-semibold text-stone-900">
				{explanation?.skip.label ?? 'Use as-is'}
			</h3>
			<p class="mt-2 text-sm text-stone-500">{explanation?.skip.when ?? ''}</p>
			<p class="mt-2 text-sm text-stone-500">{explanation?.skip.why ?? ''}</p>
			<p class="mt-2 text-xs text-zinc-500">{explanation?.skip.cost ?? ''}</p>
			<p class="mt-3 text-sm text-stone-500">
				{distilled
					? 'Continue below with the distilled brief, or revert to your original wording.'
					: 'Continue below with your material as written.'}
			</p>
		</div>
	</div>

	{#if distilled}
		<div class="border-t border-stone-300 p-4">
			<div class="flex items-center justify-between gap-4">
				<p class="text-sm text-stone-500">
					Input was distilled. The original is preserved and restorable.
				</p>
				<button
					type="button"
					class="text-sm text-stone-500 underline hover:text-stone-900"
					onclick={() => (showRevert = !showRevert)}
				>
					{showRevert ? 'Keep distilled brief' : 'Revert distillation'}
				</button>
			</div>
			{#if showRevert}
				<div class="mt-3 flex items-center gap-3">
					<p class="text-sm text-stone-500">Restore your original wording and drop the brief?</p>
					<button
						type="button"
						class="rounded border border-red-300 bg-red-50 px-3 py-1.5 text-sm font-medium text-red-700 hover:bg-red-100"
						disabled={busy}
						onclick={revert}
					>
						Revert
					</button>
				</div>
			{/if}
		</div>
	{/if}

	<details class="border-t border-stone-300 p-4">
		<summary class="cursor-pointer text-sm font-medium text-stone-700">
			{distilled ? 'Distilled brief' : 'Original material'}
		</summary>
		{#if distilled}
			<p class="mt-3 mb-1 text-xs text-zinc-500">Distilled brief (what the panel reads):</p>
			<pre class="max-h-72 overflow-auto whitespace-pre-wrap rounded bg-white p-3 text-xs leading-relaxed text-stone-700">{rawInput}</pre>
			<p class="mt-3 mb-1 text-xs text-zinc-500">Original material (preserved):</p>
			<pre class="max-h-72 overflow-auto whitespace-pre-wrap rounded bg-white p-3 text-xs leading-relaxed text-stone-500">{rawSource}</pre>
		{:else}
			<pre class="max-h-96 overflow-auto whitespace-pre-wrap text-xs leading-relaxed text-stone-700">{rawInput}</pre>
		{/if}
	</details>
</section>