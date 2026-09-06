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

    public static void EnsureDataDirectoriesExist()
    {
        Directory.CreateDirectory(StorageDir);
        Directory.CreateDirectory(BackupDir);
        Directory.CreateDirectory(ExportsDir);
        Directory.CreateDirectory(LogsDir);
    }
}
