using System.IO.Compression;
using System.Net.Http;

namespace VictoryLauncher;

/// <summary>
/// Kernel 100 §2, windows-native branch. Compared to the abandoned
/// Podman/WSL2 approach (see the k100-podman-wsl2-attempt tag), this is
/// dramatically simpler: no elevation, no reboot, no Windows feature
/// changes. Everything here runs as the current user.
///
/// Downloads two runtimes on first run: PostgreSQL (EDB's own docs
/// describe the Windows binaries zip as intended for exactly this
/// "bundle it into another application's installer" use case) and Caddy
/// (needed because the Go backend is API-only and never served
/// frontend/'s static files itself -- confirmed missing on real
/// hardware the first time this branch was tested end to end, "Open
/// Victory" 404'd even though the backend was healthy).
/// </summary>
internal static class RuntimeSetup
{
    // PostgreSQL 16.15 Windows x64 binaries, matching the major version
    // this project runs everywhere else (docker-compose.yml,
    // packaging/podman/compose.yml both pin postgres:16). Re-pin
    // deliberately when it needs to change -- check
    // https://www.enterprisedb.com/download-postgresql-binaries for the
    // current fileid, the same way the pinned Docker image digests
    // elsewhere in this repo get updated by hand, not automatically.
    private const string PostgresDownloadUrl = "https://sbp.enterprisedb.com/getfile.jsp?fileid=1260494";

    // Caddy 2.11.4 Windows x64, from its own GitHub releases (verified
    // this exact URL resolves). Re-pin by hand when it needs to change --
    // see https://github.com/caddyserver/caddy/releases.
    private const string CaddyDownloadUrl = "https://github.com/caddyserver/caddy/releases/download/v2.11.4/caddy_2.11.4_windows_amd64.zip";

    // cloudflared 2026.8.3 Windows x64, from its own GitHub releases
    // (verified this exact URL resolves). Ships as a single exe, no zip.
    // Re-pin by hand -- see https://github.com/cloudflare/cloudflared/releases.
    private const string CloudflaredDownloadUrl = "https://github.com/cloudflare/cloudflared/releases/download/2026.8.3/cloudflared-windows-amd64.exe";

    public enum Result
    {
        Ready,
        Failed,
    }

    public sealed record SetupResult(Result Result, string? Detail);

    /// <summary>
    /// Current/Total lets a caller render an overall progress bar or "step
    /// N of M" without hardcoding the step count itself -- which steps
    /// actually run varies (downloads only happen once each), so Total is
    /// computed fresh at the start of each EnsureRuntimeReadyAsync call.
    /// </summary>
    public readonly record struct StepProgress(string Label, int Current, int Total);

    /// <summary>
    /// updateProgress is optional: callers that can only show a step name
    /// (tray balloon tips, which are transient popups, not a persistent
    /// line of text) can leave it null. Callers that render a persistent
    /// status line (the wizard's progress list) should pass one --
    /// downloading Postgres is 150+MB and was sitting on one static line
    /// for however long that took with zero feedback, which is exactly
    /// what prompted "I keep wondering if it's done or broken" on real
    /// hardware. Real byte progress, not decorative filler: K100 §38
    /// already establishes "do not fake a percentage" as doctrine for
    /// this app, and that cuts the other way here too -- the fix for
    /// looking possibly-frozen is truthful progress, not a distraction.
    /// </summary>
    public static async Task<SetupResult> EnsureRuntimeReadyAsync(Action<StepProgress> reportStep, Action<string>? updateProgress = null)
    {
        var needsPostgresDownload = !RuntimeManager.IsPostgresInstalled();
        var needsDbInit = !RuntimeManager.IsDatabaseInitialized();
        var needsCaddyDownload = !RuntimeManager.IsCaddyInstalled();
        var tunnelMode = EnvGenerator.EnvFileExists() ? EnvGenerator.ReadAll().GetValueOrDefault("TUNNEL_MODE", "off") : "off";
        var needsTunnel = tunnelMode is "quick" or "named";
        var needsCloudflaredDownload = needsTunnel && !RuntimeManager.IsCloudflaredInstalled();

        // Total is whichever of the always-run steps plus the conditional
        // (first-run-only) downloads actually apply this time -- known
        // upfront since the checks above are all cheap local lookups, so
        // a caller can size a progress bar correctly before the first
        // step even starts.
        var total = 4 + (needsPostgresDownload ? 1 : 0) + (needsDbInit ? 1 : 0) + (needsCaddyDownload ? 1 : 0)
            + (needsTunnel ? 1 : 0) + (needsCloudflaredDownload ? 1 : 0);
        var current = 0;
        void Report(string label) => reportStep(new StepProgress(label, ++current, total));

        if (needsPostgresDownload)
        {
            Report("Downloading the Victory runtime (this only happens once)");
            var (downloaded, downloadError) = await DownloadAndExtractPostgresAsync(updateProgress);
            if (!downloaded)
                return new SetupResult(Result.Failed, downloadError);
        }

        if (needsDbInit)
        {
            Report("Setting up private storage");
            var (initialized, initError) = await InitializeDatabaseAsync();
            if (!initialized)
                return new SetupResult(Result.Failed, initError);
        }

        // Four granular steps, not one "Starting Victory" covering all of
        // them silently: that used to leave no visible sign of which of
        // these four was actually running, or how long is normal --
        // exactly what prompted "track the changes it takes to get caddy
        // running... so I don't prematurely click" on real hardware.
        Report("Starting Victory's database");
        var (pgOk, pgOutput) = await RuntimeManager.StartPostgresAsync();
        if (!pgOk)
            return new SetupResult(Result.Failed, "Postgres did not start:\n" + pgOutput);

        Report("Setting up Victory's database");
        var env = EnvGenerator.ReadAll();
        var (dbOk, dbOutput) = await RuntimeManager.CreateVictoryDatabaseAsync(env["POSTGRES_PASSWORD"]);
        if (!dbOk)
            return new SetupResult(Result.Failed, "Could not create the Victory database:\n" + dbOutput);

        Report("Starting Victory's server");
        var (backendOk, backendOutput) = await RuntimeManager.StartBackendAsync();
        if (!backendOk)
            return new SetupResult(Result.Failed, "Victory could not start:\n" + backendOutput);

        if (needsCaddyDownload)
        {
            Report("Downloading Victory's web server (this only happens once)");
            var (downloaded, downloadError) = await DownloadAndExtractCaddyAsync(updateProgress);
            if (!downloaded)
                return new SetupResult(Result.Failed, downloadError);
        }

        Report("Starting Victory's web server");
        var (caddyOk, caddyOutput) = RuntimeManager.StartCaddy();
        if (!caddyOk)
            return new SetupResult(Result.Failed, "Victory could not be reached:\n" + caddyOutput);

        if (needsTunnel)
        {
            // Remote access is an enhancement, not core functionality --
            // Victory must stay usable locally even if the tunnel can't
            // start (no internet, Cloudflare's own service being down,
            // whatever). Deliberately not gated behind a return-Failed
            // check the way every step above it is; failures here are
            // swallowed and left for Status to surface honestly, rather
            // than blocking the Operator from using Victory at all over
            // an optional feature they may not even be watching for.
            if (needsCloudflaredDownload)
            {
                Report("Downloading Victory's remote access tool (this only happens once)");
                await DownloadCloudflaredAsync(updateProgress);
            }

            if (RuntimeManager.IsCloudflaredInstalled())
            {
                Report("Starting your Victory address");
                try
                {
                    if (tunnelMode == "named")
                        await RuntimeManager.StartNamedTunnelAsync(env.GetValueOrDefault("CLOUDFLARE_TUNNEL_TOKEN", ""));
                    else
                        await RuntimeManager.StartQuickTunnelAsync();
                }
                catch { /* best effort -- see comment above */ }
            }
        }

        return new SetupResult(Result.Ready, null);
    }

    private static async Task<(bool Success, string? Error)> DownloadCloudflaredAsync(Action<string>? updateProgress)
    {
        try
        {
            Directory.CreateDirectory(AppPaths.CloudflaredRoot);
            using var client = new HttpClient { Timeout = TimeSpan.FromMinutes(5) };
            using var response = await client.GetAsync(CloudflaredDownloadUrl, HttpCompletionOption.ResponseHeadersRead);
            response.EnsureSuccessStatusCode();
            await using (var fileStream = File.Create(AppPaths.CloudflaredExe))
            {
                await CopyWithProgressAsync(response, fileStream, "Downloading Victory's remote access tool", updateProgress);
            }

            if (!File.Exists(AppPaths.CloudflaredExe))
                return (false, "cloudflared.exe was not found where expected after downloading it.");
            return (true, null);
        }
        catch (Exception ex)
        {
            return (false, "Could not download Victory's remote access tool:\n" + ex.Message);
        }
    }

    private static async Task<(bool Success, string? Error)> DownloadAndExtractCaddyAsync(Action<string>? updateProgress)
    {
        var tempZip = Path.Combine(Path.GetTempPath(), "victory-caddy-" + Guid.NewGuid().ToString("N") + ".zip");
        try
        {
            using var client = new HttpClient { Timeout = TimeSpan.FromMinutes(10) };
            using (var response = await client.GetAsync(CaddyDownloadUrl, HttpCompletionOption.ResponseHeadersRead))
            {
                response.EnsureSuccessStatusCode();
                await using var fileStream = File.Create(tempZip);
                await CopyWithProgressAsync(response, fileStream, "Downloading the Victory runtime", updateProgress);
            }

            Directory.CreateDirectory(AppPaths.CaddyRoot);
            ZipFile.ExtractToDirectory(tempZip, AppPaths.CaddyRoot, overwriteFiles: true);

            if (!File.Exists(AppPaths.CaddyExe))
                return (false, "The Victory runtime was downloaded, but caddy.exe was not found where expected after extracting it.");
            return (true, null);
        }
        catch (Exception ex)
        {
            return (false, "Could not download or set up the Victory runtime:\n" + ex.Message);
        }
        finally
        {
            try { File.Delete(tempZip); } catch { /* best effort */ }
        }
    }

    private static async Task<(bool Success, string? Error)> DownloadAndExtractPostgresAsync(Action<string>? updateProgress)
    {
        var tempZip = Path.Combine(Path.GetTempPath(), "victory-postgres-" + Guid.NewGuid().ToString("N") + ".zip");
        try
        {
            using var client = new HttpClient { Timeout = TimeSpan.FromMinutes(15) };
            using (var response = await client.GetAsync(PostgresDownloadUrl, HttpCompletionOption.ResponseHeadersRead))
            {
                response.EnsureSuccessStatusCode();
                await using var fileStream = File.Create(tempZip);
                await CopyWithProgressAsync(response, fileStream, "Downloading the Victory runtime", updateProgress);
            }

            Directory.CreateDirectory(AppPaths.PostgresRoot);
            ZipFile.ExtractToDirectory(tempZip, AppPaths.PostgresRoot, overwriteFiles: true);

            // EDB's zip has a top-level "pgsql" folder wrapping bin/lib/
            // share -- flatten it so AppPaths.PostgresBinDir
            // (PostgresRoot\bin) is right regardless of whether a given
            // release's archive nests it or not.
            var nestedPgsql = Path.Combine(AppPaths.PostgresRoot, "pgsql");
            if (Directory.Exists(nestedPgsql) && !File.Exists(AppPaths.PgCtlExe))
            {
                foreach (var entry in Directory.GetFileSystemEntries(nestedPgsql))
                {
                    var destination = Path.Combine(AppPaths.PostgresRoot, Path.GetFileName(entry));
                    if (Directory.Exists(entry))
                        Directory.Move(entry, destination);
                    else
                        File.Move(entry, destination);
                }
                Directory.Delete(nestedPgsql, recursive: true);
            }

            if (!File.Exists(AppPaths.PgCtlExe))
                return (false, "The Victory runtime was downloaded, but pg_ctl.exe was not found where expected after extracting it.");
            return (true, null);
        }
        catch (Exception ex)
        {
            return (false, "Could not download or set up the Victory runtime:\n" + ex.Message);
        }
        finally
        {
            try { File.Delete(tempZip); } catch { /* best effort */ }
        }
    }

    private static async Task<(bool Success, string? Error)> InitializeDatabaseAsync()
    {
        var env = EnvGenerator.ReadAll();
        var password = env["POSTGRES_PASSWORD"];
        var pwFile = Path.Combine(Path.GetTempPath(), "victory-pg-pw-" + Guid.NewGuid().ToString("N") + ".txt");
        try
        {
            await File.WriteAllTextAsync(pwFile, password + "\n");
            var (exitCode, _, stderr) = await RuntimeManager.RunAsync(
                AppPaths.InitDbExe,
                $"-D \"{AppPaths.PgDataDir}\" -U victory --auth=scram-sha-256 --pwfile=\"{pwFile}\" -E UTF8",
                TimeSpan.FromMinutes(2));
            return exitCode == 0 ? (true, null) : (false, "Setting up private storage failed:\n" + stderr);
        }
        finally
        {
            try { File.Delete(pwFile); } catch { /* best effort */ }
        }
    }

    /// <summary>
    /// Real byte-level progress, not a fixed poll interval: throttled to
    /// roughly once per 250ms so it updates smoothly on a fast connection
    /// without flooding the UI thread with marshaled calls on an even
    /// faster one. If the server doesn't report Content-Length, falls
    /// back to a running MB-downloaded counter -- still real information,
    /// just without a percentage or ETA.
    /// </summary>
    private static async Task CopyWithProgressAsync(HttpResponseMessage response, Stream destination, string label, Action<string>? updateProgress)
    {
        var totalBytes = response.Content.Headers.ContentLength;
        await using var source = await response.Content.ReadAsStreamAsync();

        var buffer = new byte[81920];
        long bytesRead = 0;
        var lastReportedAt = DateTime.UtcNow;
        int read;
        while ((read = await source.ReadAsync(buffer)) > 0)
        {
            await destination.WriteAsync(buffer.AsMemory(0, read));
            bytesRead += read;

            var now = DateTime.UtcNow;
            if (updateProgress is not null && now - lastReportedAt > TimeSpan.FromMilliseconds(250))
            {
                lastReportedAt = now;
                var downloadedMb = bytesRead / 1024.0 / 1024.0;
                updateProgress(totalBytes.HasValue
                    ? $"{label}... {downloadedMb:F0} MB / {totalBytes.Value / 1024.0 / 1024.0:F0} MB ({bytesRead * 100 / totalBytes.Value}%)"
                    : $"{label}... {downloadedMb:F0} MB");
            }
        }
    }
}
