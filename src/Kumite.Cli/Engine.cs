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
        // Launch all persona calls concurrently (Task.WhenAll); then gate
        // them sequentially in board order so the console stays readable.
        if (round.Mode != RoundMode.Parallel)
            throw new InvalidOperationException($"Round '{round.Id}' is not parallel.");

        var pending = new Dictionary<string, Task<LlmCall>>();
        foreach (var pid in personaIds)
        {
            var persona = _board.Persona(pid);
            pending[pid] = _llm.ChatAsync(persona.Model, persona.SystemPrompt, promptFor(pid), ct);
        }

        var results = new List<(string PersonaId, string Output)>();
        foreach (var pid in personaIds)
        {
            var persona = _board.Persona(pid);
            var attempt = 0;
            var call = await pending[pid];
            while (true)
            {
                attempt++;
                var trajectory = _trajectories.Log(runId, round.Id, pid, call, attempt);
                _out.WriteLine($"  trajectory: {trajectory}");
                var gate = _gate.Ask($"{round.Id} / {pid}", call.Content());
                if (gate.Choice == GateChoice.Rerun)
                {
                    _out.WriteLine($"  [{round.Id}/{pid}] rerunning (attempt {attempt + 1})…");
                    call = await _llm.ChatAsync(persona.Model, persona.SystemPrompt, promptFor(pid), ct);
                    continue; // old trajectory kept; next attempt gets .attemptN.md
                }
                results.Add((pid, gate.Content));
                break;
            }
        }
        return results;
    }

    public static string NewRunId() =>
        $"{DateTime.UtcNow:yyyyMMdd-HHmmss}-{Guid.NewGuid():N}"[..23]; // date + 7 guid chars, unique
}
