using Velopack;
using Velopack.Sources;

namespace VictoryLauncher;

/// <summary>
/// Kernel 100 §29: "the Operator should not need to download ZIPs,
/// replace source trees, use Git, rebuild containers, or run migrations
/// manually." Automatic install mode (§29's simplest required mode):
/// checks, downloads, and applies without asking, then restarts.
///
/// Update source is a separate public repo
/// (github.com/GM-Discovery/victory-releases), not this one: victory
/// itself is private, and GithubSource reading a private repo would mean
/// shipping a GitHub token in every consumer install just to check for
/// updates -- the exact same private-repo-vs-public-artifact tradeoff
/// already made for the backend's container image (ghcr.io publish).
/// That repo's actual name is pending Grant's confirmation as of this
/// writing; update ReleaseFeedRepoUrl once it exists.
/// </summary>
internal static class UpdateChecker
{
    private const string ReleaseFeedRepoUrl = "https://github.com/GM-Discovery/victory-releases";

    public static async Task CheckAndApplyAsync(Action<string> reportStatus)
    {
        try
        {
            var manager = new UpdateManager(new GithubSource(ReleaseFeedRepoUrl, accessToken: null, prerelease: false));

            // Running from a raw `dotnet publish` folder (a CI artifact,
            // or a local dev build) rather than a real Velopack-managed
            // install -- there is deliberately nothing to check yet, and
            // asking anyway would just throw.
            if (!manager.IsInstalled)
                return;

            var update = await manager.CheckForUpdatesAsync();
            if (update is null)
                return;

            reportStatus("Downloading a Victory update...");
            await manager.DownloadUpdatesAsync(update);

            reportStatus("Restarting to finish updating...");
            // Exits this process and relaunches the new version -- there
            // is no code path after this call that runs.
            manager.ApplyUpdatesAndRestart(update.TargetFullRelease);
        }
        catch
        {
            // Update checks are best-effort: a network hiccup, DNS
            // failure, or the release feed not existing yet must never
            // interrupt someone just trying to use Victory.
        }
    }
}
