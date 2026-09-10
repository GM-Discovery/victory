namespace VictoryLauncher;

/// <summary>
/// Kernel 100 §27-28: Victory behaves like installed software -- a tray
/// icon the Operator can start/stop from, start-with-Windows, reopen the
/// Operator UI, and never spawn a second stack by clicking twice. This is
/// the whole UI surface outside first-run and status; there is
/// deliberately no main window.
/// </summary>
internal sealed class TrayApplicationContext : ApplicationContext
{
    private static readonly string LocalOperatorUrl = $"http://localhost:{RuntimeManager.PublicPort}/";

    // K100 §33: once every 4 hours is fine while nothing is actually
    // waiting on anything. Once UpdateChecker has a downloaded
    // backend-changing update waiting for the 2:30am-local quiet window,
    // that cadence is far too coarse -- a tick that happens to land at,
    // say, 2:15 and next at 6:15 would skip straight over the entire
    // 30-minute window. Tightened to every 15 minutes for exactly that
    // waiting period, then relaxed back once applied.
    private static readonly TimeSpan UpdateCheckInterval = TimeSpan.FromHours(4);
    private static readonly TimeSpan PendingUpdateCheckInterval = TimeSpan.FromMinutes(15);

    private readonly NotifyIcon _trayIcon;
    private readonly ToolStripMenuItem _startWithWindowsItem;
    private readonly System.Windows.Forms.Timer _updateTimer = new() { Interval = (int)UpdateCheckInterval.TotalMilliseconds };
    private StatusForm? _statusForm;

    public TrayApplicationContext()
    {
        var menu = new ContextMenuStrip();

        var openItem = new ToolStripMenuItem("Open Victory");
        openItem.Click += (_, _) => OpenOperatorUi();
        menu.Items.Add(openItem);

        var statusItem = new ToolStripMenuItem("Status...");
        statusItem.Click += (_, _) => ShowStatus();
        menu.Items.Add(statusItem);

        var checkUpdatesItem = new ToolStripMenuItem("Check for Updates");
        // Manual: a deliberate click may apply a backend-changing update
        // right away instead of waiting for the 2:30am window -- the
        // Operator is actively here for it. Never skips the live-session
        // safety check regardless (UpdateChecker's own rule, not this
        // caller's to relax).
        checkUpdatesItem.Click += async (_, _) => await CheckForUpdatesAsync(manual: true);
        menu.Items.Add(checkUpdatesItem);

        menu.Items.Add(new ToolStripSeparator());

        _startWithWindowsItem = new ToolStripMenuItem("Start with Windows") { CheckOnClick = true, Checked = StartupRegistration.IsEnabled() };
        _startWithWindowsItem.Click += (_, _) => StartupRegistration.SetEnabled(_startWithWindowsItem.Checked);
        menu.Items.Add(_startWithWindowsItem);

        menu.Items.Add(new ToolStripSeparator());

        var quitItem = new ToolStripMenuItem("Quit Victory");
        quitItem.Click += async (_, _) => await OnQuitAsync();
        menu.Items.Add(quitItem);

        // Visible = false at construction, deliberately: a tray icon
        // that exists but is doing nothing useful yet is still something
        // to click, and every real bug found on hardware so far in this
        // whole runtime-startup sequence traced back to exactly that --
        // clicking Open Victory (or right-clicking the menu at all)
        // before Postgres/backend/Caddy were all actually confirmed
        // running. If there's nothing in the tray to click, there's
        // nothing to click too early. ShowTrayIconNowReady() is the only
        // place this flips to true.
        _trayIcon = new NotifyIcon
        {
            Icon = AppIcon.Shared,
            Text = "Victory",
            Visible = false,
            ContextMenuStrip = menu,
        };
        _trayIcon.DoubleClick += (_, _) => OpenOperatorUi();

        _updateTimer.Tick += async (_, _) => await CheckForUpdatesAsync(manual: false);
    }

    /// <summary>
    /// Runs first-run setup if needed, then starts the runtime in the
    /// background. Kept separate from the constructor so Program.cs can
    /// await it before entering the message loop.
    /// </summary>
    public async Task<bool> InitializeAsync()
    {
        if (!EnvGenerator.EnvFileExists())
        {
            // Checked before the wizard even opens: failing obscurely
            // partway through a 150+MB Postgres download or an initdb call
            // with no disk left is a much worse first impression than
            // saying up front this machine can't run Victory yet.
            var prereqs = HostPrerequisites.Check();
            if (!prereqs.Ok)
            {
                MessageBox.Show(
                    prereqs.Detail,
                    "Victory can't set up on this computer",
                    MessageBoxButtons.OK,
                    MessageBoxIcon.Error);
                return false;
            }

            using var wizard = new FirstRunWizardForm();
            // wizard.Completed alone, not ShowDialog()'s own DialogResult:
            // closing the window via the X button after setup finished
            // but before clicking "Continue" leaves DialogResult at its
            // WinForms default (Cancel) even though Victory's runtime is
            // now genuinely up and running -- checking DialogResult too
            // would wrongly exit the whole tray app over a still-running
            // install with no UI left to manage it.
            wizard.ShowDialog();
            if (!wizard.Completed)
                return false; // Operator closed the wizard without finishing -- nothing to run yet.

            // Only now: Postgres, the database, the backend, and Caddy
            // are all confirmed running (that's what wizard.Completed
            // means -- it only becomes true after the wizard's own
            // "waiting for Victory to be ready" check succeeds).
            _trayIcon.Visible = true;

            // The wizard's own "Continue to Create Your Login" button
            // already opened the browser to /signup/?handle= directly,
            // as a real user gesture -- nothing left to do here if they
            // clicked it. If they closed the window instead without
            // clicking it, Victory is still running, but there is
            // currently no other prompt anywhere reminding them an
            // account still needs creating -- "Open Victory" from the
            // tray just goes to the map. Known gap, not yet worth a
            // second reminder mechanism for what should be a rare path
            // (the button is the obvious, labeled thing to click).
            return true;
        }

        // A relaunch (e.g. after a crash, or "start with Windows") can
        // land here with Postgres not yet downloaded/initialized, or not
        // currently running -- EnsureRuntimeReadyAsync handles every one
        // of those states itself and leaves things running when it
        // returns Ready, so this one call covers both "totally fresh"
        // and "resuming from wherever it left off." No step-by-step
        // balloon tips here: the icon isn't visible yet (nothing to
        // anchor a balloon to), and that's deliberate -- see the tray
        // icon's own Visible=false comment.
        var setupResult = await RuntimeSetup.EnsureRuntimeReadyAsync(reportStep: _ => { });
        _trayIcon.Visible = true;
        if (setupResult.Result == RuntimeSetup.Result.Failed)
        {
            _trayIcon.ShowBalloonTip(6000, "Victory", setupResult.Detail ?? "Victory's runtime could not be set up. Click Status for details.", ToolTipIcon.Error);
            return true; // tray icon still shows so the Operator can retry via Status, rather than the process just vanishing
        }

        // A no-op on every ordinary launch; fires exactly once, right
        // after a launch that was actually the result of a successful
        // update applying and relaunching this process.
        UpdateChecker.AnnounceIfJustUpdated(status => _trayIcon.ShowBalloonTip(5000, "Victory", status, ToolTipIcon.Info));

        // This branch (.env already exists) skips the first-run wizard
        // entirely, including its own explicit "Continue to Create Your
        // Login" moment -- a fresh Setup.exe install over previously-
        // configured data otherwise announces nothing at all beyond the
        // tray icon quietly appearing. Confirmed on real hardware as a
        // real gap. Fires once ever per install, gated by the marker file.
        try
        {
            if (!File.Exists(AppPaths.FirstLaunchAnnouncedMarkerFile))
            {
                _trayIcon.ShowBalloonTip(5000, "Victory", "Victory is running -- use this tray icon to open it.", ToolTipIcon.Info);
                Directory.CreateDirectory(AppPaths.DataRoot);
                File.WriteAllText(AppPaths.FirstLaunchAnnouncedMarkerFile, DateTime.UtcNow.ToString("O"));
            }
        }
        catch { /* best effort -- a missed one-time balloon isn't worth failing startup over */ }

        // Only reached once Victory is actually up and running -- never
        // during first-run setup itself (that path returns earlier,
        // above) and never mid-retry after a failure (the branch just
        // above also returns first). ApplyUpdatesAndRestart itself only
        // restarts this launcher process, not the Postgres/backend/Caddy
        // processes it started (those aren't child processes of this one
        // in any way Windows would tear down together) -- but a
        // backend-changing update does now explicitly restart
        // victory-backend.exe too (UpdateChecker), which is exactly the
        // disruption §33's quiet-window/live-session gating exists for.
        // A frontend-only update never touches it, so those really are
        // interruption-free.
        _updateTimer.Start();
        _ = CheckForUpdatesAsync(manual: false);

        return true;
    }

    private async Task CheckForUpdatesAsync(bool manual)
    {
        await UpdateChecker.CheckAndApplyAsync(
            status => _trayIcon.ShowBalloonTip(3000, "Victory", status, ToolTipIcon.Info),
            manual);
        // Tightened while something is actually waiting on the quiet
        // window; relaxed back once it's applied (or once it turns out
        // there was nothing pending after all).
        _updateTimer.Interval = (int)(UpdateChecker.HasPendingBackendUpdate ? PendingUpdateCheckInterval : UpdateCheckInterval).TotalMilliseconds;
    }

    private void OpenOperatorUi()
    {
        // "Clicking Victory should either start services then open the
        // local Operator interface, or focus/open the already-running
        // instance" (K100 §28) -- Start is idempotent, so this is safe to
        // call whether or not Victory is already running, and a plain
        // browser launch is "focus" for a normal person: their browser
        // already has the tab, or opens a new one to the same place.
        _ = RuntimeManager.StartAsync();
        try
        {
            System.Diagnostics.Process.Start(new System.Diagnostics.ProcessStartInfo(LocalOperatorUrl) { UseShellExecute = true });
        }
        catch
        {
            _trayIcon.ShowBalloonTip(4000, "Victory", "Could not open your browser. Visit " + LocalOperatorUrl + " manually.", ToolTipIcon.Warning);
        }
    }

    private void ShowStatus()
    {
        if (_statusForm is { IsDisposed: false })
        {
            _statusForm.Activate();
            return;
        }
        _statusForm = new StatusForm();
        _statusForm.FormClosed += (_, _) => _statusForm = null;
        _statusForm.Show();
    }

    private async Task OnQuitAsync()
    {
        _trayIcon.Visible = false;
        // Deliberately does NOT stop the compose stack: quitting the tray
        // icon should not take down a lot other people may be connected
        // to remotely. "Stop Victory" (Status window) is the explicit,
        // separate action for that -- matches §27's start/stop/background
        // distinction rather than conflating "close the launcher" with
        // "shut down the server."
        await Task.CompletedTask;
        Application.Exit();
    }
}
