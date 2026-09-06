namespace VictoryLauncher;

/// <summary>
/// The tray icon and every window's title-bar icon should be Victory's
/// actual favicon, not a generic system icon -- the same asset every
/// page in the web app already uses (frontend/assets/favicon.png,
/// linked from every venue's &lt;head&gt;). Bundled as plain content here
/// and converted to a Windows Icon at runtime rather than maintaining a
/// separately-generated .ico file that could drift from the real asset.
/// </summary>
internal static class AppIcon
{
    private static Icon? _shared;

    public static Icon Shared => _shared ??= Load();

    private static Icon Load()
    {
        var path = Path.Combine(AppContext.BaseDirectory, "favicon.png");
        if (!File.Exists(path))
            return SystemIcons.Application; // shipped-but-missing asset shouldn't crash the app over an icon

        using var source = new Bitmap(path);
        // Windows scales a single reasonably-sized source down for the
        // tray (effectively 16-32px) and up for taskbar/alt-tab far
        // better than it upscales a tiny one -- 256 is the conventional
        // "give Windows a good source to work from" size, not a magic
        // number tied to any single display context.
        using var resized = new Bitmap(source, new Size(256, 256));
        var handle = resized.GetHicon();
        return Icon.FromHandle(handle); // caller (Shared) keeps this alive for the process lifetime; never disposed/DestroyIcon'd, an acceptable one-time leak for a single long-lived app icon
    }
}
