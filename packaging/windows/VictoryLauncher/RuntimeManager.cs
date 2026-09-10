using System.Diagnostics;
using System.Net.Http;
using System.Text.Json;
using System.Text.RegularExpressions;
using System.Threading;

namespace VictoryLauncher;

/// <summary>
/// Kernel 100 §2-3, windows-native branch: "Victory manages its runtime"
/// now means running Postgres and the Victory backend as plain native
/// Windows processes this app supervises directly -- no container
/// runtime, no WSL2, no elevation, ever. Postgres's own pg_ctl handles
/// start/stop/status via its standard postmaster.pid mechanism (so this
/// class never has to track that process's PID itself across launcher
/// restarts); the backend is a plain child process this class does track
/// via a PID file, since Go has no pg_ctl equivalent.
///
/// Installing Postgres itself (downloading, extracting, initdb) is
/// RuntimeSetup's job, not this class's -- this class assumes
/// AppPaths.PgCtlExe and AppPaths.PgDataDir already exist and are ready.
/// </summary>
internal static class RuntimeManager
{
    public const int PostgresPort = 55432; // arbitrary local-only port, chosen to avoid colliding with any real system Postgres on the default 5432
    public const int BackendPort = 8081; // internal only -- Caddy is the only thing that talks to this port
    public const int PublicPort = 8080; // what a browser actually opens; matches the local dev loop's Caddyfile.local convention

    // ---- Postgres ----

    public static bool IsPostgresInstalled() => File.Exists(AppPaths.PgCtlExe);

    public static bool IsDatabaseInitialized() =>
        Directory.Exists(AppPaths.PgDataDir) && File.Exists(Path.Combine(AppPaths.PgDataDir, "PG_VERSION"));

    public static async Task<bool> IsPostgresRunningAsync()
    {
        var (exitCode, _, _) = await RunAsync(AppPaths.PgCtlExe, $"status -D \"{AppPaths.PgDataDir}\"", TimeSpan.FromSeconds(10));
        // pg_ctl status: 0 = running, 3 = not running, 4 = data directory
        // inaccessible -- only 0 means "already up," everything else means
        // "safe to try starting."
        return exitCode == 0;
    }

    /// <summary>
    /// Windows-only: pg_ctl start can fail intermittently with "could not
    /// reserve shared memory region" (Windows error 487), confirmed on
    /// real hardware via postgres.log -- ASLR relocated a Postgres helper
    /// process on top of the address the shared memory segment wanted.
    /// Each attempt gets a fresh random base address, so a short retry
    /// clears the transient case. If it's actually a fixed-address
    /// collision (some antivirus/EDR products inject a DLL at the same
    /// address on every process launch), retries won't help -- which is
    /// why the final failure surfaces the real postgres.log tail instead
    /// of pg_ctl's own generic "could not start server, examine the log
    /// output" (which never actually shows the log it's telling you to
    /// examine).
    /// </summary>
    public static async Task<(bool Success, string Output)> StartPostgresAsync()
    {
        if (await IsPostgresRunningAsync())
            return (true, "already running");

        Directory.CreateDirectory(AppPaths.LogsDir);

        const int maxAttempts = 3;
        for (var attempt = 1; attempt <= maxAttempts; attempt++)
        {
            var (exitCode, stdout, _) = await RunAsync(
                AppPaths.PgCtlExe,
                $"start -D \"{AppPaths.PgDataDir}\" -l \"{AppPaths.PostgresLogFile}\" -w -o \"-p {PostgresPort}\"",
                TimeSpan.FromMinutes(2));
            if (exitCode == 0)
                return (true, stdout);

            if (attempt < maxAttempts)
                await Task.Delay(2000);
        }

        var logTail = ReadLogTail(AppPaths.PostgresLogFile, 20);
        return (false, string.IsNullOrWhiteSpace(logTail)
            ? "Postgres did not start after several attempts."
            : "Postgres did not start after several attempts. Its own log said:\n" + logTail);
    }

    private static string? ReadLogTail(string path, int lineCount)
    {
        try
        {
            if (!File.Exists(path))
                return null;
            using var stream = new FileStream(path, FileMode.Open, FileAccess.Read, FileShare.ReadWrite);
            using var reader = new StreamReader(stream);
            var lines = reader.ReadToEnd().Split('\n');
            return string.Join('\n', lines.Length > lineCount ? lines[^lineCount..] : lines).Trim();
        }
        catch
        {
            return null; // best effort -- a missing/locked log shouldn't hide the real failure behind a secondary one
        }
    }

    public static async Task<(bool Success, string Output)> StopPostgresAsync()
    {
        var (exitCode, stdout, stderr) = await RunAsync(
            AppPaths.PgCtlExe, $"stop -D \"{AppPaths.PgDataDir}\" -m fast -w", TimeSpan.FromSeconds(30));
        return (exitCode == 0, exitCode == 0 ? stdout : stderr);
    }

    /// <summary>
    /// initdb only creates the "postgres"/template0/template1 databases --
    /// this creates the actual "victory" database compose.yml's
    /// DATABASE_URL always pointed at, run once, after the server is up.
    /// </summary>
    public static async Task<(bool Success, string Output)> CreateVictoryDatabaseAsync(string postgresPassword)
    {
        var createdbExe = Path.Combine(AppPaths.PostgresBinDir, "createdb.exe");
        var (exitCode, stdout, stderr) = await RunAsync(
            createdbExe,
            $"-U victory -h 127.0.0.1 -p {PostgresPort} victory",
            TimeSpan.FromSeconds(30),
            extraEnv: new() { ["PGPASSWORD"] = postgresPassword });
        // createdb exits non-zero if the database already exists -- treated
        // as success since this is meant to be safe to call on every boot,
        // same idempotency contract as the Ensure*Surface bootstraps on
        // the backend side.
        if (exitCode == 0 || stderr.Contains("already exists", StringComparison.OrdinalIgnoreCase))
            return (true, stdout);
        return (false, stderr);
    }

    // ---- Backend ----

    public static async Task<(bool Success, string Output)> StartBackendAsync()
    {
        if (IsBackendProcessRunning())
            return (true, "already running");

        Directory.CreateDirectory(AppPaths.LogsDir);
        var env = EnvGenerator.ReadAll();
        var postgresPassword = env["POSTGRES_PASSWORD"];

        var psi = new ProcessStartInfo
        {
            FileName = AppPaths.BackendExePath,
            UseShellExecute = false,
            CreateNoWindow = true,
            RedirectStandardOutput = true,
            RedirectStandardError = true,
        };
        psi.Environment["PORT"] = BackendPort.ToString();
        // Binding every interface (the backend's own default, needed for
        // container deployments) is what triggers a Windows Firewall
        // "allow this app" prompt -- confirmed on real hardware. Nothing
        // outside this machine ever needs to reach the backend directly
        // (Caddy is the only thing that talks to it, also on localhost),
        // so loopback-only is both correct and firewall-prompt-free.
        psi.Environment["BIND_HOST"] = "127.0.0.1";
        psi.Environment["DATABASE_URL"] = $"postgres://victory:{postgresPassword}@127.0.0.1:{PostgresPort}/victory?sslmode=disable";
        psi.Environment["STORAGE_ROOT"] = env["STORAGE_ROOT"];
        psi.Environment["BACKUP_DIR"] = env["BACKUP_DIR"];
        psi.Environment["EXPORTS_ROOT"] = env["EXPORTS_ROOT"];
        psi.Environment["OPERATOR_HANDLE"] = env["OPERATOR_HANDLE"];
        psi.Environment["DEFAULT_LOCATION_SLUG"] = env["DEFAULT_LOCATION_SLUG"];
        psi.Environment["DEFAULT_LOCATION_NAME"] = env.GetValueOrDefault("DEFAULT_LOCATION_NAME", "");
        psi.Environment["COOKIE_SECURE"] = "true";
        // The backend shells out to a bare "pg_dump" for its pre-migration
        // backup (internal/migrate/migrate.go) -- silently skipped on a
        // database with no tables yet (a fresh install), so this doesn't
        // block first-run setup, but it WOULD fail the moment a future
        // update ships a schema-changing migration against an
        // already-populated database, since pg_dump.exe only exists
        // inside the Postgres download this app manages, never on the
        // system PATH. Prepending here, not replacing: the backend still
        // needs whatever else is normally on PATH.
        psi.Environment.TryGetValue("PATH", out var existingPath);
        psi.Environment["PATH"] = AppPaths.PostgresBinDir + ";" + (existingPath ?? "");

        try
        {
            var process = Process.Start(psi);
            if (process is null)
                return (false, "failed to start victory-backend.exe");

            await File.WriteAllTextAsync(AppPaths.BackendPidFile, process.Id.ToString());

            // Not awaited: this is a long-running server process, not a
            // one-shot command RunAsync's model fits. Its own log file
            // (redirected below) is the record of what it does after this
            // point, same as postgres.log for the database.
            _ = LogProcessOutputAsync(process, AppPaths.BackendLogFile);

            return (true, $"started (pid {process.Id})");
        }
        catch (Exception ex)
        {
            return (false, ex.Message);
        }
    }

    public static bool IsBackendProcessRunning() => IsProcessRunning(AppPaths.BackendPidFile, "victory-backend");

    public static void StopBackend() => StopProcessByPidFile(AppPaths.BackendPidFile);

    // ---- Caddy ----
    // The Go backend is API-only (never served frontend/'s static files
    // or handled "/"). Caddy is what makes "Open Victory" show the
    // campus map instead of a 404 -- confirmed missing on real hardware.
    // Managed the same way as the backend: no pg_ctl equivalent exists
    // for Caddy either, so this app tracks its PID itself.

    public static bool IsCaddyInstalled() => File.Exists(AppPaths.CaddyExe);

    public static bool IsCaddyRunning() => IsProcessRunning(AppPaths.CaddyPidFile, "caddy");

    public static (bool Success, string Output) StartCaddy()
    {
        if (IsCaddyRunning())
            return (true, "already running");

        Directory.CreateDirectory(AppPaths.LogsDir);
        var psi = new ProcessStartInfo
        {
            FileName = AppPaths.CaddyExe,
            Arguments = $"run --config \"{AppPaths.CaddyfilePath}\" --adapter caddyfile",
            // frontend/'s Caddyfile "root * frontend" is relative to this
            // -- AppContext.BaseDirectory is where both the Caddyfile and
            // the bundled frontend/ copy actually live.
            WorkingDirectory = AppContext.BaseDirectory,
            UseShellExecute = false,
            CreateNoWindow = true,
            RedirectStandardOutput = true,
            RedirectStandardError = true,
        };

        try
        {
            var process = Process.Start(psi);
            if (process is null)
                return (false, "failed to start caddy.exe");

            File.WriteAllText(AppPaths.CaddyPidFile, process.Id.ToString());
            _ = LogProcessOutputAsync(process, AppPaths.CaddyLogFile);
            return (true, $"started (pid {process.Id})");
        }
        catch (Exception ex)
        {
            return (false, ex.Message);
        }
    }

    public static void StopCaddy() => StopProcessByPidFile(AppPaths.CaddyPidFile);

    // ---- Tunnel (Cloudflare) ----
    // Off by default -- remote access is an explicit Operator/first-run
    // choice (TUNNEL_MODE in .env), never forced. Quick Tunnel needs no
    // Cloudflare account from anyone: cloudflared just prints a fresh
    // https://*.trycloudflare.com address to its own output every time
    // it starts, which is captured below the same way the backend/Caddy
    // logs are captured, with one extra regex check per line. That
    // address changes on every restart -- Status is expected to surface
    // that plainly rather than pretend it's stable.

    public static bool IsCloudflaredInstalled() => File.Exists(AppPaths.CloudflaredExe);

    public static bool IsTunnelRunning() => IsProcessRunning(AppPaths.CloudflaredPidFile, "cloudflared");

    public static void StopTunnel() => StopProcessByPidFile(AppPaths.CloudflaredPidFile);

    private static readonly Regex QuickTunnelUrlPattern = new(@"https://[a-z0-9-]+\.trycloudflare\.com", RegexOptions.Compiled);

    public static async Task<(bool Success, string Output)> StartQuickTunnelAsync()
    {
        if (IsTunnelRunning())
            return (true, "already running");

        Directory.CreateDirectory(AppPaths.LogsDir);
        var psi = new ProcessStartInfo
        {
            FileName = AppPaths.CloudflaredExe,
            Arguments = $"tunnel --url http://127.0.0.1:{PublicPort}",
            UseShellExecute = false,
            CreateNoWindow = true,
            RedirectStandardOutput = true,
            RedirectStandardError = true,
        };

        try
        {
            var process = Process.Start(psi);
            if (process is null)
                return (false, "failed to start cloudflared.exe");

            await File.WriteAllTextAsync(AppPaths.CloudflaredPidFile, process.Id.ToString());
            _ = LogTunnelOutputAndCaptureUrlAsync(process);
            return (true, $"started (pid {process.Id})");
        }
        catch (Exception ex)
        {
            return (false, ex.Message);
        }
    }

    /// <summary>
    /// For an Operator who already has their own Cloudflare account,
    /// domain, and Tunnel token -- created entirely outside Victory, in
    /// Cloudflare's own dashboard. No URL to capture here: unlike Quick
    /// Tunnel, the address is whatever the Operator already configured on
    /// Cloudflare's side (CLOUDFLARE_TUNNEL_HOSTNAME, stored only for
    /// Status to display/copy), so this just logs plainly like the
    /// backend/Caddy already do.
    /// </summary>
    public static async Task<(bool Success, string Output)> StartNamedTunnelAsync(string tunnelToken)
    {
        if (IsTunnelRunning())
            return (true, "already running");
        if (string.IsNullOrWhiteSpace(tunnelToken))
            return (false, "no tunnel token configured");

        Directory.CreateDirectory(AppPaths.LogsDir);
        var psi = new ProcessStartInfo
        {
            FileName = AppPaths.CloudflaredExe,
            Arguments = $"tunnel run --token {tunnelToken}",
            UseShellExecute = false,
            CreateNoWindow = true,
            RedirectStandardOutput = true,
            RedirectStandardError = true,
        };

        try
        {
            var process = Process.Start(psi);
            if (process is null)
                return (false, "failed to start cloudflared.exe");

            await File.WriteAllTextAsync(AppPaths.CloudflaredPidFile, process.Id.ToString());
            _ = LogProcessOutputAsync(process, AppPaths.CloudflaredLogFile);
            return (true, $"started (pid {process.Id})");
        }
        catch (Exception ex)
        {
            return (false, ex.Message);
        }
    }

    private static async Task LogTunnelOutputAndCaptureUrlAsync(Process process)
    {
        try
        {
            await using var log = new StreamWriter(AppPaths.CloudflaredLogFile, append: true) { AutoFlush = true };
            var stdoutTask = CopyStreamAndWatchForTunnelUrlAsync(process.StandardOutput, log);
            var stderrTask = CopyStreamAndWatchForTunnelUrlAsync(process.StandardError, log);
            await Task.WhenAll(stdoutTask, stderrTask);
        }
        catch
        {
            // Logging failures must never take the actual tunnel process down with them.
        }
    }

    private static async Task CopyStreamAndWatchForTunnelUrlAsync(StreamReader reader, StreamWriter writer)
    {
        string? line;
        while ((line = await reader.ReadLineAsync()) is not null)
        {
            await writer.WriteLineAsync(line);
            var match = QuickTunnelUrlPattern.Match(line);
            if (match.Success)
            {
                try { await File.WriteAllTextAsync(AppPaths.TunnelUrlFile, match.Value); } catch { /* best effort */ }
                _ = UpdateGistPointerAsync(match.Value);
            }
        }
    }

    public static string? GetLastKnownTunnelUrl()
    {
        try
        {
            return File.Exists(AppPaths.TunnelUrlFile) ? File.ReadAllText(AppPaths.TunnelUrlFile).Trim() : null;
        }
        catch
        {
            return null;
        }
    }

    /// <summary>
    /// Quick Tunnel's address is inherently ephemeral -- this is the
    /// optional fix an Operator can turn on themselves: a GitHub Gist
    /// (their own account, their own token, nothing for anyone else to
    /// host) that always holds the current address, so a link shared
    /// once keeps working across restarts. No-op entirely if no token is
    /// configured (the default -- this is opt-in, not asked at first
    /// run). Best-effort like the rest of the tunnel subsystem: a failed
    /// Gist update never affects Victory's own availability.
    /// </summary>
    private static async Task UpdateGistPointerAsync(string currentUrl)
    {
        try
        {
            if (!EnvGenerator.EnvFileExists())
                return;
            var token = EnvGenerator.ReadAll().GetValueOrDefault("GITHUB_GIST_TOKEN", "");
            if (string.IsNullOrWhiteSpace(token))
                return;

            using var client = new HttpClient { Timeout = TimeSpan.FromSeconds(10) };
            client.DefaultRequestHeaders.Add("Authorization", $"token {token}");
            client.DefaultRequestHeaders.Add("User-Agent", "VictoryLauncher");
            client.DefaultRequestHeaders.Add("Accept", "application/vnd.github+json");

            var fileContent = JsonSerializer.Serialize(new
            {
                files = new Dictionary<string, object> { ["victory-address.txt"] = new { content = currentUrl } },
            });

            var existingGistId = File.Exists(AppPaths.GistIdFile) ? File.ReadAllText(AppPaths.GistIdFile).Trim() : "";
            if (!string.IsNullOrEmpty(existingGistId))
            {
                var patchResponse = await client.PatchAsync(
                    $"https://api.github.com/gists/{existingGistId}",
                    new StringContent(fileContent, System.Text.Encoding.UTF8, "application/json"));
                if (patchResponse.IsSuccessStatusCode)
                    return;
                // Fall through to create a fresh one if the stored id is
                // stale (e.g. the Gist was deleted on GitHub's side).
            }

            var createBody = JsonSerializer.Serialize(new
            {
                description = "Victory's current address (kept updated automatically)",
                @public = false,
                files = new Dictionary<string, object> { ["victory-address.txt"] = new { content = currentUrl } },
            });
            var createResponse = await client.PostAsync(
                "https://api.github.com/gists",
                new StringContent(createBody, System.Text.Encoding.UTF8, "application/json"));
            if (!createResponse.IsSuccessStatusCode)
                return;

            using var doc = JsonDocument.Parse(await createResponse.Content.ReadAsStringAsync());
            if (doc.RootElement.TryGetProperty("id", out var idProp))
                await File.WriteAllTextAsync(AppPaths.GistIdFile, idProp.GetString());
        }
        catch
        {
            // best effort -- see summary above
        }
    }

    public static async Task<bool> IsPublicSiteReachableAsync()
    {
        try
        {
            using var client = new HttpClient { Timeout = TimeSpan.FromSeconds(3) };
            var response = await client.GetAsync($"http://127.0.0.1:{PublicPort}/");
            return response.IsSuccessStatusCode;
        }
        catch
        {
            return false;
        }
    }

    // ---- Shared PID-file process tracking (backend and Caddy both use this; Postgres uses pg_ctl instead) ----

    private static bool IsProcessRunning(string pidFile, string expectedProcessName)
    {
        if (!File.Exists(pidFile))
            return false;
        if (!int.TryParse(File.ReadAllText(pidFile).Trim(), out var pid))
            return false;
        try
        {
            var process = Process.GetProcessById(pid);
            return !process.HasExited && process.ProcessName.Equals(expectedProcessName, StringComparison.OrdinalIgnoreCase);
        }
        catch (ArgumentException)
        {
            return false; // no process with that PID exists anymore
        }
    }

    private static void StopProcessByPidFile(string pidFile)
    {
        if (!File.Exists(pidFile))
            return;
        if (int.TryParse(File.ReadAllText(pidFile).Trim(), out var pid))
        {
            try
            {
                var process = Process.GetProcessById(pid);
                if (!process.HasExited)
                {
                    process.Kill(entireProcessTree: true);
                    // Kill() requests termination but returns immediately --
                    // it does not wait for Windows to actually finish tearing
                    // the process down and releasing its open file handles.
                    // Confirmed on real hardware as the likely cause of
                    // update-apply failures specific to this app: Caddy
                    // serves static files directly out of the very
                    // directory (current\frontend\) Velopack's updater needs
                    // to rename moments later, and callers here (StopCaddy,
                    // called right before ApplyUpdatesAndRestart) had no
                    // guarantee those handles were actually released by the
                    // time the risky call happened. 5s is generous for a
                    // process that was just force-killed, not asked to shut
                    // down gracefully -- if it somehow still hasn't exited by
                    // then, proceeding anyway is no worse than the previous
                    // unconditional behavior.
                    process.WaitForExit(5000);
                }
            }
            catch (ArgumentException)
            {
                // already gone -- nothing to do
            }
        }
        try { File.Delete(pidFile); } catch { /* best effort */ }
    }

    // ---- Combined lifecycle (what the tray icon and wizard actually call) ----

    /// <summary>
    /// Delegates entirely to RuntimeSetup.EnsureRuntimeReadyAsync rather
    /// than re-implementing a shorter version of the same sequence. This
    /// used to duplicate the Postgres-start/create-database/backend-start
    /// steps here, which drifted out of sync with the real sequence twice
    /// in a row on real hardware: first missing the database-creation
    /// step (a race with the wizard's own setup), then missing the
    /// Caddy-not-yet-downloaded check entirely (confirmed on hardware:
    /// Postgres and the backend both running, no caddy.exe process at
    /// all, because this function tried to start it without ever
    /// checking it had been downloaded). One authoritative "make sure
    /// everything is actually running" sequence, used everywhere,
    /// closes off this entire class of bug rather than patching each
    /// missing step as it's discovered.
    /// </summary>
    public static async Task<(bool Success, string Output)> StartAsync()
    {
        var result = await RuntimeSetup.EnsureRuntimeReadyAsync(reportStep: _ => { });
        return (result.Result == RuntimeSetup.Result.Ready, result.Detail ?? "Victory is running");
    }

    public static async Task<(bool Success, string Output)> StopAsync()
    {
        StopTunnel();
        StopCaddy();
        StopBackend();
        return await StopPostgresAsync();
    }

    public static async Task<bool> IsBackendHealthyAsync()
    {
        try
        {
            using var client = new HttpClient { Timeout = TimeSpan.FromSeconds(3) };
            var response = await client.GetAsync($"http://127.0.0.1:{BackendPort}/health");
            return response.IsSuccessStatusCode;
        }
        catch
        {
            return false;
        }
    }

    // ---- Shared process-execution helpers ----

    private static async Task LogProcessOutputAsync(Process process, string logFile)
    {
        try
        {
            await using var log = new StreamWriter(logFile, append: true) { AutoFlush = true };
            var stdoutTask = CopyStreamToWriterAsync(process.StandardOutput, log);
            var stderrTask = CopyStreamToWriterAsync(process.StandardError, log);
            await Task.WhenAll(stdoutTask, stderrTask);
        }
        catch
        {
            // Logging failures must never take the actual backend process
            // down with them.
        }
    }

    private static async Task CopyStreamToWriterAsync(StreamReader reader, StreamWriter writer)
    {
        string? line;
        while ((line = await reader.ReadLineAsync()) is not null)
            await writer.WriteLineAsync(line);
    }

    internal static async Task<(int ExitCode, string Stdout, string Stderr)> RunAsync(
        string fileName, string arguments, TimeSpan timeout, Dictionary<string, string>? extraEnv = null)
    {
        var psi = new ProcessStartInfo
        {
            FileName = fileName,
            Arguments = arguments,
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false,
            CreateNoWindow = true,
        };
        if (extraEnv is not null)
        {
            foreach (var (key, value) in extraEnv)
                psi.Environment[key] = value;
        }

        using var process = Process.Start(psi) ?? throw new InvalidOperationException($"failed to start {fileName}");
        var stdoutTask = process.StandardOutput.ReadToEndAsync();
        var stderrTask = process.StandardError.ReadToEndAsync();

        using var cts = new CancellationTokenSource(timeout);
        try
        {
            await process.WaitForExitAsync(cts.Token);
        }
        catch (OperationCanceledException)
        {
            try { process.Kill(entireProcessTree: true); } catch { /* best effort */ }
            throw new TimeoutException($"{fileName} {arguments} did not exit within {timeout}");
        }

        // The process itself exiting is not the same as its stdout/stderr
        // pipes reaching end-of-stream: `pg_ctl start` launches postgres.exe
        // as a detached background process, confirms it's ready, and exits
        // -- but postgres.exe inherits the same redirected pipe handles,
        // and since it runs indefinitely by design, ReadToEndAsync would
        // otherwise wait forever for EOF that will only arrive when
        // postgres itself eventually stops. Confirmed on real hardware:
        // "Starting Victory's database" hung indefinitely even though
        // postgres.log showed the server had started fine in under a
        // second -- this await was never covered by the timeout above,
        // which only guarded WaitForExitAsync. A short bounded wait here
        // means a still-open inherited handle can no longer block forever;
        // whatever text arrived in time is still returned.
        var outputReady = Task.WhenAll(stdoutTask, stderrTask);
        var finished = await Task.WhenAny(outputReady, Task.Delay(TimeSpan.FromSeconds(5)));
        var stdout = finished == outputReady ? await stdoutTask : "";
        var stderr = finished == outputReady ? await stderrTask : "";
        return (process.ExitCode, stdout, stderr);
    }
}
