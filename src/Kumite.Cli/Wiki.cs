namespace Kumite;

/// <summary>Writes the LLMWiki markdown artifacts. Paths are part of the spec.</summary>
public sealed class Wiki
{
    private readonly string _root;

    public Wiki(string root = "wiki") => _root = root;

    private string Write(string fileName, string content)
    {
        Directory.CreateDirectory(_root);
        var path = Path.Combine(_root, fileName);
        File.WriteAllText(path, content);
        return path;
    }

    /// <summary>The original idea, verbatim. Written once per run.</summary>
    public string WriteIdea(string runId, string idea) =>
        Write($"idea-{runId}.md", $"# Idea (run {runId})\n\n{idea}\n");

    public string WriteRound(string runId, string roundId,
        IReadOnlyList<(string PersonaId, string Output)> outputs)
    {
        var parts = new List<string> { $"# {roundId} (run {runId})\n" };
        foreach (var (id, output) in outputs)
            parts.Add($"\n## {id}\n\n{output}\n");
        return Write($"{roundId}-{runId}.md", string.Concat(parts));
    }

    public string WriteVerdict(string runId, string verdict) =>
        Write($"verdict-{runId}.md", $"# Verdict (run {runId})\n\n{verdict}\n");

    public string WriteBaseline(string content) =>
        Write("baseline-result.md", $"# Baseline result\n\n{content}\n");
}
