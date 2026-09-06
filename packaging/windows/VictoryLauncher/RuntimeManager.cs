using System.Diagnostics;
using System.Net.Http;
using System.Threading;

namespace VictoryLauncher;

/// <summary>
/// Kernel 100 §2-3: "Victory manages its runtime" -- the Operator never
/// runs `podman` themselves. This wraps the exact compose stack proven
/// out in packaging/podman (compose.yml, the same env var contract) so a
/// Windows install and a Linux dev install are running identical
/// Victory, not a parallel Windows-specific stack.
///
/// Installing Podman itself when it's missing (including WSL2
/// enablement and reboot handling, §2) is RuntimeSetup's job, not this
/// class's -- that needs elevation and this class deliberately never
/// does, so a normal (non-admin) tray-icon action can never silently
/// trigger a UAC prompt. DetectPodmanAsync here only detects; see
/// RuntimeSetup.EnsureRuntimeReadyAsync for the install flow.
/// </summary>
internal static class RuntimeManager
{
    public sealed record PodmanCheckResult(bool Found, string? Version, string? Error);

    public static async Task<PodmanCheckResult> DetectPodmanAsync()
    {
        try
        {
            var (exitCode, stdout, stderr) = await RunAsync("podman", "--version", TimeSpan.FromSeconds(10));
            if (exitCode != 0)
                return new PodmanCheckResult(false, null, string.IsNullOrWhiteSpace(stderr) ? "podman --version failed" : stderr);
            return new PodmanCheckResult(true, stdout.Trim(), null);
        }
        catch (Exception ex) when (ex is System.ComponentModel.Win32Exception or FileNotFoundException)
        {
            return new PodmanCheckResult(false, null, "Podman is not installed or not on PATH.");
        }
    }

    // Published by .github/workflows/backend-image.yml on every push to
    // main that touches backend/. That workflow's GITHUB_TOKEN can push
    // the image but cannot change its visibility -- until someone with
    // admin on the package has run the one-time `gh api` visibility
    // change documented in that workflow's header, this reference is
    // correct but not yet pullable by a real consumer install (a private
    // GHCR package requires credentials no install should ship with).
    private const string BackendImage = "ghcr.io/gm-discovery/victory-backend:latest";

    public static async Task<(bool Success, string Output)> StartAsync()
    {
        AppPaths.EnsureDataDirectoriesExist();
        var args = $"compose --env-file \"{AppPaths.EnvFile}\" -f \"{AppPaths.ComposeFile}\" up -d";
        var (exitCode, stdout, stderr) = await RunAsync("podman", args, TimeSpan.FromMinutes(5), BackendImage);
        return (exitCode == 0, exitCode == 0 ? stdout : stderr);
    }

    public static async Task<(bool Success, string Output)> StopAsync()
    {
        var args = $"compose --env-file \"{AppPaths.EnvFile}\" -f \"{AppPaths.ComposeFile}\" down";
        var (exitCode, stdout, stderr) = await RunAsync("podman", args, TimeSpan.FromMinutes(2));
        return (exitCode == 0, exitCode == 0 ? stdout : stderr);
    }

    /// <summary>
    /// Run once, right after Podman itself is first installed: unlike
    /// Linux, Windows Podman needs an explicit WSL2-backed machine before
    /// `podman compose` has anywhere to run. `machine init` is expected
    /// to fail with "already exists" on every call after the first --
    /// treated as success here rather than requiring a separate existence
    /// check, since that failure mode is unambiguous and harmless.
    /// </summary>
    public static async Task<(bool Success, string Output)> InitializeMachineAsync()
    {
        var (initExit, initOut, initErr) = await RunAsync("podman", "machine init", TimeSpan.FromMinutes(10));
        if (initExit != 0 && !initErr.Contains("already exists", StringComparison.OrdinalIgnoreCase))
            return (false, initErr);

        var (startExit, startOut, startErr) = await RunAsync("podman", "machine start", TimeSpan.FromMinutes(5));
        if (startExit != 0 && !startErr.Contains("already running", StringComparison.OrdinalIgnoreCase))
            return (false, startErr);

        return (true, initOut + startOut);
    }

    /// <summary>
    /// Backend health, for the consumer status surface (K100 §26) --
    /// never surface raw container state as the primary signal, per that
    /// section's "do not expose container internals" rule.
    /// </summary>
    public static async Task<bool> IsBackendHealthyAsync(string baseUrl = "http://127.0.0.1:8081")
    {
        try
        {
            using var client = new HttpClient { Timeout = TimeSpan.FromSeconds(3) };
            var response = await client.GetAsync($"{baseUrl}/health");
            return response.IsSuccessStatusCode;
        }
        catch
        {
            return false;
        }
    }

    private static async Task<(int ExitCode, string Stdout, string Stderr)> RunAsync(
        string fileName, string arguments, TimeSpan timeout, string? backendImage = null)
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
        if (backendImage is not null)
            psi.Environment["VICTORY_BACKEND_IMAGE"] = backendImage;

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
