using Kumite;

// kumite — MVP CLI (see MVP.md). Modes: init | run | baseline.
var output = Console.Out;

if (args.Length == 0)
{
    output.WriteLine("""
        usage:
          kumite init                                          scaffold wiki/, trajectories/, boards/, .env.example, .gitignore
          kumite run --board <name> --idea <text-or-file>      full debate (gates at every step)
          kumite run --board <name> --idea <...> --auto        unattended mode: auto-approve every gate
          kumite run --board <name> --idea <...> --no-round2   kill switch: skip round 2
          kumite baseline --idea <text-or-file>                single LLM prompt, no debate
        """);
    return 2;
}

var command = args[0];
try
{
    switch (command)
    {
        case "init":
            return Init(output);
        case "run":
            return await RunAsync(args[1..], output);
        case "baseline":
            return await BaselineAsync(args[1..], output);
        default:
            output.WriteLine($"unknown command '{command}'. Try: init | run | baseline");
            return 2;
    }
}
catch (Exception ex) when (ex is InvalidOperationException or FileNotFoundException
    or HttpRequestException or TaskCanceledException)
{
    Console.Error.WriteLine($"error: {ex.Message}");
    return 1;
}

static int Init(TextWriter output)
{
    Directory.CreateDirectory("wiki");
    Directory.CreateDirectory("trajectories");
    if (!File.Exists(".env.example"))
        File.WriteAllText(".env.example",
            "# OpenAI-compatible endpoint. Examples:\n" +
            "#   https://openrouter.ai/api/v1   |   http://localhost:4000/v1 (LiteLLM)\n" +
            "KUMITE_BASE_URL=https://openrouter.ai/api/v1\nKUMITE_API_KEY=sk-or-...\n");
    if (!File.Exists(".gitignore"))
        File.WriteAllText(".gitignore", ".env\nbin/\nobj/\n");
    if (!File.Exists(".env"))
        File.Copy(".env.example", ".env");
    if (!Directory.Exists("boards") || Directory.GetFiles("boards", "*.yaml").Length == 0)
    {
        Directory.CreateDirectory("boards");
        // Placeholder; real board ships with the repo and is not overwritten.
        File.WriteAllText(Path.Combine("boards", "software_squad.yaml"),
            "# Create boards/software_squad.yaml here (see repo README).\n");
    }
    output.WriteLine("initialized: wiki/, trajectories/, boards/, .env.example, .env (fill it in).");
    return 0;
}

static (string? Board, string? Idea, bool NoRound2, bool Auto) ParseArgs(string[] args)
{
    string? board = null, idea = null;
    var noRound2 = false;
    var auto = false;
    for (var i = 0; i < args.Length; i++)
    {
        switch (args[i])
        {
            case "--board":
                board = args[++i];
                break;
            case "--idea":
                idea = args[++i];
                break;
            case "--no-round2":
                noRound2 = true;
                break;
            case "--auto":
                auto = true;
                break;
            default:
                throw new InvalidOperationException($"unknown argument '{args[i]}'");
        }
    }
    return (board, idea, noRound2, auto);
}

static (string BoardPath, string Idea) RequireRunInputs(string? board, string? idea)
{
    if (string.IsNullOrWhiteSpace(board))
        throw new InvalidOperationException("--board is required");
    if (string.IsNullOrWhiteSpace(idea))
        throw new InvalidOperationException("--idea is required (text or file path)");
    var boardPath = File.Exists(board) ? board : Path.Combine("boards", $"{board}.yaml");
    if (!File.Exists(boardPath))
        throw new FileNotFoundException($"board not found: {boardPath}");
    return (boardPath, IdeaSource.Resolve(idea));
}

static async Task<int> RunAsync(string[] args, TextWriter output)
{
    var (board, idea, noRound2, auto) = ParseArgs(args);
    var (boardPath, resolvedIdea) = RequireRunInputs(board, idea);

    var config = Config.Load();
    var parsed = BoardParser.LoadFile(boardPath);
    var engine = new Engine(parsed,
        new LlmClient(config),
        new Gate(auto: auto),
        new Wiki(),
        new TrajectoryLogger(),
        new GitSink(),
        includeRound2: !noRound2,
        output);

    await engine.RunAsync(Engine.NewRunId(), resolvedIdea);
    return 0;
}

static async Task<int> BaselineAsync(string[] args, TextWriter output)
{
    var (_, idea, _, _) = ParseArgs(args);
    if (string.IsNullOrWhiteSpace(idea))
        throw new InvalidOperationException("--idea is required (text or file path)");
    var resolvedIdea = IdeaSource.Resolve(idea);

    var config = Config.Load();
    // Baseline model: first persona's model is NOT right — use a plain strong model.
    // The board's chief model is the single-model comparison point.
    var board = BoardParser.LoadFile(Path.Combine("boards", "software_squad.yaml"));
    var chief = board.Persona("chief");
    output.WriteLine($"baseline: single call to {chief.Model} (no debate)…");
    var call = await new LlmClient(config).ChatAsync(chief.Model,
        "You are a senior evaluator. Be thorough, concrete, and critical.",
        PromptBuilder.ForBaseline(resolvedIdea));
    var path = new Wiki().WriteBaseline(call.Content());
    output.WriteLine($"baseline saved: {path}");
    return 0;
}