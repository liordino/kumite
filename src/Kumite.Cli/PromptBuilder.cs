using System.Text;

namespace Kumite;

/// <summary>Builds per-persona user prompts. Pure functions — no LLM here.</summary>
public static class PromptBuilder
{
    public static string ForRound1(string idea) => $"""
        ## The idea under review

        {idea}

        Produce your critique following the output contract in your system prompt.
        """;

    public static string ForRound2(string idea, IReadOnlyList<(string PersonaId, string Output)> round1,
        IReadOnlyList<(string PersonaId, string Output)> priorRound2)
    {
        var sb = new StringBuilder();
        sb.Append("## The idea under review\n\n").Append(idea).AppendLine();
        sb.AppendLine("\n## Round 1 — independent critiques (all personas)");
        foreach (var (id, output) in round1)
            sb.Append("\n### ").Append(id).AppendLine().AppendLine(output);
        if (priorRound2.Count > 0)
        {
            sb.AppendLine("\n## Round 2 so far — responses before you");
            foreach (var (id, output) in priorRound2)
                sb.Append("\n### ").Append(id).AppendLine().AppendLine(output);
        }
        sb.AppendLine("\nRespond to the other personas: where are they wrong, where are they right, " +
                      "and what did everyone miss? Keep your own lens. Follow your output contract.");
        return sb.ToString();
    }

    public static string ForVerdict(string idea, IReadOnlyList<(string PersonaId, string Output)> round1,
        IReadOnlyList<(string PersonaId, string Output)> round2)
    {
        var sb = new StringBuilder();
        sb.Append("## The idea under review\n\n").Append(idea).AppendLine();
        sb.AppendLine("\n## Round 1 — independent critiques");
        foreach (var (id, output) in round1)
            sb.Append("\n### ").Append(id).AppendLine().AppendLine(output);
        sb.AppendLine("\n## Round 2 — cross-examination");
        foreach (var (id, output) in round2)
            sb.Append("\n### ").Append(id).AppendLine().AppendLine(output);
        sb.AppendLine("\nSynthesize the full debate into the final verdict per your output contract. " +
                      "Do not introduce analysis that is not grounded in the log above.");
        return sb.ToString();
    }

    public static string ForBaseline(string idea) => $"""
        You are asked to evaluate the following idea. Identify its flaws, assess its
        completeness as a spec, list concrete next actions, and consider it from
        multiple perspectives (business, technical, risks).

        ## The idea

        {idea}
        """;
}
