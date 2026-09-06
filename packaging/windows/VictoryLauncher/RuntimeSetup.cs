using System.IO.Compression;
using System.Net.Http;

namespace VictoryLauncher;

/// <summary>
/// Kernel 100 §2, windows-native branch. Compared to the abandoned
/// Podman/WSL2 approach (see the k100-podman-wsl2-attempt tag), this is
/// dramatically simpler: no elevation, no reboot, no Windows feature
/// changes. Everything here runs as the current user.
///
/// NOT VERIFIED ON REAL WINDOWS HARDWARE. Reasoned through carefully
/// (EDB's own docs describe the Windows binaries zip as intended for
/// exactly this "bundle it into another application's installer" use
/// case), but the extracted zip's exact folder layout and every initdb/
/// pg_ctl flag combination have not been run on a real machine yet.
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

    public enum Result
    {
        Ready,
        Failed,
    }

    public sealed record SetupResult(Result Result, string? Detail);

    public static async Task<SetupResult> EnsureRuntimeReadyAsync(Action<string> reportStep)
    {
        if (!RuntimeManager.IsPostgresInstalled())
        {
            reportStep("Downloading the Victory runtime (this only happens once)");
            var (downloaded, downloadError) = await DownloadAndExtractPostgresAsync();
            if (!downloaded)
                return new SetupResult(Result.Failed, downloadError);
        }

        if (!RuntimeManager.IsDatabaseInitialized())
        {
            reportStep("Setting up private storage");
            var (initialized, initError) = await InitializeDatabaseAsync();
            if (!initialized)
                return new SetupResult(Result.Failed, initError);
        }

        reportStep("Starting Victory");
        var (pgOk, pgOutput) = await RuntimeManager.StartPostgresAsync();
        if (!pgOk)
            return new SetupResult(Result.Failed, "Postgres did not start:\n" + pgOutput);

        var env = EnvGenerator.ReadAll();
        var (dbOk, dbOutput) = await RuntimeManager.CreateVictoryDatabaseAsync(env["POSTGRES_PASSWORD"]);
        if (!dbOk)
            return new SetupResult(Result.Failed, "Could not create the Victory database:\n" + dbOutput);

        var (backendOk, backendOutput) = await RuntimeManager.StartBackendAsync();
        if (!backendOk)
            return new SetupResult(Result.Failed, "Victory could not start:\n" + backendOutput);

        return new SetupResult(Result.Ready, null);
    }

    private static async Task<(bool Success, string? Error)> DownloadAndExtractPostgresAsync()
    {
        var tempZip = Path.Combine(Path.GetTempPath(), "victory-postgres-" + Guid.NewGuid().ToString("N") + ".zip");
        try
        {
            using var client = new HttpClient { Timeout = TimeSpan.FromMinutes(15) };
            using (var response = await client.GetAsync(PostgresDownloadUrl, HttpCompletionOption.ResponseHeadersRead))
            {
                response.EnsureSuccessStatusCode();
                await using var fileStream = File.Create(tempZip);
                await response.Content.CopyToAsync(fileStream);
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
}
