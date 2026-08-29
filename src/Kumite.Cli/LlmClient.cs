using System.Text;
using System.Text.Json;

namespace Kumite;

public sealed record LlmCall(string RequestJson, string RawResponseJson)
{
    /// <summary>Extracts the assistant message content from the raw response.</summary>
    public string Content()
    {
        using var doc = JsonDocument.Parse(RawResponseJson);
        var choices = doc.RootElement.GetProperty("choices");
        var message = choices[0].GetProperty("message");
        return message.GetProperty("content").GetString() ?? "";
    }
}

/// <summary>Minimal OpenAI-compatible chat-completions client.</summary>
public sealed class LlmClient
{
    private readonly Config _config;
    private readonly HttpClient _http;

    public LlmClient(Config config, HttpClient? http = null)
    {
        _config = config;
        _http = http ?? new HttpClient();
        // Reasoning/thinking models (e.g. 120b+ on Ollama Cloud) can run well
        // over HttpClient's default 100 s; 5 min covers them without hanging forever.
        _http.Timeout = TimeSpan.FromMinutes(5);
        _http.DefaultRequestHeaders.TryAddWithoutValidation("Authorization", $"Bearer {config.ApiKey}");
    }

    public async Task<LlmCall> ChatAsync(string model, string systemPrompt, string userPrompt,
        CancellationToken ct = default)
    {
        var payload = new
        {
            model,
            messages = new object[]
            {
                new { role = "system", content = systemPrompt },
                new { role = "user", content = userPrompt },
            },
        };
        var requestJson = JsonSerializer.Serialize(payload, new JsonSerializerOptions { WriteIndented = true });

        using var request = new HttpRequestMessage(HttpMethod.Post, $"{_config.BaseUrl}/chat/completions")
        {
            Content = new StringContent(JsonSerializer.Serialize(payload), Encoding.UTF8, "application/json"),
        };
        using var response = await _http.SendAsync(request, ct);
        var body = await response.Content.ReadAsStringAsync(ct);
        if (!response.IsSuccessStatusCode)
            throw new InvalidOperationException(
                $"LLM API error {(int)response.StatusCode}: {body[..Math.Min(body.Length, 500)]}");

        // Keep the raw response verbatim (pretty-printed JSON, no content loss).
        string raw;
        try
        {
            using var doc = JsonDocument.Parse(body);
            raw = JsonSerializer.Serialize(doc.RootElement, new JsonSerializerOptions { WriteIndented = true });
        }
        catch (JsonException)
        {
            raw = body;
        }
        return new LlmCall(requestJson, raw);
    }
}
