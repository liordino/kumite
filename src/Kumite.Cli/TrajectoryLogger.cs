namespace Kumite;

/// <summary>
/// Writes FULL request + raw response for every LLM call, no exceptions.
/// trajectories/{run-id}/{round}/{persona}.md — a deliverable, stays git-tracked.
/// </summary>
public sealed class TrajectoryLogger
{
    private readonly string _root;

    public TrajectoryLogger(string root = "trajectories") => _root = root;

    public string Log(string runId, string round, string personaId, LlmCall call, int attempt = 1)
    {
        var dir = Path.Combine(_root, runId, round);
        Directory.CreateDirectory(dir);
        var fileName = attempt <= 1 ? $"{personaId}.md" : $"{personaId}.attempt{attempt}.md";
        var path = Path.Combine(dir, fileName);
        File.WriteAllText(path, $"""
            # Trajectory — {round} / {personaId} (attempt {attempt})

            run-id: {runId}
            timestamp (UTC): {DateTime.UtcNow:O}

            ## Request (verbatim JSON)

            ```json
            {call.RequestJson}
            ```

            ## Raw response (verbatim JSON)

            ```json
            {call.RawResponseJson}
            ```

            ## Extracted assistant content

            {call.Content()}
            """);
        return path;
    }
}
