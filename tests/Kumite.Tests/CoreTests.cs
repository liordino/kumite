using Kumite;
using Xunit;

namespace Kumite.Tests;

public class IdeaSourceTests : IDisposable
{
    private readonly string _tmp = Path.Combine(Path.GetTempPath(), $"kumite-test-{Guid.NewGuid():N}");

    [Fact]
    public void Literal_text_passes_through()
    {
        Assert.Equal("some idea text", IdeaSource.Resolve("some idea text"));
    }

    [Fact]
    public void File_path_is_read_and_trimmed()
    {
        Directory.CreateDirectory(_tmp);
        var file = Path.Combine(_tmp, "idea.md");
        File.WriteAllText(file, "  Andante idea\n");
        Assert.Equal("Andante idea", IdeaSource.Resolve(file));
    }

    public void Dispose()
    {
        if (Directory.Exists(_tmp))
            Directory.Delete(_tmp, true);
    }
}

public class PromptBuilderTests
{
    private static readonly (string, string)[] R1 = [("po", "PO output"), ("arch", "ARCH output")];
    private static readonly (string, string)[] R2 = [("po", "PO response")];

    [Fact]
    public void Round1_contains_idea()
    {
        var prompt = PromptBuilder.ForRound1("Andante");
        Assert.Contains("Andante", prompt);
        Assert.Contains("idea under review", prompt);
    }

    [Fact]
    public void Round2_includes_idea_and_all_round1_outputs_and_prior_round2()
    {
        var prompt = PromptBuilder.ForRound2("Andante", R1, R2);
        Assert.Contains("Andante", prompt);
        Assert.Contains("PO output", prompt);
        Assert.Contains("ARCH output", prompt);
        Assert.Contains("PO response", prompt);
        Assert.Contains("responses before you", prompt);
    }

    [Fact]
    public void Round2_without_prior_responses_omits_that_section()
    {
        var prompt = PromptBuilder.ForRound2("Andante", R1, []);
        Assert.DoesNotContain("responses before you", prompt);
    }

    [Fact]
    public void Verdict_includes_everything()
    {
        var prompt = PromptBuilder.ForVerdict("Andante", R1, R2);
        Assert.Contains("Andante", prompt);
        Assert.Contains("PO output", prompt);
        Assert.Contains("PO response", prompt);
        Assert.Contains("cross-examination", prompt);
    }

    [Fact]
    public void Baseline_keeps_plain_single_prompt_scope()
    {
        var prompt = PromptBuilder.ForBaseline("Andante");
        Assert.Contains("Andante", prompt);
        Assert.Contains("multiple perspectives", prompt);
        // Deliberately weaker scaffold than persona prompts — that is the experiment.
        Assert.DoesNotContain("output contract", prompt, StringComparison.OrdinalIgnoreCase);
    }
}

public class EngineTests
{
    [Fact]
    public void NewRunId_is_filesystem_safe_and_unique()
    {
        var a = Engine.NewRunId();
        var b = Engine.NewRunId();
        Assert.NotEqual(a, b);
        Assert.Matches("^[a-zA-Z0-9\\-]+$", a);
        Assert.True(a.Length < 30);
    }
}

public class ConfigTests
{
    [Fact]
    public void Throws_when_env_file_missing()
    {
        var missing = Path.Combine(Path.GetTempPath(), $"kumite-test-{Guid.NewGuid():N}", "nope.env");
        Assert.Throws<InvalidOperationException>(() => Config.Load(missing));
    }

    [Fact]
    public void Parses_env_file_and_trims_trailing_slash()
    {
        var tmp = Path.Combine(Path.GetTempPath(), $"kumite-test-{Guid.NewGuid():N}");
        Directory.CreateDirectory(tmp);
        try
        {
            var env = Path.Combine(tmp, ".env");
            File.WriteAllLines(env,
                ["# comment", "KUMITE_BASE_URL=https://api.example.com/v1/", "KUMITE_API_KEY=sk-test"]);
            var config = Config.Load(env);
            Assert.Equal("https://api.example.com/v1", config.BaseUrl);
            Assert.Equal("sk-test", config.ApiKey);
        }
        finally
        {
            Directory.Delete(tmp, true);
        }
    }
}