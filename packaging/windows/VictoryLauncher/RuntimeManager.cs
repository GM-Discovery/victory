using System.Diagnostics;
using System.Net.Http;
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

    public static async Task<(bool Success, string Output)> StartPostgresAsync()
    {
        if (await IsPostgresRunningAsync())
            return (true, "already running");

        Directory.CreateDirectory(AppPaths.LogsDir);
        var (exitCode, stdout, stderr) = await RunAsync(
            AppPaths.PgCtlExe,
            $"start -D \"{AppPaths.PgDataDir}\" -l \"{AppPaths.PostgresLogFile}\" -w -o \"-p {PostgresPort}\"",
            TimeSpan.FromMinutes(2));
        return (exitCode == 0, exitCode == 0 ? stdout : stderr);
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
                    process.Kill(entireProcessTree: true);
            }
            catch (ArgumentException)
            {
                // already gone -- nothing to do
            }
        }
        try { File.Delete(pidFile); } catch { /* best effort */ }
    }

    // ---- Combined lifecycle (what the tray icon and wizard actually call) ----

    public static async Task<(bool Success, string Output)> StartAsync()
    {
        var (pgOk, pgOutput) = await StartPostgresAsync();
        if (!pgOk)
            return (false, "Postgres did not start:\n" + pgOutput);

        // Confirmed on real hardware: the tray icon stays clickable
        // during first-run setup (a deliberate, separate fix), which
        // means "Open Victory" can race the wizard's own
        // RuntimeSetup.EnsureRuntimeReadyAsync -- if this StartAsync
        // call reaches the backend before that sequence's own
        // CreateVictoryDatabaseAsync call does, the backend starts
        // pointed at a database that doesn't exist yet ("FATAL:
        // database victory does not exist" in postgres.log). createdb
        // is safe to call every time (already-exists is treated as
        // success), so making this function self-sufficient here costs
        // nothing on the already-correct path and fixes the race on
        // every other one, rather than relying on every caller of
        // StartAsync to already know to create the database first.
        var env = EnvGenerator.ReadAll();
        var (dbOk, dbOutput) = await CreateVictoryDatabaseAsync(env["POSTGRES_PASSWORD"]);
        if (!dbOk)
            return (false, "Could not create the Victory database:\n" + dbOutput);

        var (backendOk, backendOutput) = await StartBackendAsync();
        if (!backendOk)
            return (false, backendOutput);

        var (caddyOk, caddyOutput) = StartCaddy();
        return (caddyOk, caddyOk ? backendOutput : caddyOutput);
    }

    public static async Task<(bool Success, string Output)> StopAsync()
    {
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

        return (process.ExitCode, await stdoutTask, await stderrTask);
    }
}
