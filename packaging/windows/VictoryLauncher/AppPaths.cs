namespace VictoryLauncher;

/// <summary>
/// Kernel 100 §6: application files, user data, generated secrets, and logs
/// are four separate things with different lifetimes. Velopack owns the
/// app-files directory entirely (it gets replaced wholesale on every
/// update and removed on uninstall) -- nothing this app cares about
/// persisting may live there.
///
/// windows-native: everything persistent lives under %LocalAppData%\Victory,
/// not %ProgramData%\Victory (the Podman-era choice). Nothing in this
/// app ever runs elevated anymore -- that was the whole point of this
/// branch -- and %LocalAppData% is unconditionally writable by the
/// current user on every Windows configuration, including locked-down
/// ones where %ProgramData% writes for standard users can be restricted
/// by policy. Single-user/per-account data fits %LocalAppData% anyway;
/// nothing here is meant to be shared across Windows accounts on the
/// same machine.
/// </summary>
internal static class AppPaths
{
    public static string DataRoot { get; } =
        Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData), "Victory");

    public static string StorageDir => Path.Combine(DataRoot, "data", "storage");
    public static string BackupDir => Path.Combine(DataRoot, "data", "backups");
    public static string ExportsDir => Path.Combine(DataRoot, "data", "exports");
    public static string LogsDir => Path.Combine(DataRoot, "logs");

    /// <summary>
    /// Generated config/secrets (K100 §10-11): persistent, not checked in.
    /// ACL-tightened in EnvGenerator regardless of living under
    /// %LocalAppData% (already private to this Windows account by
    /// default) -- defense in depth costs nothing here.
    /// </summary>
    public static string EnvFile => Path.Combine(DataRoot, ".env");

    /// <summary>Downloaded once on first run (RuntimeSetup); never bundled with the installer -- see its own header comment for why.</summary>
    public static string PostgresRoot => Path.Combine(DataRoot, "pgsql");
    public static string PostgresBinDir => Path.Combine(PostgresRoot, "bin");
    public static string PgCtlExe => Path.Combine(PostgresBinDir, "pg_ctl.exe");
    public static string InitDbExe => Path.Combine(PostgresBinDir, "initdb.exe");
    public static string PsqlExe => Path.Combine(PostgresBinDir, "psql.exe");

    /// <summary>The actual database cluster directory pg_ctl/initdb operate on -- distinct from PostgresRoot, which is just the versioned binaries.</summary>
    public static string PgDataDir => Path.Combine(DataRoot, "pgdata");
    public static string PostgresLogFile => Path.Combine(LogsDir, "postgres.log");

    /// <summary>
    /// Bundled directly alongside the launcher exe (Velopack's own
    /// versioned app-files directory, not %LocalAppData%) -- this is
    /// "what version of Victory's backend this release of the launcher
    /// runs," which changes with every release, unlike the Postgres
    /// runtime which barely ever needs to change.
    /// </summary>
    public static string BackendExePath => Path.Combine(AppContext.BaseDirectory, "victory-backend.exe");
    public static string BackendLogFile => Path.Combine(LogsDir, "victory-backend.log");
    public static string BackendPidFile => Path.Combine(DataRoot, "victory-backend.pid");

    /// <summary>
    /// The Go backend is API-only: it never served frontend/'s static
    /// files or handled "/" at all. Every existing deployment (Grant's
    /// own server, the local Linux dev loop) relies on Caddy in front of
    /// it for exactly that -- a piece neither the Podman nor the first
    /// native attempt at this launcher included, so "Open Victory" 404'd
    /// on real hardware even though the backend itself was healthy.
    /// Downloaded once on first run (RuntimeSetup), same as Postgres.
    /// </summary>
    public static string CaddyRoot => Path.Combine(DataRoot, "caddy");
    public static string CaddyExe => Path.Combine(CaddyRoot, "caddy.exe");
    public static string CaddyLogFile => Path.Combine(LogsDir, "caddy.log");
    public static string CaddyPidFile => Path.Combine(DataRoot, "caddy.pid");

    /// <summary>
    /// Bundled as static content, not generated: the frontend and Caddy
    /// binary locations are both fixed relative to AppContext.BaseDirectory
    /// at build time, so unlike EnvGenerator's secrets there is nothing
    /// here that actually varies per install.
    /// </summary>
    public static string CaddyfilePath => Path.Combine(AppContext.BaseDirectory, "Caddyfile");
    public static string FrontendRoot => Path.Combine(AppContext.BaseDirectory, "frontend");

    /// <summary>
    /// Cloudflare Tunnel, off by default (TUNNEL_MODE in .env) -- remote
    /// access is meaningfully different from the other bundled runtimes
    /// (it changes who can reach this machine at all), so unlike
    /// Postgres/Caddy this is downloaded lazily the first time a tunnel
    /// mode actually needs it, not unconditionally on every first run.
    /// cloudflared.exe ships as a single file, no zip to extract.
    /// </summary>
    public static string CloudflaredRoot => Path.Combine(DataRoot, "cloudflared");
    public static string CloudflaredExe => Path.Combine(CloudflaredRoot, "cloudflared.exe");
    public static string CloudflaredLogFile => Path.Combine(LogsDir, "cloudflared.log");
    public static string CloudflaredPidFile => Path.Combine(DataRoot, "cloudflared.pid");

    /// <summary>
    /// Quick Tunnel's assigned https://*.trycloudflare.com address only
    /// ever appears once, printed to cloudflared's own output when it
    /// starts -- captured from there and persisted here so Status can
    /// show the last-known address even before a freshly (re)started
    /// tunnel has printed a new one.
    /// </summary>
    public static string TunnelUrlFile => Path.Combine(DataRoot, "tunnel-url.txt");

    /// <summary>
    /// Optional: a Quick Tunnel address is inherently ephemeral, so an
    /// Operator who wants invited players to be able to find them again
    /// after a restart can point Victory at their own GitHub Gist (their
    /// account, their token, nothing for anyone else to host). The Gist's
    /// id is persisted here once created so later address changes PATCH
    /// the same Gist instead of creating a new one each time.
    /// </summary>
    public static string GistIdFile => Path.Combine(DataRoot, "gist-id.txt");

    /// <summary>
    /// Records which version an apply was last attempted for, written right
    /// before ApplyUpdatesAndRestart is called (UpdateChecker) -- so it's on
    /// disk regardless of whether that call throws, hangs, or actually
    /// replaces the process. A relaunched process's own fresh UpdateManager
    /// can end up seeing the same "new" release as still available (a
    /// version-staleness race, not something this app controls) and would
    /// otherwise reapply it, relaunch, see it again, and repeat -- confirmed
    /// on real hardware as a tight restart loop with no exception ever
    /// logged anywhere, meaning the applies were genuinely succeeding, just
    /// pointlessly repeating.
    /// </summary>
    public static string LastAppliedUpdateMarkerFile => Path.Combine(DataRoot, "last-applied-update.txt");

    /// <summary>
    /// A relaunch on a machine where .env already exists (a fresh Setup.exe
    /// install over previously-configured data, or "Start with Windows")
    /// skips the first-run wizard entirely -- which has its own explicit
    /// "Continue to Create Your Login" moment -- and otherwise announces
    /// nothing at all: the tray icon just quietly appears. Confirmed on real
    /// hardware as a real gap ("there should be at least a pop-up"). This
    /// marker gates a one-time "Victory is running" balloon for exactly that
    /// path so it fires once ever per install, not on every ordinary
    /// relaunch afterward.
    /// </summary>
    public static string FirstLaunchAnnouncedMarkerFile => Path.Combine(DataRoot, "first-launch-announced.txt");

    public static void EnsureDataDirectoriesExist()
    {
        Directory.CreateDirectory(StorageDir);
        Directory.CreateDirectory(BackupDir);
        Directory.CreateDirectory(ExportsDir);
        Directory.CreateDirectory(LogsDir);
    }
}
