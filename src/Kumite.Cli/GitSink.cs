using System.Diagnostics;

namespace Kumite;

/// <summary>
/// Git integration via shelling out to the git CLI (no libraries).
/// Kill switch (MVP.md): on failure, print suggested commands instead.
/// </summary>
public sealed class GitSink
{
    private readonly TextWriter _output;

    public GitSink(TextWriter? output = null) => _output = output ?? Console.Out;

    public bool CommitRun(string runId)
    {
        try
        {
            Run("add", "wiki/ trajectories/");
            var status = Run("status", "--porcelain");
            if (string.IsNullOrWhiteSpace(status.Stdout))
            {
                _output.WriteLine("git: nothing to commit (wiki/trajectories unchanged).");
                return true;
            }
            Run("commit", $"-m \"kumite run {runId}: approved wiki artifacts + trajectories\"");
            _output.WriteLine($"git: committed run {runId} artifacts.");
            return true;
        }
        catch (Exception ex)
        {
            _output.WriteLine($"git integration failed ({ex.Message}).");
            _output.WriteLine("Suggested manual commands:");
            _output.WriteLine("  git add wiki/ trajectories/");
            _output.WriteLine($"  git commit -m \"kumite run {runId}: approved wiki artifacts + trajectories\"");
            return false;
        }
    }

    private static (string Stdout, string Stderr) Run(string command, string args)
    {
        var psi = new ProcessStartInfo("git", $"{command} {args}")
        {
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false,
        };
        using var proc = Process.Start(psi)
            ?? throw new InvalidOperationException("git CLI not found on PATH.");
        var stdout = proc.StandardOutput.ReadToEnd();
        var stderr = proc.StandardError.ReadToEnd();
        proc.WaitForExit();
        if (proc.ExitCode != 0)
            throw new InvalidOperationException($"git {command} exit {proc.ExitCode}: {stderr.Trim()}");
        return (stdout, stderr);
    }
}
