using System.Diagnostics;
using System.Threading;

namespace VictoryLauncher;

/// <summary>
/// The actual privileged operations, run ONLY inside the elevated child
/// process Program.cs spawns via PrivilegedSetup -- never in the normal
/// tray process. Each step does exactly one thing and returns a process
/// exit code; no UI, no tray icon, nothing else running here.
/// </summary>
internal static class ElevatedSteps
{
    /// <summary>
    /// `wsl --install --no-distribution` is Microsoft's own current
    /// all-in-one command: enables the WSL and Virtual Machine Platform
    /// Windows features, installs/updates the WSL2 kernel, and sets WSL2
    /// as the default version -- all in one elevated call, rather than
    /// this app hand-rolling separate DISM feature-enable calls that
    /// would need to be kept in sync with whatever Microsoft's own
    /// installer does next. Still requires a reboot afterward; this
    /// process only requests the change, it does not reboot anything
    /// itself (K100 §39: tell the user why, offer restart, don't do it
    /// out from under them).
    /// </summary>
    public static async Task<int> EnableWsl2Async()
    {
        return await RunToCompletionAsync("wsl.exe", "--install --no-distribution", TimeSpan.FromMinutes(10));
    }

    /// <summary>
    /// Installs the Podman CLI via winget (built into Windows 10 2004+/
    /// Windows 11 -- no separate download/checksum management this app
    /// has to own). Deliberately not pinned to a specific version: a
    /// fresh install should get current stable, not a version frozen at
    /// whenever this code was written.
    /// </summary>
    public static async Task<int> InstallPodmanAsync()
    {
        return await RunToCompletionAsync(
            "winget.exe",
            "install --id Podman.CLI --silent --accept-package-agreements --accept-source-agreements",
            TimeSpan.FromMinutes(10));
    }

    private static async Task<int> RunToCompletionAsync(string fileName, string arguments, TimeSpan timeout)
    {
        var psi = new ProcessStartInfo
        {
            FileName = fileName,
            Arguments = arguments,
            UseShellExecute = false,
            CreateNoWindow = true,
        };

        using var process = Process.Start(psi);
        if (process is null)
            return -1;

        using var cts = new CancellationTokenSource(timeout);
        try
        {
            await process.WaitForExitAsync(cts.Token);
        }
        catch (OperationCanceledException)
        {
            try { process.Kill(entireProcessTree: true); } catch { /* best effort */ }
            return -1;
        }
        return process.ExitCode;
    }
}
