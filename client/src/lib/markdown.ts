// Minimal, XSS-safe markdown rendering for Kumite surfaces: the PSD and the
// handoff artifacts are model-generated but structurally constrained (fixed
// section headings, paragraphs, lists, bold, attribution lines). The parser
// tokenizes into typed blocks rendered as Svelte elements — never innerHTML,
// so model output can never inject markup.

export type MdInline =
	| { kind: "text"; text: string }
	| { kind: "strong"; text: string }
	| { kind: "em"; text: string }
	| { kind: "code"; text: string };

export type MdBlock =
	| { kind: "heading"; level: 1 | 2 | 3; inlines: MdInline[] }
	| { kind: "paragraph"; inlines: MdInline[] }
	| { kind: "list"; items: MdInline[][] }
	| { kind: "blank" };

// Inline tokenizer: **strong**, *em* and `code` → typed segments. Everything
// else is text. No HTML is ever produced — the segments render as Svelte
// elements, so model output can never inject markup.
export function parseInline(text: string): MdInline[] {
	const out: MdInline[] = [];
	let buf = "";
	let i = 0;
	const flush = () => {
		if (buf) out.push({ kind: "text", text: buf });
		buf = "";
	};
	while (i < text.length) {
		if (text.startsWith("**", i)) {
			const end = text.indexOf("**", i + 2);
			if (end > 0) {
				flush();
				out.push({ kind: "strong", text: text.slice(i + 2, end) });
				i = end + 2;
				continue;
			}
		}
		if (text[i] === "*" && text[i + 1] !== "*") {
			const end = text.indexOf("*", i + 1);
			if (end > 0) {
				flush();
				out.push({ kind: "em", text: text.slice(i + 1, end) });
				i = end + 1;
				continue;
			}
		}
		if (text[i] === "`") {
			const end = text.indexOf("`", i + 1);
			if (end > 0) {
				flush();
				out.push({ kind: "code", text: text.slice(i + 1, end) });
				i = end + 1;
				continue;
			}
		}
		buf += text[i];
		i += 1;
	}
	flush();
	return out;
}

// Block tokenizer: headings, list items, paragraphs. Blank lines separate
// paragraphs.
export function parseBlocks(src: string): MdBlock[] {
	const out: MdBlock[] = [];
	const lines = src.split("\n");
	let para: string[] = [];
	const flushPara = () => {
		if (para.length) {
			out.push({ kind: "paragraph", inlines: parseInline(para.join(" ")) });
			para = [];
		}
	};
	for (const raw of lines) {
		const line = raw.trimEnd();
		if (line.trim() === "") {
			flushPara();
			out.push({ kind: "blank" });
			continue;
		}
		const h = /^(#{1,3})\s+(.*)$/.exec(line);
		if (h) {
			flushPara();
			out.push({
				kind: "heading",
				level: h[1].length as 1 | 2 | 3,
				inlines: parseInline(h[2]),
			});
			continue;
		}
		if (/^[-*]\s+/.test(line)) {
			flushPara();
			const item = line.slice(2);
			const last = out[out.length - 1];
			if (last && last.kind === "list") last.items.push(parseInline(item));
			else out.push({ kind: "list", items: [parseInline(item)] });
			continue;
		}
		para.push(line);
	}
	flushPara();
	return out;
}

// PSD splitter: the frontmatter block, the document title, and the ten fixed
// sections (## headings) with their markdown bodies. Section titles keep
// their numbering; bodies are raw markdown for per-section rendering.
export interface PsdSection {
	title: string;
	body: string;
}

export interface ParsedPsd {
	frontmatter: string;
	title: string;
	sections: PsdSection[];
}

export function parsePsd(psd: string): ParsedPsd {
	const out: ParsedPsd = { frontmatter: "", title: "", sections: [] };
	let rest = psd.trimStart();
	if (rest.startsWith("---")) {
		const end = rest.indexOf("\n---", 3);
		if (end > 0) {
			out.frontmatter = rest.slice(3, end).trim();
			rest = rest.slice(end + 4).trimStart();
		}
	}
	const titleMatch = /^#\s+(.+)$/m.exec(rest);
	if (titleMatch) {
		out.title = titleMatch[1].trim();
		rest = rest.slice(rest.indexOf(titleMatch[0]) + titleMatch[0].length);
	}
	// Sections: split on ## headings, keep the numbering in the title.
	const re = /^##\s+(.*)$/gm;
	const marks: { title: string; start: number; end: number }[] = [];
	let m: RegExpExecArray | null;
	while ((m = re.exec(rest)) !== null) {
		marks.push({
			title: m[1].trim(),
			start: m.index,
			end: m.index + m[0].length,
		});
	}
	for (let i = 0; i < marks.length; i += 1) {
		const bodyEnd = i + 1 < marks.length ? marks[i + 1].start : rest.length;
		out.sections.push({
			title: marks[i].title,
			body: rest.slice(marks[i].end, bodyEnd).trim(),
		});
	}
	return out;
}

// Download text as a markdown file — no dependencies, one object URL.
export function downloadMarkdown(filename: string, text: string): void {
	const blob = new Blob([text], { type: "text/markdown;charset=utf-8" });
	const url = URL.createObjectURL(blob);
	const a = document.createElement("a");
	a.href = url;
	a.download = filename;
	document.body.appendChild(a);
	a.click();
	a.remove();
	URL.revokeObjectURL(url);
}
