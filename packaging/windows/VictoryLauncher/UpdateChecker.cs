using System.Net.Http;
using System.Text.Json;
using Velopack;
using Velopack.Sources;

namespace VictoryLauncher;

/// <summary>
/// Kernel 100 §29: "the Operator should not need to download ZIPs,
/// replace source trees, use Git, rebuild containers, or run migrations
/// manually." Kernel 100 §33: "Victory should not auto-restart in the
/// middle of a live Showing."
///
/// Those two requirements pull in different directions for anything that
/// actually changes victory-backend.exe, so this class treats two kinds
/// of release differently:
///
/// - Frontend-only (Caddy serves new static files; the bundled backend
///   binary is byte-identical to what's already running): applied
///   immediately. Caddy restarts as part of the launcher's own relaunch
///   -- sub-second, WebSocket clients auto-reconnect -- but
///   victory-backend.exe is never touched, so nothing an Operator or
///   player was doing is interrupted. The Operator gets a balloon telling
///   them to ask people to refresh (Ctrl+R) using whatever they already
///   use to reach their players -- no new broadcast feature needed.
/// - Backend-changing: downloaded right away, but held until a 2:30am
///   *local-time* quiet window, and only then applied if nothing is
///   actually live (checked via the backend's own /api/system/live-sessions).
///   A manual "Check for Updates" click may apply a backend-changing
///   update immediately instead of waiting for the window, but the
///   live-session check is never skipped -- that one is a safety
///   guarantee, not a courtesy.
///
/// Update source is a separate public repo (github.com/GM-Discovery/victory-releases),
/// not this one: victory itself is private, and GithubSource reading a
/// private repo would mean shipping a GitHub token in every consumer
/// install just to check for updates -- the same private-repo-vs-public-
/// artifact tradeoff already made for the backend's container image
/// (ghcr.io publish).
/// </summary>
internal static class UpdateChecker
{
    private const string ReleaseFeedRepoUrl = "https://github.com/GM-Discovery/victory-releases";
    private const string ReleaseAssetsBaseUrl = "https://github.com/GM-Discovery/victory-releases/releases/download";

    private static readonly TimeSpan QuietWindowStart = new(2, 30, 0);
    private static readonly TimeSpan QuietWindowEnd = new(3, 0, 0);

    private static UpdateManager? _manager;
    private static UpdateInfo? _pendingUpdate;
    private static bool _pendingRequiresBackendRestart;
    private static bool _checkInProgress;

    /// <summary>
    /// True whenever a downloaded update is waiting on the quiet window
    /// (or on a live-session check) rather than having already applied --
    /// TrayApplicationContext uses this to tighten its own check interval,
    /// since the normal 4-hour cadence could easily skip straight over a
    /// 30-minute window entirely.
    /// </summary>
    public static bool HasPendingBackendUpdate => _pendingUpdate is not null && _pendingRequiresBackendRestart;

    /// <summary>
    /// Status shows this directly so "did the update actually take" is
    /// something an Operator can just look at, rather than infer from
    /// whether a feature seems to be there or not.
    /// </summary>
    public static string CurrentVersionText()
    {
        try
        {
            var manager = new UpdateManager(new GithubSource(ReleaseFeedRepoUrl, accessToken: null, prerelease: false));
            if (!manager.IsInstalled)
                return "not a real install";
            return manager.CurrentVersion?.ToString() ?? "unknown";
        }
        catch
        {
            return "unknown";
        }
    }

    public static async Task CheckAndApplyAsync(Action<string> reportStatus, bool manual = false)
    {
        // Confirmed on real hardware: repeated "Check for Updates" clicks
        // while one was already in flight raced multiple full
        // check/download/apply cycles against the same shared state --
        // "downloading" and "restarting" messages interleaving forever,
        // never landing anywhere stable. A second click while one is
        // already running just tells the Operator that plainly instead
        // of starting a competing cycle.
        if (_checkInProgress)
        {
            if (manual)
                reportStatus("Already checking for an update -- give it a moment.");
            return;
        }
        _checkInProgress = true;
        try
        {
            if (!manual && ReadUpdateMode() == "off")
                return;

            _manager ??= new UpdateManager(new GithubSource(ReleaseFeedRepoUrl, accessToken: null, prerelease: false));

            // Running from a raw `dotnet publish` folder (a CI artifact,
            // or a local dev build) rather than a real Velopack-managed
            // install -- there is deliberately nothing to check yet, and
            // asking anyway would just throw. Confirmed on real hardware:
            // a manual "Check for Updates" click here silently did
            // nothing, indistinguishable from broken -- "not sure how I
            // installed" was the exact right question, and this is the
            // silent branch that made it hard to tell.
            if (!_manager.IsInstalled)
            {
                if (manual)
                    reportStatus("This isn't a real Victory install (looks like it's running from an unpacked folder) -- updates can't be checked from here.");
                return;
            }

            if (_pendingUpdate is null)
            {
                var update = await _manager.CheckForUpdatesAsync();
                if (update is null)
                {
                    if (manual)
                        reportStatus("Victory is already up to date.");
                    return;
                }

                reportStatus("Downloading a Victory update...");
                await _manager.DownloadUpdatesAsync(update);

                _pendingUpdate = update;
                _pendingRequiresBackendRestart = await FetchBackendChangedAsync(update.TargetFullRelease.Version.ToString());
            }

            if (!_pendingRequiresBackendRestart)
            {
                await ApplyPendingUpdateAsync(reportStatus, alsoStopBackend: false);
                return;
            }

            // Live-session safety is never skipped, manual or not -- only
            // the quiet-hours *timing* is something a deliberate manual
            // check can bypass.
            if (!await IsSafeToRestartBackendAsync())
            {
                reportStatus("A Victory update is ready, but a Show looks like it's in progress -- it will apply once things are quiet.");
                return;
            }

            if (manual || IsWithinQuietWindow(DateTime.Now.TimeOfDay))
            {
                await ApplyPendingUpdateAsync(reportStatus, alsoStopBackend: true);
            }
            else
            {
                reportStatus("A Victory update is ready -- it will apply automatically overnight (around 2:30am).");
            }
        }
        catch (Exception ex)
        {
            // Update checks are best-effort: a network hiccup, DNS
            // failure, or the release feed not existing yet must never
            // interrupt someone just trying to use Victory -- but a
            // manual, deliberate check should still say *something*
            // rather than fail exactly as silently as "no update found"
            // does, which is indistinguishable from broken.
            if (manual)
                reportStatus("Could not check for updates: " + ex.Message);
        }
        finally
        {
            // Moot if ApplyPendingUpdateAsync actually reached
            // ApplyUpdatesAndRestart (the process is exiting regardless),
            // but every other return path needs this cleared so the next
            // check isn't permanently locked out.
            _checkInProgress = false;
        }
    }

    /// <summary>
    /// Exits this process and relaunches the new version -- there is no
    /// code path after ApplyUpdatesAndRestart that runs. Caddy and the
    /// backend are both independent processes, not children of the
    /// launcher, so the *new* launcher instance would otherwise just see
    /// them as "already running" and leave them alone -- permanently
    /// serving stale files/code from a version directory Velopack is
    /// about to replace. Caddy is always stopped first so it actually
    /// picks up new frontend files on every update, not just
    /// backend-changing ones; the backend is only stopped when this
    /// specific release actually changed it.
    /// </summary>
    private static Task ApplyPendingUpdateAsync(Action<string> reportStatus, bool alsoStopBackend)
    {
        reportStatus("Restarting to finish updating...");
        RuntimeManager.StopCaddy();
        if (alsoStopBackend)
            RuntimeManager.StopBackend();
        _manager!.ApplyUpdatesAndRestart(_pendingUpdate!.TargetFullRelease);
        return Task.CompletedTask;
    }

    private static string ReadUpdateMode()
    {
        if (!EnvGenerator.EnvFileExists())
            return "automatic";
        return EnvGenerator.ReadAll().GetValueOrDefault("UPDATE_MODE", "automatic");
    }

    private static bool IsWithinQuietWindow(TimeSpan localTimeOfDay) => localTimeOfDay >= QuietWindowStart && localTimeOfDay <= QuietWindowEnd;

    /// <summary>
    /// Asks the already-running backend itself, the same one this update
    /// would restart -- not some separate tracked state that could drift.
    /// Unreachable/errors are treated as "not safe," the conservative
    /// default: better to wait an extra cycle than restart into an
    /// unknown state.
    /// </summary>
    private static async Task<bool> IsSafeToRestartBackendAsync()
    {
        try
        {
            using var client = new HttpClient { Timeout = TimeSpan.FromSeconds(5) };
            var response = await client.GetAsync($"http://127.0.0.1:{RuntimeManager.BackendPort}/api/system/live-sessions");
            if (!response.IsSuccessStatusCode)
                return false;
            using var stream = await response.Content.ReadAsStreamAsync();
            using var doc = await JsonDocument.ParseAsync(stream);
            return doc.RootElement.TryGetProperty("live_count", out var liveCount) && liveCount.GetInt32() == 0;
        }
        catch
        {
            return false;
        }
    }

    /// <summary>
    /// CI computes this (windows-installer.yml), since it's the only
    /// place that actually knows what changed in a given push -- guessing
    /// client-side from version numbers alone isn't reliable. Missing
    /// manifest (an older release published before this existed, or
    /// RELEASES_REPO_TOKEN still not configured) defaults to true: safer
    /// to treat an unknown release as backend-changing than to skip the
    /// live-session check on one that actually was.
    /// </summary>
    private static async Task<bool> FetchBackendChangedAsync(string version)
    {
        try
        {
            using var client = new HttpClient { Timeout = TimeSpan.FromSeconds(10) };
            var json = await client.GetStringAsync($"{ReleaseAssetsBaseUrl}/v{version}/update-manifest.json");
            using var doc = JsonDocument.Parse(json);
            return !doc.RootElement.TryGetProperty("backend_changed", out var changed) || changed.GetBoolean();
        }
        catch
        {
            return true;
        }
    }
}
