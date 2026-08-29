namespace Kumite;

/// <summary>Idea is either literal text or a path to a text file.</summary>
public static class IdeaSource
{
    public static string Resolve(string textOrPath)
    {
        if (File.Exists(textOrPath))
            return File.ReadAllText(textOrPath).Trim();
        return textOrPath;
    }
}
