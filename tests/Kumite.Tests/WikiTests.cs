using Kumite;
using Xunit;

namespace Kumite.Tests;

public class WikiTests
{
    [Fact]
    public void Writes_expected_artifact_paths()
    {
        var root = Path.Combine(Path.GetTempPath(), $"kumite-wiki-{Guid.NewGuid():N}");
        try
        {
            var wiki = new Wiki(root);
            wiki.WriteIdea("run1", "Andante");
            wiki.WriteRound("run1", "round-1", [("po", "out")]);
            wiki.WriteVerdict("run1", "ship it");
            Assert.True(File.Exists(Path.Combine(root, "idea-run1.md")));
            Assert.True(File.Exists(Path.Combine(root, "round-1-run1.md")));
            Assert.True(File.Exists(Path.Combine(root, "verdict-run1.md")));
            Assert.Contains("Andante", File.ReadAllText(Path.Combine(root, "idea-run1.md")));
        }
        finally
        {
            if (Directory.Exists(root)) Directory.Delete(root, true);
        }
    }
}