namespace Kumite;

/// <summary>
/// Loads configuration from environment variables, with a .env file as
/// fallback (no dependencies; .env is never committed).
/// </summary>
public sealed record Config(string BaseUrl, string ApiKey)
{
    public const string BaseUrlVar = "KUMITE_BASE_URL";
    public const string ApiKeyVar = "KUMITE_API_KEY";

    public static Config Load(string? envFilePath = null)
    {
        var values = new Dictionary<string, string>(StringComparer.Ordinal);
        envFilePath ??= Path.Combine(Directory.GetCurrentDirectory(), ".env");
        if (File.Exists(envFilePath))
        {
            foreach (var line in File.ReadAllLines(envFilePath))
            {
                var trimmed = line.Trim();
                if (trimmed.Length == 0 || trimmed.StartsWith('#'))
                    continue;
                var idx = trimmed.IndexOf('=');
                if (idx <= 0)
                    continue;
                values[trimmed[..idx].Trim()] = trimmed[(idx + 1)..].Trim();
            }
        }

        string Get(string name) =>
            Environment.GetEnvironmentVariable(name)
            ?? (values.TryGetValue(name, out var v) ? v : "")
            ?? "";

        var baseUrl = Get(BaseUrlVar);
        var apiKey = Get(ApiKeyVar);
        if (string.IsNullOrWhiteSpace(baseUrl))
            throw new InvalidOperationException($"{BaseUrlVar} not set (.env or environment).");
        if (string.IsNullOrWhiteSpace(apiKey))
            throw new InvalidOperationException($"{ApiKeyVar} not set (.env or environment).");
        return new Config(baseUrl.TrimEnd('/'), apiKey);
    }
}
