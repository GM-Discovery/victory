using System.Diagnostics;
using System.Threading;

namespace VictoryLauncher;

/// <summary>
/// Kernel 100 §2: "Victory manages its runtime." Owns the whole
/// missing-Podman path end to end -- checking WSL2, elevating to enable
/// it, elevating to install Podman, and starting its machine -- so
/// nothing above this class ever needs to know Podman didn't exist yet.
///
/// NOT VERIFIED ON REAL WINDOWS HARDWARE. GitHub's hosted windows-latest
/// CI runners cannot exercise WSL2 feature enablement or elevation
/// prompts (no nested virtualization, no interactive session for UAC) --
/// this code compiles and its control flow has been reasoned through
/// carefully, but "compiles" is not "works." This needs a real Windows
/// machine before it reaches an actual tester, ideally run by a human
/// who can watch what Windows actually does at each elevated step.
/// </summary>
internal static class RuntimeSetup
{
    public enum Result
    {
        Ready,
        RebootRequired,
        ElevationDeclined,
        Failed,
    }

    public sealed record SetupResult(Result Result, string? Detail);

    public static async Task<SetupResult> EnsureRuntimeReadyAsync(Action<string> reportStep)
    {
        reportStep("Checking for the Victory runtime");
        if ((await RuntimeManager.DetectPodmanAsync()).Found)
            return new SetupResult(Result.Ready, null);

        reportStep("Checking Windows compatibility");
        if (!await IsWsl2AvailableAsync())
        {
            reportStep("Setting up a required Windows feature (WSL2) -- Windows will ask you to approve this");
            var wslResult = await PrivilegedSetup.RunElevatedAsync(PrivilegedSetup.ElevatedStep.EnableWsl2);
            if (wslResult.UserDeclinedElevation)
                return new SetupResult(Result.ElevationDeclined, "Victory needs permission to enable a required Windows feature (WSL2) to continue.");
            if (!wslResult.Succeeded)
                return new SetupResult(Result.Failed, $"Enabling WSL2 did not succeed (exit code {wslResult.ExitCode}).");

            // wsl --install always requires a reboot to finish taking
            // effect, even when it reports success -- there is no
            // "succeeded and no reboot needed" outcome to distinguish
            // here (K100 §39).
            return new SetupResult(Result.RebootRequired, "Windows needs to restart to finish enabling a feature Victory depends on (WSL2).");
        }

        reportStep("Installing the Victory runtime -- Windows will ask you to approve this");
        var podmanInstall = await PrivilegedSetup.RunElevatedAsync(PrivilegedSetup.ElevatedStep.InstallPodman);
        if (podmanInstall.UserDeclinedElevation)
            return new SetupResult(Result.ElevationDeclined, "Victory needs permission to install its runtime (Podman) to continue.");
        if (!podmanInstall.Succeeded)
            return new SetupResult(Result.Failed, $"Installing Podman did not succeed (exit code {podmanInstall.ExitCode}).");

        // The elevated winget install just updated the machine/user PATH
        // in the registry -- this already-running process's own cached
        // environment block does not pick that up on its own, so the
        // very next DetectPodmanAsync would still report "not found"
        // without this. A brand new process (e.g. after the eventual
        // restart-with-Windows relaunch) would not have needed it, but
        // this same run does.
        RefreshProcessPathFromRegistry();

        reportStep("Starting the Victory runtime for the first time");
        var (machineOk, machineOutput) = await RuntimeManager.InitializeMachineAsync();
        if (!machineOk)
            return new SetupResult(Result.Failed, "Podman installed, but its machine could not start:\n" + machineOutput);

        return (await RuntimeManager.DetectPodmanAsync()).Found
            ? new SetupResult(Result.Ready, null)
            : new SetupResult(Result.Failed, "Podman was installed but still cannot be found on PATH.");
    }

    private static async Task<bool> IsWsl2AvailableAsync()
    {
        try
        {
            var psi = new ProcessStartInfo
            {
                FileName = "wsl.exe",
                Arguments = "--status",
                UseShellExecute = false,
                CreateNoWindow = true,
                RedirectStandardOutput = true,
                RedirectStandardError = true,
            };
            using var process = Process.Start(psi);
            if (process is null)
                return false;

            using var cts = new CancellationTokenSource(TimeSpan.FromSeconds(10));
            await process.WaitForExitAsync(cts.Token);
            // Exit code alone, not the (possibly localized) text output:
            // `wsl --status` succeeds only once WSL is actually installed
            // and functional, regardless of display language.
            return process.ExitCode == 0;
        }
        catch
        {
            return false;
        }
    }

    private static void RefreshProcessPathFromRegistry()
    {
        var machinePath = Environment.GetEnvironmentVariable("PATH", EnvironmentVariableTarget.Machine) ?? "";
        var userPath = Environment.GetEnvironmentVariable("PATH", EnvironmentVariableTarget.User) ?? "";
        Environment.SetEnvironmentVariable("PATH", machinePath + ";" + userPath, EnvironmentVariableTarget.Process);
    }
}
