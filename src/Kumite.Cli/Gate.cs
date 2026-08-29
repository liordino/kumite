using System.Diagnostics;

namespace Kumite;

public enum GateChoice { Approve, Rerun, Edit }

public sealed record GateResult(GateChoice Choice, string Content);

/// <summary>
/// Human approval gate. This is the product: nothing reaches the wiki
/// without [a]pprove, [r]erun, or [e]dit.
/// </summary>
public sealed class Gate
{
    private readonly TextReader _input;
    private readonly TextWriter _output;

    public Gate(TextReader? input = null, TextWriter? output = null)
    {
        _input = input ?? Console.In;
        _output = output ?? Console.Out;
    }

    public GateResult Ask(string stepName, string draft)
    {
        _output.WriteLine();
        _output.WriteLine(new string('=', 70));
        _output.WriteLine($"GATE — {stepName}");
        _output.WriteLine(new string('=', 70));
        _output.WriteLine(draft);
        _output.WriteLine(new string('-', 70));

        while (true)
        {
            _output.Write("[a]pprove / [r]erun / [e]dit > ");
            var key = _input.ReadLine()?.Trim().ToLowerInvariant();
            switch (key)
            {
                case "a" or "approve":
                    return new GateResult(GateChoice.Approve, draft);
                case "r" or "rerun":
                    return new GateResult(GateChoice.Rerun, draft);
                case "e" or "edit":
                    var edited = OpenInEditor(draft);
                    if (string.IsNullOrWhiteSpace(edited))
                    {
                        _output.WriteLine("(editor returned empty — draft unchanged)");
                        continue;
                    }
                    draft = edited;
                    return new GateResult(GateChoice.Approve, draft);
                default:
                    _output.WriteLine("Please answer a, r, or e.");
                    break;
            }
        }
    }

    private string OpenInEditor(string content)
    {
        var file = Path.Combine(Path.GetTempPath(), $"kumite-gate-{Guid.NewGuid():N}.md");
        File.WriteAllText(file, content);
        var editor = Environment.GetEnvironmentVariable("EDITOR");
        if (string.IsNullOrWhiteSpace(editor))
        {
            _output.WriteLine($"(no $EDITOR set — edit this file, save it, then press Enter): {file}");
            _input.ReadLine();
            return File.ReadAllText(file);
        }
        var psi = new ProcessStartInfo(editor, $"\"{file}\"") { UseShellExecute = false };
        using var proc = Process.Start(psi)
            ?? throw new InvalidOperationException($"Could not start editor '{editor}'.");
        proc.WaitForExit();
        return File.ReadAllText(file);
    }
}
