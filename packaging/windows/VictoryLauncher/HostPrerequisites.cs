namespace VictoryLauncher;

/// <summary>
/// Checked once, before the first-run wizard ever opens: a machine that
/// can't actually run Victory should say so plainly up front, not after
/// downloading 150+MB of Postgres only to fail obscurely partway through
/// initdb or a Caddy download with no disk left to extract it to.
/// </summary>
internal static class HostPrerequisites
{
    // Covers Postgres + Caddy + cloudflared downloads (a few hundred MB
    // combined) plus real headroom for a database that actually grows --
    // not a tight minimum, just enough that "ran out of disk mid-Show"
    // isn't the first time this gets checked.
    private const long MinFreeDiskBytes = 2L * 1024 * 1024 * 1024;

    // Windows 10 1809 (build 17763): the floor .NET 8's own support policy
    // documents for a self-contained win-x64 publish, which is what this
    // launcher ships as.
    private const int MinWindowsBuildNumber = 17763;

    public readonly record struct CheckResult(bool Ok, string? Detail);

    public static CheckResult Check()
    {
        var problems = new List<string>();

        if (Environment.OSVersion.Platform != PlatformID.Win32NT || Environment.OSVersion.Version.Build < MinWindowsBuildNumber)
        {
            problems.Add(
                "Victory needs Windows 10 (version 1809) or newer. This machine is reporting: " +
                Environment.OSVersion.VersionString);
        }

        try
        {
            var root = Path.GetPathRoot(AppPaths.DataRoot);
            if (!string.IsNullOrEmpty(root))
            {
                var drive = new DriveInfo(root);
                if (drive.AvailableFreeSpace < MinFreeDiskBytes)
                {
                    problems.Add(
                        $"Victory needs at least {MinFreeDiskBytes / 1024 / 1024 / 1024} GB free on {drive.Name} to download and run " +
                        $"(only {drive.AvailableFreeSpace / 1024.0 / 1024 / 1024:F1} GB is available).");
                }
            }
        }
        catch
        {
            // Best effort -- an inability to check free space isn't itself a reason to block setup.
        }

        return problems.Count == 0
            ? new CheckResult(true, null)
            : new CheckResult(false, string.Join("\n\n", problems));
    }
}
