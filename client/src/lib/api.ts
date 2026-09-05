// Typed API client. The backend is client-agnostic: all state lives
// server-side, this module only moves JSON and SSE over relative /api paths
// (same-origin in production; Vite dev proxy in development).

export type SessionPhase =
	| 'intake'
	| 'distilling'
	| 'pipeline_review'
	| 'running'
	| 'synthesis'
	| 'interrupted'
	| 'complete'
	| 'error';

export type AgentStatus = 'pending' | 'running' | 'done' | 'skipped' | 'error';

export type Wave = 1 | 2 | 3;

export interface AgentNode {
	id: string;
	agent_id: string;
	display_name: string;
	role_summary: string;
	source_url: string;
	wave: Wave;
	status: AgentStatus;
	enabled: boolean;
	rationale: string;
}

export interface Finding {
	type: 'opportunity' | 'risk' | 'question' | 'constraint';
	severity: 'low' | 'medium' | 'high' | 'critical';
	title: string;
	body: string;
}

export interface PsdContribution {
	section: string;
	content: string;
}

export interface AgentOutputData {
	summary: string;
	findings: Finding[];
	recommendation: string;
	open_questions: string[];
	psd_contributions: PsdContribution[];
}

export interface AgentOutput {
	agent_id: string;
	display_name: string;
	wave: number;
	status: 'done' | 'partial' | 'error';
	output: AgentOutputData;
	thinking: string;
	error?: string;
}

export interface PipelineContext {
	problem_statement: string;
	maturity: string;
	four_block: {
		what_is_wanted: string;
		how_it_should_be_done: string;
		what_is_not_wanted: string;
		how_success_is_measured: string;
	};
	flags: string[];
	domain: string;
	tags: string[];
	distilled: boolean;
}

export interface PipelinePlan {
	session_id: string;
	project_name: string;
	input_type: string;
	project_type: string;
	pipeline: { wave1: AgentNode[]; wave2: AgentNode[]; fixed: AgentNode[] };
	context: PipelineContext;
}

export interface FindingSummary {
	critical: number;
	high: number;
	medium: number;
	low: number;
}

export interface Session {
	id: string;
	project_name: string;
	domain: string;
	tags: string[];
	created_at: string;
	updated_at: string;
	phase: SessionPhase;
	raw_source: string;
	raw_input: string;
	pipeline_plan: PipelinePlan | null;
	agent_outputs: AgentOutput[];
	psd: string;
	finding_summary: FindingSummary;
}

export interface SessionListItem {
	id: string;
	project_name: string;
	domain: string;
	tags: string[];
	created_at: string;
	updated_at: string;
	phase: SessionPhase;
	distilled: boolean;
	agent_count: number;
	has_psd: boolean;
	finding_summary: FindingSummary;
}

export interface RosterEntry {
	agent_id: string;
	display_name: string;
	division: string;
	source_url: string;
	default_wave: number;
	conditional: boolean;
	condition?: string;
	fixed: boolean;
}

export interface RepoAgent {
	agent_id: string;
	display_name: string;
	source_url: string;
}

export interface CustomAgent {
	id: string;
	display_name: string;
	role_summary: string;
	wave_preference: number;
	system_prompt: string;
	created_at: string;
	updated_at: string;
}

export interface IntakeExplanation {
	distill: { label: string; when: string; why: string; cost: string };
	skip: { label: string; when: string; why: string; cost: string };
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
	const res = await fetch(path, {
		method,
		headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
		body: body === undefined ? undefined : JSON.stringify(body)
	});
	const text = await res.text();
	let parsed: unknown = null;
	if (text) {
		try {
			parsed = JSON.parse(text);
		} catch {
			// non-JSON body — fall through with the status error
		}
	}
	if (!res.ok) {
		const msg =
			parsed && typeof parsed === 'object' && 'error' in parsed
				? String((parsed as { error: unknown }).error)
				: `${res.status} ${res.statusText}`;
		throw new Error(msg);
	}
	return parsed as T;
}

export const api = {
	// Sessions
	listSessions(filters?: Record<string, string>): Promise<SessionListItem[]> {
		const q = filters ? '?' + new URLSearchParams(filters).toString() : '';
		return request('GET', '/api/sessions' + q);
	},
	getSession(id: string): Promise<Session> {
		return request('GET', `/api/sessions/${id}`);
	},
	createSession(project_name: string, raw_input: string): Promise<{ id: string }> {
		return request('POST', '/api/sessions', { project_name, raw_input });
	},
	patchSession(id: string, patch: Partial<Session>): Promise<{ ok: boolean }> {
		return request('PATCH', `/api/sessions/${id}`, patch);
	},
	deleteSession(id: string): Promise<{ ok: boolean }> {
		return request('DELETE', `/api/sessions/${id}`);
	},

	// Intake
	intakeExplain(): Promise<IntakeExplanation> {
		return request('GET', '/api/intake/explain');
	},
	runIntake(id: string): Promise<{ raw_input: string; raw_source: string }> {
		return request('POST', `/api/intake/${id}`);
	},
	revertIntake(id: string): Promise<{ ok: boolean }> {
		return request('DELETE', `/api/intake/${id}`);
	},

	// Pipeline
	phase0(id: string): Promise<{ pipeline_plan: PipelinePlan }> {
		return request('POST', `/api/pipeline/phase0/${id}`);
	},
	updatePlan(id: string, plan: PipelinePlan): Promise<{ ok: boolean }> {
		return request('PATCH', `/api/pipeline/${id}/plan`, plan);
	},
	run(id: string): Promise<Response> {
		return fetch(`/api/pipeline/run/${id}`, { method: 'POST' });
	},
	resume(id: string): Promise<Response> {
		return fetch(`/api/pipeline/resume/${id}`, { method: 'POST' });
	},

	// Agents
	roster(): Promise<RosterEntry[]> {
		return request('GET', '/api/agents/roster');
	},
	repoListing(): Promise<RepoAgent[]> {
		return request('GET', '/api/agents/repo');
	},
	listCustomAgents(): Promise<CustomAgent[]> {
		return request('GET', '/api/agents/custom');
	},
	createCustomAgent(a: Omit<CustomAgent, 'id' | 'created_at' | 'updated_at'>): Promise<{ id: string }> {
		return request('POST', '/api/agents/custom', a);
	},
	deleteCustomAgent(id: string): Promise<{ ok: boolean }> {
		return request('DELETE', `/api/agents/custom/${id}`);
	},

	// Handoff
	handoff(sessionId: string): Promise<{ brief_md: string; context_md: string }> {
		return request('POST', `/api/handoff/${sessionId}`);
	},

	// Config
	getConfig(): Promise<Record<string, string>> {
		return request('GET', '/api/config');
	},
	putConfig(values: Record<string, string>): Promise<{ ok: boolean }> {
		return request('PUT', '/api/config', values);
	}
};

// SSE event parsed from a run/resume stream.
export interface RunEvent {
	event: string;
	data: Record<string, unknown>;
}

// streamRun consumes the SSE body of a run/resume POST. The fetch response
// is a text/event-stream; parse it line by line until the server closes.
export async function streamRun(
	promise: Promise<Response>,
	onEvent: (ev: RunEvent) => void
): Promise<void> {
	const res = await promise;
	if (!res.ok || !res.body) {
		const text = await res.text().catch(() => '');
		let msg = `${res.status} ${res.statusText}`;
		try {
			const parsed = JSON.parse(text) as { error?: string };
			if (parsed.error) msg = parsed.error;
		} catch {
			/* keep status message */
		}
		throw new Error(msg);
	}
	const reader = res.body.getReader();
	const decoder = new TextDecoder();
	let buf = '';
	for (;;) {
		const { done, value } = await reader.read();
		if (done) break;
		buf += decoder.decode(value, { stream: true });
		let idx: number;
		while ((idx = buf.indexOf('\n\n')) >= 0) {
			const block = buf.slice(0, idx);
			buf = buf.slice(idx + 2);
			let event = 'message';
			let data = '';
			for (const line of block.split('\n')) {
				if (line.startsWith('event: ')) event = line.slice(7).trim();
				else if (line.startsWith('data: ')) data += line.slice(6);
			}
			let parsed: Record<string, unknown> = {};
			try {
				parsed = JSON.parse(data) as Record<string, unknown>;
			} catch {
				/* empty data blocks */
			}
			onEvent({ event, data: parsed });
		}
	}
}