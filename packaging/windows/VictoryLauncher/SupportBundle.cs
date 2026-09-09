using System.IO.Compression;

namespace VictoryLauncher;

/// <summary>
/// An Operator with a real problem needs to hand someone (Grant, a future
/// support channel) enough evidence to actually diagnose it -- confirmed
/// painful on real hardware during the .37 update-retry-loop investigation,
/// where that meant walking through %LocalAppData%\Victory\logs by hand and
/// attaching five separate files one at a time. One button, one zip, dropped
/// somewhere obvious (the Desktop) instead of a folder the Operator has to
/// go find.
/// </summary>
internal static class SupportBundle
{
    /// <summary>
    /// Returns the created zip's path, or null if it couldn't be created at
    /// all (nothing left to show the Operator in that case beyond a plain
    /// failure -- StatusForm surfaces that itself).
    /// </summary>
    public static string? Create()
    {
        var stagingDir = Path.Combine(Path.GetTempPath(), "victory-support-bundle-" + Guid.NewGuid().ToString("N"));
        try
        {
            Directory.CreateDirectory(stagingDir);
            File.WriteAllText(Path.Combine(stagingDir, "manifest.txt"), BuildManifest());

            // Whichever of these exist -- a fresh install may not have a
            // cloudflared.log yet, and launcher.log only exists once
            // something has actually gone wrong (UpdateChecker.LogApplyFailure,
            // Program.cs's global exception handler). Missing ones are just
            // silently skipped, not a reason to fail the whole bundle.
            foreach (var logFile in new[]
            {
                AppPaths.PostgresLogFile,
                AppPaths.BackendLogFile,
                AppPaths.CaddyLogFile,
                AppPaths.CloudflaredLogFile,
                Path.Combine(AppPaths.LogsDir, "launcher.log"),
            })
            {
                if (File.Exists(logFile))
                {
                    try { File.Copy(logFile, Path.Combine(stagingDir, Path.GetFileName(logFile)), overwrite: true); }
                    catch { /* a locked/in-use log file shouldn't block the rest of the bundle */ }
                }
            }

            var desktopDir = Environment.GetFolderPath(Environment.SpecialFolder.DesktopDirectory);
            var zipPath = Path.Combine(desktopDir, $"victory-support-bundle-{DateTime.Now:yyyyMMdd-HHmmss}.zip");
            ZipFile.CreateFromDirectory(stagingDir, zipPath);
            return zipPath;
        }
        catch
        {
            return null;
        }
        finally
        {
            try { Directory.Delete(stagingDir, recursive: true); } catch { /* best effort */ }
        }
    }

    /// <summary>
    /// Plain values, not raw .env contents -- POSTGRES_PASSWORD and the
    /// Cloudflare/GitHub tokens live in the same file and must never end up
    /// in something an Operator might paste into a support channel. Only
    /// the handful of fields actually useful for diagnosis are read out
    /// individually.
    /// </summary>
    private static string BuildManifest()
    {
        var env = EnvGenerator.EnvFileExists() ? EnvGenerator.ReadAll() : new Dictionary<string, string>();
        var lines = new List<string>
        {
            $"Generated: {DateTime.Now:O}",
            $"Victory version: {UpdateChecker.CurrentVersionText()}",
            $"OS: {Environment.OSVersion.VersionString} ({(Environment.Is64BitOperatingSystem ? "64-bit" : "32-bit")})",
            $".NET runtime: {Environment.Version}",
            $"Tunnel mode: {env.GetValueOrDefault("TUNNEL_MODE", "off")}",
            $"Update mode: {env.GetValueOrDefault("UPDATE_MODE", "automatic")}",
            $"Postgres installed: {RuntimeManager.IsPostgresInstalled()}",
            $"Caddy installed: {RuntimeManager.IsCaddyInstalled()}",
            $"Backend process running: {RuntimeManager.IsBackendProcessRunning()}",
            $"Caddy process running: {RuntimeManager.IsCaddyRunning()}",
        };

        try
        {
            var drive = new DriveInfo(Path.GetPathRoot(AppPaths.DataRoot) ?? "C:\\");
            lines.Add($"Free disk space on {drive.Name}: {drive.AvailableFreeSpace / 1024 / 1024 / 1024} GB of {drive.TotalSize / 1024 / 1024 / 1024} GB");
        }
        catch { /* best effort -- not worth failing the whole bundle over */ }

        return string.Join(Environment.NewLine, lines) + Environment.NewLine;
    }
}
