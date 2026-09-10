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
/// Every update, backend-changing or not, downloads right away but is held
/// until a 2:30am *local-time* quiet window, and only then applied if
/// nothing is actually live (checked via the backend's own
/// /api/system/live-sessions). A manual "Check for Updates" click may apply
/// immediately instead of waiting for the window, but the live-session
/// check is never skipped -- that one is a safety guarantee, not a
/// courtesy.
///
/// This used to treat "frontend-only" releases (CI's own backend_changed
/// flag in update-manifest.json false) as exempt from all of the above --
/// applied immediately, victory-backend.exe deliberately left running
/// through the apply since its content genuinely hadn't changed. Confirmed
/// on real hardware as the actual cause of a silent, unrecoverable update
/// failure: CI rebuilds and repackages victory-backend.exe into *every*
/// release regardless of whether backend/ changed, so Velopack still has to
/// place a full new copy of that file as part of the swap -- with the old
/// copy's own process still holding it open the whole time, since nothing
/// had stopped it. Every update now stops the backend and goes through the
/// same live-session/quiet-window gate, full stop -- the "frontend-only,
/// nothing is interrupted" exemption was built on a wrong assumption about
/// what Velopack's update actually touches.
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

    private static readonly TimeSpan QuietWindowStart = new(2, 30, 0);
    private static readonly TimeSpan QuietWindowEnd = new(3, 0, 0);

    private static UpdateManager? _manager;
    private static UpdateInfo? _pendingUpdate;
    private static bool _checkInProgress;

    /// <summary>
    /// True whenever a downloaded update is waiting on the quiet window
    /// (or on a live-session check) rather than having already applied --
    /// TrayApplicationContext uses this to tighten its own check interval,
    /// since the normal 4-hour cadence could easily skip straight over a
    /// 30-minute window entirely. Every pending update needs this now (see
    /// this class's own header comment) -- there is no longer a
    /// frontend-only fast path that skips the wait entirely.
    /// </summary>
    public static bool HasPendingBackendUpdate => _pendingUpdate is not null;

    /// <summary>
    /// Called once on every launch (TrayApplicationContext.InitializeAsync).
    /// A successful ApplyUpdatesAndRestart replaces this process entirely --
    /// there is no code path in the OLD process that runs afterward to
    /// announce anything, and the NEW process starting up looks identical
    /// to any other launch. Confirmed as a real gap: every version check
    /// tonight required opening Status and reading a version number by
    /// hand, nothing ever actually said "this landed." The marker written
    /// right before every apply attempt (MarkAttempted) is repurposed here
    /// as positive confirmation too -- if this version now matches what was
    /// last attempted, that attempt is what got us here. Deleted once
    /// announced so it fires exactly once per successful update, not on
    /// every subsequent unrelated launch.
    /// </summary>
    public static void AnnounceIfJustUpdated(Action<string> reportBalloon)
    {
        try
        {
            var path = AppPaths.LastAppliedUpdateMarkerFile;
            if (!File.Exists(path))
                return;
            var attemptedVersion = File.ReadAllText(path).Trim();
            if (attemptedVersion.Length == 0 || attemptedVersion != CurrentVersionText())
                return; // either no marker, or the last attempt didn't land -- already reported at attempt time
            reportBalloon($"Victory updated to v{attemptedVersion}.");
            File.Delete(path);
        }
        catch
        {
            // Best effort -- a missed announcement isn't worth failing startup over.
        }
    }

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

                // Confirmed on real hardware as a tight restart loop with
                // no exception ever logged anywhere (LogApplyFailure never
                // fired): applying this exact version had already been
                // attempted moments ago -- ApplyUpdatesAndRestart genuinely
                // succeeded, but the freshly-relaunched process's own new
                // UpdateManager still saw the same release as available
                // (a version-staleness race, not something in this app's
                // control) and would otherwise reapply it, relaunch, see it
                // again, and repeat. Skip rather than loop; 10 minutes is
                // long enough for that race to resolve itself and short
                // enough that a genuine future re-release of this exact
                // version number (shouldn't happen, but not this code's
                // job to assume) isn't blocked for long.
                var targetVersion = update.TargetFullRelease.Version.ToString();
                if (WasRecentlyAttempted(targetVersion))
                {
                    if (manual)
                        reportStatus("Victory just updated to this version -- give it a moment to finish starting.");
                    return;
                }

                reportStatus("Downloading a Victory update...");
                await _manager.DownloadUpdatesAsync(update);

                _pendingUpdate = update;
            }

            // Live-session safety is never skipped, manual or not -- only
            // the quiet-hours *timing* is something a deliberate manual
            // check can bypass. Applies to every update now, not just ones
            // CI flagged as backend-changing -- see this class's header
            // comment for why that distinction turned out not to be safe.
            if (!await IsSafeToRestartBackendAsync())
            {
                reportStatus("A Victory update is ready, but a Show looks like it's in progress -- it will apply once things are quiet.");
                return;
            }

            if (manual || IsWithinQuietWindow(DateTime.Now.TimeOfDay))
            {
                await ApplyPendingUpdateAsync(reportStatus);
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
    /// Exits this process and relaunches the new version -- normally
    /// there is no code path after ApplyUpdatesAndRestart that runs. But
    /// if it throws instead of exiting (permissions, antivirus, a locked
    /// file -- something on the machine actually preventing the swap),
    /// two real problems compound: Caddy/backend were already stopped to
    /// prepare for it, so Victory is now down with nothing left to bring
    /// it back up (the process never exited to trigger a fresh start);
    /// and _pendingUpdate stays cached, so every future automatic check
    /// just retries the identical failing apply forever. Confirmed on
    /// real hardware as exactly this: "tried to do .37 again and again
    /// and again," on its own schedule, not from repeated clicking.
    /// </summary>
    private static async Task ApplyPendingUpdateAsync(Action<string> reportStatus)
    {
        reportStatus("Restarting to finish updating...");
        RuntimeManager.StopCaddy();
        // Always, not just for backend-changing releases: CI repackages
        // victory-backend.exe into every release regardless of whether its
        // source changed, so Velopack needs this file released even when
        // the update is otherwise frontend-only. See this class's header
        // comment.
        RuntimeManager.StopBackend();
        // The actual root cause of every update failure tonight, found by
        // Grant: Postgres was never stopped here. Every long-running child
        // process this app starts (Caddy explicitly, the backend and
        // Postgres implicitly) inherits VictoryLauncher.exe's own current
        // directory as ITS working directory whenever ProcessStartInfo
        // doesn't set one -- which is "current\", the exact folder
        // Velopack's updater needs to rename. Caddy and the backend both
        // got stopped already, releasing that lock; postgres.exe, kept
        // alive deliberately by pg_ctl long after pg_ctl itself exits,
        // never did. A process holding a directory as its working
        // directory blocks Windows from renaming it -- no antivirus, no
        // OneDrive, nothing else required to produce exactly the
        // "being used by another process" error confirmed via
        // velopack_VictoryLauncher.log on every single failed attempt.
        await RuntimeManager.StopPostgresAsync();

        // Written before the call below, not after: if it actually
        // succeeds, this process is about to exit, and the marker needs to
        // already be on disk for the next process to find -- there is no
        // reliable "after" in the success case.
        var targetVersion = _pendingUpdate!.TargetFullRelease.Version.ToString();
        MarkAttempted(targetVersion);
        AppendLauncherLog($"about to call ApplyUpdatesAndRestart for v{targetVersion}");

        try
        {
            _manager!.ApplyUpdatesAndRestart(_pendingUpdate!.TargetFullRelease);
            // No code below this line normally runs -- the process just exited.
            AppendLauncherLog("ApplyUpdatesAndRestart returned without exiting the process (unexpected)");
        }
        catch (Exception ex)
        {
            LogApplyFailure(ex);
            // Undo the stop above so Victory stays usable rather than
            // silently down until someone notices and manually restarts
            // it -- EnsureRuntimeReadyAsync is idempotent, safe to call
            // here the same way Status's own Start button does.
            await RuntimeSetup.EnsureRuntimeReadyAsync(reportStep: _ => { });
            // Clear the cached update so this exact failure can't retry
            // itself forever -- the next periodic check starts fresh
            // rather than reusing state tied to whatever just broke.
            _pendingUpdate = null;
            reportStatus("The update couldn't apply (see launcher.log) -- Victory is still running the current version.");
        }
    }

    private static readonly TimeSpan RecentAttemptWindow = TimeSpan.FromMinutes(10);

    private static bool WasRecentlyAttempted(string version)
    {
        try
        {
            var path = AppPaths.LastAppliedUpdateMarkerFile;
            if (!File.Exists(path))
                return false;
            if (DateTime.UtcNow - File.GetLastWriteTimeUtc(path) > RecentAttemptWindow)
                return false;
            return File.ReadAllText(path).Trim() == version;
        }
        catch
        {
            return false; // best effort -- an unreadable marker shouldn't block a real update
        }
    }

    private static void MarkAttempted(string version)
    {
        try
        {
            Directory.CreateDirectory(AppPaths.DataRoot);
            File.WriteAllText(AppPaths.LastAppliedUpdateMarkerFile, version);
        }
        catch { /* best effort -- worst case this specific loop-guard just doesn't engage */ }
    }

    private static void LogApplyFailure(Exception ex) => AppendLauncherLog("update apply failed: " + ex);

    /// <summary>
    /// A checkpoint trail around the risky call, not just its exception:
    /// three silent failures on real hardware in a row produced nothing in
    /// launcher.log at all, meaning the process died somewhere between
    /// "about to call ApplyUpdatesAndRestart" and whatever would have
    /// followed it -- with no managed exception ever thrown on this side.
    /// A line written immediately before that call, flushed synchronously,
    /// at least proves how far execution got even when nothing after it
    /// ever runs.
    /// </summary>
    private static void AppendLauncherLog(string message)
    {
        try
        {
            Directory.CreateDirectory(AppPaths.LogsDir);
            File.AppendAllText(
                Path.Combine(AppPaths.LogsDir, "launcher.log"),
                $"{DateTime.UtcNow:O} {message}\n");
        }
        catch { /* best effort -- the balloon message is the fallback if even logging fails */ }
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
}
