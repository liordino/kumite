// View state only — what is expanded, what is selected, what is being
// watched. No session state lives here: everything durable comes from the
// API and reloads from it.

import { writable } from "svelte/store";

/** Session id of the run currently being watched, when any. */
export const watchingRun = writable<string | null>(null);

/** Agent ids whose thinking trace is expanded in the execution monitor. */
export const expandedThinking = writable<Set<string>>(new Set());

export function toggleThinking(agentId: string): void {
	expandedThinking.update((set) => {
		const next = new Set(set);
		if (next.has(agentId)) next.delete(agentId);
		else next.add(agentId);
		return next;
	});
}
