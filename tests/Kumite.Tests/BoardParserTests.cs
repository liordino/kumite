using Kumite;
using Xunit;

namespace Kumite.Tests;

public class BoardParserTests
{
    private const string MinimalBoard = """
        board_name: test_board
        description: test board
        personas:
          - id: alpha
            role: business
            model: m-a
            system_prompt: |
              Be alpha.
          - id: chief
            role: synthesis
            model: m-c
            system_prompt: |
              Be chief.
        rounds:
          - id: round-1
            mode: parallel
            personas: [alpha]
          - id: verdict
            mode: single
            personas: [chief]
        """;

    [Fact]
    public void Parses_personas_rounds_and_trims_prompts()
    {
        var board = BoardParser.Parse(MinimalBoard);

        Assert.Equal("test_board", board.Name);
        Assert.Equal(2, board.Personas.Count);
        Assert.Equal("m-a", board.Persona("alpha").Model);
        Assert.Equal("Be alpha.", board.Persona("alpha").SystemPrompt);
    }

    [Fact]
    public void Parses_all_round_modes()
    {
        var board = BoardParser.Parse("""
            board_name: b
            personas:
              - { id: a, role: r, model: m, system_prompt: p }
            rounds:
              - { id: r1, mode: parallel, personas: [a] }
              - { id: r2, mode: sequential, personas: [a] }
              - { id: r3, mode: single, personas: [a] }
            """);

        Assert.Equal(RoundMode.Parallel, board.Rounds[0].Mode);
        Assert.Equal(RoundMode.Sequential, board.Rounds[1].Mode);
        Assert.Equal(RoundMode.Single, board.Rounds[2].Mode);
    }

    [Fact]
    public void Rejects_unknown_round_mode()
    {
        var yaml = MinimalBoard.Replace("mode: parallel", "mode: chaotic");
        Assert.Throws<InvalidOperationException>(() => BoardParser.Parse(yaml));
    }

    [Fact]
    public void Rejects_unknown_persona_reference()
    {
        var yaml = MinimalBoard.Replace("personas: [alpha]", "personas: [ghost]");
        Assert.Throws<InvalidOperationException>(() => BoardParser.Parse(yaml));
    }

    [Fact]
    public void Parses_the_real_software_squad_board()
    {
        var path = FindRepoRoot("boards/software_squad.yaml");
        var board = BoardParser.LoadFile(path);

        Assert.Equal("software_squad", board.Name);
        Assert.Equal(4, board.Personas.Count);
        var ids = board.Personas.Select(p => p.Id).ToList();
        Assert.Equal(["product_owner", "architect", "reality_check", "chief"], ids);
        // Every drafted prompt must be real (no TODO placeholders, canary gone).
        Assert.All(board.Personas, p =>
        {
            Assert.DoesNotContain("TODO", p.SystemPrompt);
            Assert.True(p.SystemPrompt.Split(' ').Length <= 170,
                $"{p.Id} system prompt too long ({p.SystemPrompt.Split(' ').Length} words)");
            Assert.DoesNotContain("SET_ME", p.SystemPrompt);
        });
        // Round ids: round-1, round-2, verdict (typo 'verifierdict' fixed).
        Assert.Equal("verdict", board.Rounds.Single(r => r.Mode == RoundMode.Single).Id);
 Assert.Contains(board.Rounds, r => r.Id == "round-2" && r.Mode == RoundMode.Sequential);
    }

    [Fact]
    public void Reality_check_contract_requires_adversarial_lens()
    {
        var path = FindRepoRoot("boards/software_squad.yaml");
        var board = BoardParser.LoadFile(path);
        var rc = board.Persona("reality_check");
        Assert.Contains("3 concrete flaws", rc.SystemPrompt);
        Assert.Contains("must NOT", rc.SystemPrompt);
    }

    private static string FindRepoRoot(string relative)
    {
        var dir = new DirectoryInfo(AppContext.BaseDirectory);
        while (dir is not null && !File.Exists(Path.Combine(dir.FullName, relative)))
            dir = dir.Parent;
        Assert.NotNull(dir);
        return Path.Combine(dir!.FullName, relative);
    }
}