namespace VictoryLauncher;

/// <summary>
/// Kernel 100 §6: application files, user data, generated secrets, and logs
/// are four separate things with different lifetimes. Velopack owns the
/// app-files directory entirely (it gets replaced wholesale on every
/// update and removed on uninstall) -- nothing this app cares about
/// persisting may live there. Everything in this class lives under
/// %ProgramData%\Victory instead, which Velopack's installer/uninstaller
/// never touches, satisfying §7 (data survives upgrades) and §8 (routine
/// uninstall preserves it) by construction rather than by an uninstaller
/// carve-out.
/// </summary>
internal static class AppPaths
{
    public static string DataRoot { get; } =
        Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.CommonApplicationData), "Victory");

    public static string StorageDir => Path.Combine(DataRoot, "data", "storage");
    public static string BackupDir => Path.Combine(DataRoot, "data", "backups");
    public static string ExportsDir => Path.Combine(DataRoot, "data", "exports");
    public static string LogsDir => Path.Combine(DataRoot, "logs");

    /// <summary>
    /// Generated config/secrets (K100 §10-11): persistent, not checked in,
    /// and -- unlike the Podman proof-of-concept's generate-env.sh, which
    /// only had a single developer's own filesystem permissions to worry
    /// about -- this file's ACL is tightened explicitly in EnvGenerator,
    /// since a shared consumer machine may have other Windows accounts.
    /// </summary>
    public static string EnvFile => Path.Combine(DataRoot, ".env");

    public static string ComposeFile
    {
        get
        {
            // The compose file ships as app content alongside the launcher
            // exe -- Velopack's publish output, not %ProgramData% -- since
            // it's part of "what version of Victory is this," not user data.
            var dir = AppContext.BaseDirectory;
            return Path.Combine(dir, "podman", "compose.yml");
        }
    }

    public static void EnsureDataDirectoriesExist()
    {
        Directory.CreateDirectory(StorageDir);
        Directory.CreateDirectory(BackupDir);
        Directory.CreateDirectory(ExportsDir);
        Directory.CreateDirectory(LogsDir);
    }
}
