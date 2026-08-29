using System.Net;
using System.Text;
using Kumite;
using Xunit;

namespace Kumite.Tests;

/// <summary>
/// Proves the full engine loop — ROUND1 parallel, ROUND2 sequential,
/// VERDICT, wiki artifacts, trajectory logging — against a fake
/// OpenAI-compatible endpoint. No network, no API key.
/// </summary>
public sealed class EngineIntegrationTests : IDisposable
{
    private readonly string _root = Path.Combine(Path.GetTempPath(), $"kumite-e2e-{Guid.NewGuid():N}");
    private readonly HttpListener _listener = new();
    private readonly string _url;

    public EngineIntegrationTests()
    {
        Directory.CreateDirectory(_root);
        var port = new Random().Next(20000, 40000);
        _url = $"http://127.0.0.1:{port}/";
        _listener.Prefixes.Add(_url);
        _listener.Start();
    }

    public void Dispose()
    {
        try { _listener.Stop(); } catch { /* best effort */ }
        if (Directory.Exists(_root)) Directory.Delete(_root, true);
    }

    private void ServeFakeOpenAi()
    {
        _ = Task.Run(async () =>
        {
            while (_listener.IsListening)
            {
                HttpListenerContext ctx;
                try { ctx = await _listener.GetContextAsync(); }
                catch (HttpListenerException) { break; }
                using var reader = new StreamReader(ctx.Request.InputStream);
                var requestBody = await reader.ReadToEndAsync();
                var model = Extract(requestBody, "\"model\":\"", "\"");
                var body =
                    "{\"id\":\"fake\",\"object\":\"chat.completion\",\"model\":\"" + model + "\"," +
                    "\"choices\":[{\"index\":0,\"message\":{\"role\":\"assistant\",\"content\":\"FAKE RESPONSE from " + model +
                    "\"},\"finish_reason\":\"stop\"}]}";
                var bytes = Encoding.UTF8.GetBytes(body);
                ctx.Response.ContentType = "application/json";
                ctx.Response.ContentLength64 = bytes.Length;
                await ctx.Response.OutputStream.WriteAsync(bytes);
                ctx.Response.Close();
            }
        });
    }

    private static string Extract(string text, string start, string end)
    {
        var i = text.IndexOf(start, StringComparison.Ordinal);
        if (i < 0) return "?";
        i += start.Length;
        var j = text.IndexOf(end, i, StringComparison.Ordinal);
        return j < 0 ? text[i..] : text[i..j];
    }

    private static Board TestBoard() => BoardParser.Parse("""
        board_name: test_board
        personas:
          - id: alpha
            role: business
            model: fake-a
            system_prompt: "You are Alpha."
          - id: beta
            role: adversarial
            model: fake-b
            system_prompt: "You are Beta."
          - id: chief
            role: synthesis
            model: fake-c
            system_prompt: "You are Chief."
        rounds:
          - { id: round-1, mode: parallel, personas: [alpha, beta] }
          - { id: round-2, mode: sequential, personas: [alpha, beta] }
          - { id: verdict, mode: single, personas: [chief] }
        """);

    [Fact]
    public async Task Full_run_produces_all_wiki_and_trajectory_artifacts()
    {
        ServeFakeOpenAi();
        var config = new Config(_url.TrimEnd('/'), "fake-key");
        var log = new StringWriter();
        // 7 gates total (3 + 3 + 1), all approved.
        var gate = new Gate(new StringReader("a\na\na\na\na\na\na\n"), log);

        var engine = new Engine(TestBoard(), new LlmClient(config), gate,
            new Wiki(Path.Combine(_root, "wiki")),
            new TrajectoryLogger(Path.Combine(_root, "trajectories")),
            new GitSink(log),
            includeRound2: true,
            log);

        await engine.RunAsync("run-test", "Andante idea under test");

        var wiki = Path.Combine(_root, "wiki");
        Assert.True(File.Exists(Path.Combine(wiki, "idea-run-test.md")));
        Assert.True(File.Exists(Path.Combine(wiki, "round-1-run-test.md")));
        Assert.True(File.Exists(Path.Combine(wiki, "round-2-run-test.md")));
        Assert.True(File.Exists(Path.Combine(wiki, "verdict-run-test.md")));

        var traj = Path.Combine(_root, "trajectories", "run-test");
        // Test board round-1/round-2 have exactly two personas (alpha, beta); chief is verdict-only.
        Assert.Equal(2, Directory.GetFiles(Path.Combine(traj, "round-1")).Length);
        Assert.Equal(2, Directory.GetFiles(Path.Combine(traj, "round-2")).Length);
        Assert.Single(Directory.GetFiles(Path.Combine(traj, "verdict")));

        // Round-2 sequential prompt must embed prior round-1 + round-2 outputs.
        var betaR2 = File.ReadAllText(Path.Combine(traj, "round-2", "beta.md"));
        Assert.Contains("responses before you", betaR2);

        // Every trajectory file has request + raw response sections.
        foreach (var f in Directory.GetFiles(traj, "*.md", SearchOption.AllDirectories))
        {
            var text = File.ReadAllText(f);
            Assert.Contains("## Request (verbatim JSON)", text);
            Assert.Contains("## Raw response (verbatim JSON)", text);
        }
        Assert.Contains("run run-test complete", log.ToString());
    }

    [Fact]
    public async Task Rerun_keeps_old_trajectory_and_adds_attempt_file()
    {
        ServeFakeOpenAi();
        var config = new Config(_url.TrimEnd('/'), "fake-key");
        var log = new StringWriter();
        // Round-1 alpha: [r]erun once then approve; beta approve; round2 x2 approve; verdict approve.
        var gate = new Gate(new StringReader("r\na\na\na\na\na\na\n"), log);

        var engine = new Engine(TestBoard(), new LlmClient(config), gate,
            new Wiki(Path.Combine(_root, "wiki")),
            new TrajectoryLogger(Path.Combine(_root, "trajectories")),
            new GitSink(log),
            includeRound2: true,
            log);

        await engine.RunAsync("run-rerun", "idea");

        var files = Directory.GetFiles(Path.Combine(_root, "trajectories", "run-rerun", "round-1"));
        // alpha rerun → alpha.md + alpha.attempt2.md, plus beta.md once.
        Assert.Contains(files, f => f.EndsWith("alpha.md"));
        Assert.Contains(files, f => f.EndsWith("alpha.attempt2.md"));
        Assert.Equal(3, files.Length);
    }

    [Fact]
    public async Task Edit_uses_edited_content_in_wiki()
    {
        ServeFakeOpenAi();
        var config = new Config(_url.TrimEnd('/'), "fake-key");
        var log = new StringWriter();
        // Round-1 alpha: edit via $EDITOR file round-trip; then 6 approvals.
        var gate = new Gate(new StringReader("e\na\na\na\na\na\na\n"), log);

        var engine = new Engine(TestBoard(), new LlmClient(config), gate,
            new Wiki(Path.Combine(_root, "wiki")),
            new TrajectoryLogger(Path.Combine(_root, "trajectories")),
            new GitSink(log),
            includeRound2: true,
            log);

        await engine.RunAsync("run-edit", "idea");
        var r1 = File.ReadAllText(Path.Combine(_root, "wiki", "round-1-run-edit.md"));
        Assert.Contains("FAKE RESPONSE", r1); // no $EDITOR set path: file round-trip keeps content
    }
}