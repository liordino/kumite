using YamlDotNet.Serialization;
using YamlDotNet.Serialization.NamingConventions;

namespace Kumite;

public sealed record Persona(string Id, string Role, string Model, string SystemPrompt);

public enum RoundMode { Parallel, Sequential, Single }

public sealed record BoardRound(string Id, RoundMode Mode, IReadOnlyList<string> PersonaIds);

public sealed record Board(string Name, string Description, IReadOnlyList<Persona> Personas, IReadOnlyList<BoardRound> Rounds)
{
    public Persona Persona(string id) =>
        Personas.FirstOrDefault(p => p.Id == id)
        ?? throw new InvalidOperationException($"Board '{Name}' references unknown persona '{id}'.");
}

public static class BoardParser
{
    // Private shapes mirroring the YAML file.
    private sealed class RootDto
    {
        public string BoardName { get; set; } = "";
        public string Description { get; set; } = "";
        public List<PersonaDto> Personas { get; set; } = [];
        public List<RoundDto> Rounds { get; set; } = [];
    }

    private sealed class PersonaDto
    {
        public string Id { get; set; } = "";
        public string Role { get; set; } = "";
        public string Model { get; set; } = "";
        public string SystemPrompt { get; set; } = "";
    }

    private sealed class RoundDto
    {
        public string Id { get; set; } = "";
        public string Mode { get; set; } = "";
        public List<string> Personas { get; set; } = [];
    }

    public static Board Parse(string yaml)
    {
        var deserializer = new DeserializerBuilder()
            .WithNamingConvention(UnderscoredNamingConvention.Instance)
            .IgnoreUnmatchedProperties()
            .Build();
        var dto = deserializer.Deserialize<RootDto>(yaml)
            ?? throw new InvalidOperationException("Board file is empty or invalid YAML.");

        // YamlDotNet deserializes empty flow sequences ([]) as null.
        var personas = (dto.Personas ?? []).Select(p =>
            new Persona(p.Id, p.Role, p.Model, (p.SystemPrompt ?? "").Trim())).ToList();

        var knownIds = personas.Select(p => p.Id).ToHashSet();
        var rounds = (dto.Rounds ?? []).Select(r =>
        {
            foreach (var pid in r.Personas ?? [])
            {
                if (!knownIds.Contains(pid))
                    throw new InvalidOperationException(
                        $"Round '{r.Id}' references unknown persona '{pid}'.");
            }
            var mode = r.Mode.ToLowerInvariant() switch
            {
                "parallel" => RoundMode.Parallel,
                "sequential" => RoundMode.Sequential,
                "single" => RoundMode.Single,
                var other => throw new InvalidOperationException(
                    $"Round '{r.Id}' has unknown mode '{other}'."),
            };
            return new BoardRound(r.Id, mode, r.Personas ?? []);
        }).ToList();

        if (personas.Count == 0 || rounds.Count == 0)
            throw new InvalidOperationException("Board must define at least one persona and one round.");

        return new Board(dto.BoardName, dto.Description.Trim(), personas, rounds);
    }

    public static Board LoadFile(string path) => Parse(File.ReadAllText(path));
}
