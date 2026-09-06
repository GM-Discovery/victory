using System.ComponentModel;
using System.Diagnostics;
using System.Threading;

namespace VictoryLauncher;

/// <summary>
/// Enabling WSL2 and installing Podman both require Administrator rights;
/// nothing else in this app does. Rather than manifest the whole tray app
/// as requireAdministrator (which would UAC-prompt on every ordinary
/// launch and defeat "start with Windows" running quietly), this
/// relaunches THIS SAME exe with a hidden `--elevated-step` argument and
/// the "runas" verb, so only that one narrow operation is elevated. See
/// Program.cs's dispatch on that argument for what actually runs.
/// </summary>
internal static class PrivilegedSetup
{
    public enum ElevatedStep
    {
        EnableWsl2,
        InstallPodman,
    }

    public sealed record ElevatedResult(bool Succeeded, bool UserDeclinedElevation, int ExitCode);

    public static async Task<ElevatedResult> RunElevatedAsync(ElevatedStep step)
    {
        var exePath = Environment.ProcessPath
            ?? throw new InvalidOperationException("could not determine this process's own executable path");

        var psi = new ProcessStartInfo
        {
            FileName = exePath,
            Arguments = $"--elevated-step {step}",
            UseShellExecute = true, // required for the "runas" verb to trigger UAC
            Verb = "runas",
        };

        Process process;
        try
        {
            process = Process.Start(psi) ?? throw new InvalidOperationException("failed to start elevated process");
        }
        catch (Win32Exception ex) when (ex.NativeErrorCode == 1223) // ERROR_CANCELLED: user clicked "No" on the UAC prompt
        {
            return new ElevatedResult(Succeeded: false, UserDeclinedElevation: true, ExitCode: -1);
        }

        using (process)
        {
            // These steps download/install real software (winget, wsl
            // --install) -- generous timeout, not the few-second budget
            // used elsewhere for quick local commands.
            using var cts = new CancellationTokenSource(TimeSpan.FromMinutes(15));
            try
            {
                await process.WaitForExitAsync(cts.Token);
            }
            catch (OperationCanceledException)
            {
                try { process.Kill(entireProcessTree: true); } catch { /* best effort */ }
                return new ElevatedResult(Succeeded: false, UserDeclinedElevation: false, ExitCode: -1);
            }
            return new ElevatedResult(Succeeded: process.ExitCode == 0, UserDeclinedElevation: false, ExitCode: process.ExitCode);
        }
    }
}
