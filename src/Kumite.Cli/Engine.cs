namespace Kumite;

/// <summary>
/// The state machine:
/// IDEA → ROUND1 (parallel) → GATE → ROUND2 (sequential) → GATE
///      → VERDICT → GATE → git add wiki/ + commit.
/// Supports the MVP kill switch: skip round 2 (--no-round2).
/// </summary>
public sealed class Engine
{
    private readonly Board _board;
    private readonly LlmClient _llm;
    private readonly Gate _gate;
    private readonly Wiki _wiki;
    private readonly TrajectoryLogger _trajectories;
    private readonly GitSink _git;
    private readonly bool _includeRound2;
    private readonly TextWriter _out;

    public Engine(Board board, LlmClient llm, Gate gate, Wiki wiki,
        TrajectoryLogger trajectories, GitSink git, bool includeRound2,
        TextWriter? output = null)
    {
        _board = board;
        _llm = llm;
        _gate = gate;
        _wiki = wiki;
        _trajectories = trajectories;
        _git = git;
        _includeRound2 = includeRound2;
        _out = output ?? Console.Out;
    }

    private BoardRound Round(string prefix) =>
        _board.Rounds.First(r => r.Id.StartsWith(prefix, StringComparison.OrdinalIgnoreCase));

    private async Task<string> RunPersonaAsync(BoardRound round, Persona persona, string prompt,
        string runId, CancellationToken ct)
    {
        var attempt = 0;
        while (true)
        {
            attempt++;
            _out.WriteLine($"  [{round.Id}/{persona.Id}] calling {persona.Model} (attempt {attempt})…");
            var call = await _llm.ChatAsync(persona.Model, persona.SystemPrompt, prompt, ct);
            var trajectory = _trajectories.Log(runId, round.Id, persona.Id, call, attempt);
            _out.WriteLine($"  trajectory: {trajectory}");
            var gate = _gate.Ask($"{round.Id} / {persona.Id}", call.Content());
            if (gate.Choice == GateChoice.Rerun)
                continue; // old trajectory kept, next attempt gets .attemptN.md
            return gate.Content;
        }
    }

    public async Task RunAsync(string runId, string idea, CancellationToken ct = default)
    {
        _out.WriteLine($"run {runId} — board '{_board.Name}'");
        var ideaPath = _wiki.WriteIdea(runId, idea);
        _out.WriteLine($"wiki: {ideaPath}");

        // ROUND 1 — parallel
        var round1 = Round("round-1");
        var r1 = await RunParallelRoundAsync(round1, runId,
            pid => PromptBuilder.ForRound1(idea), round1.PersonaIds, ct);
        _wiki.WriteRound(runId, round1.Id, r1);
        _out.WriteLine($"wiki: wiki/{round1.Id}-{runId}.md");

        // ROUND 2 — sequential
        var r2 = new List<(string, string)>();
        if (_includeRound2)
        {
            var round2 = Round("round-2");
            foreach (var pid in round2.PersonaIds)
            {
                var persona = _board.Persona(pid);
                var prompt = PromptBuilder.ForRound2(idea, r1, r2);
                var output = await RunPersonaAsync(round2, persona, prompt, runId, ct);
                r2.Add((pid, output));
            }
            _wiki.WriteRound(runId, round2.Id, r2);
            _out.WriteLine($"wiki: wiki/{round2.Id}-{runId}.md");
        }
        else
        {
            _out.WriteLine("round 2 skipped (--no-round2 kill switch).");
        }

        // VERDICT — chief sees everything
        var verdictRound = Round("verdict");
        var chief = _board.Persona(verdictRound.PersonaIds[0]);
        var verdict = await RunPersonaAsync(verdictRound, chief,
            PromptBuilder.ForVerdict(idea, r1, r2), runId, ct);
        _wiki.WriteVerdict(runId, verdict);
        _out.WriteLine($"wiki: wiki/verdict-{runId}.md");

        // Final: git add wiki/ + commit (kill switch: print commands on failure)
        _git.CommitRun(runId);
        _out.WriteLine($"run {runId} complete.");
    }

    private async Task<List<(string PersonaId, string Output)>> RunParallelRoundAsync(
        BoardRound round, string runId, Func<string, string> promptFor,
        IReadOnlyList<string> personaIds, CancellationToken ct)
    {
        // Launch all personas in parallel (Task.WhenAll). Each persona still
        // goes through its own gate before its result is committed.
        if (round.Mode != RoundMode.Parallel)
            throw new InvalidOperationException($"Round '{round.Id}' is not parallel.");

        var calls = personaIds.Select(async pid =>
        {
            var persona = _board.Persona(pid);
            var prompt = promptFor(pid);
            var attempt = 0;
            string currentOutput;
            // First call happens concurrently; gate/rerun loop stays sequential per persona.
            var pending = await _llm.ChatAsync(persona.Model, persona.SystemPrompt, prompt, ct);
            while (true)
            {
                attempt++;
                var trajectory = _trajectories.Log(runId, round.Id, pid, pending, attempt);
                _out.WriteLine($"  trajectory: {trajectory}");
                var gate = _gate.Ask($"{round.Id} / {pid}", pending.Content());
                if (gate.Choice == GateChoice.Rerun)
                {
                    _out.WriteLine($"  [{round.Id}/{pid}] rerunning (attempt {attempt + 1})…");
                    pending = await _llm.ChatAsync(persona.Model, persona.SystemPrompt, prompt, ct);
                    continue;
                }
                currentOutput = gate.Content;
                break;
            }
            return (pid, currentOutput);
        }).ToList();

        var results = await Task.WhenAll(calls);
        // Preserve board persona order in the wiki file.
        return personaIds.Select(pid => results.First(r => r.pid == pid)).ToList()!;
    }

    public static string NewRunId() =>
        $"{DateTime.UtcNow:yyyyMMdd-HHmmss}-{Guid.NewGuid():N}"[..23]; // date + 7 guid chars, unique
}
