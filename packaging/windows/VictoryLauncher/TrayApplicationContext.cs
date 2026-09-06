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
    private const string LocalOperatorUrl = "http://localhost:8081/";

    private readonly NotifyIcon _trayIcon;
    private readonly ToolStripMenuItem _startWithWindowsItem;
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

        menu.Items.Add(new ToolStripSeparator());

        _startWithWindowsItem = new ToolStripMenuItem("Start with Windows") { CheckOnClick = true, Checked = StartupRegistration.IsEnabled() };
        _startWithWindowsItem.Click += (_, _) => StartupRegistration.SetEnabled(_startWithWindowsItem.Checked);
        menu.Items.Add(_startWithWindowsItem);

        menu.Items.Add(new ToolStripSeparator());

        var quitItem = new ToolStripMenuItem("Quit Victory");
        quitItem.Click += async (_, _) => await OnQuitAsync();
        menu.Items.Add(quitItem);

        _trayIcon = new NotifyIcon
        {
            Icon = SystemIcons.Application,
            Text = "Victory",
            Visible = true,
            ContextMenuStrip = menu,
        };
        _trayIcon.DoubleClick += (_, _) => OpenOperatorUi();
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
            using var wizard = new FirstRunWizardForm();
            var result = wizard.ShowDialog();
            if (result != DialogResult.OK || !wizard.Completed)
                return false; // Operator closed the wizard without finishing -- nothing to run yet.

            OpenOperatorUi();
            return true;
        }

        // A relaunch (e.g. "start with Windows" after the reboot WSL2
        // setup itself required) can land here with the runtime still
        // not actually ready yet -- resume that setup automatically
        // (K100 §39) rather than just trying to start compose and
        // failing with a confusing error.
        var setupResult = await RuntimeSetup.EnsureRuntimeReadyAsync(
            step => _trayIcon.ShowBalloonTip(3000, "Victory", step, ToolTipIcon.Info));
        switch (setupResult.Result)
        {
            case RuntimeSetup.Result.RebootRequired:
                StartupRegistration.SetEnabled(true); // so this same setup resumes after the restart
                using (var restartForm = new RestartRequiredForm(setupResult.Detail!))
                    restartForm.ShowDialog();
                return false;
            case RuntimeSetup.Result.ElevationDeclined:
            case RuntimeSetup.Result.Failed:
                _trayIcon.ShowBalloonTip(6000, "Victory", setupResult.Detail ?? "Victory's runtime could not be set up. Click Status for details.", ToolTipIcon.Error);
                return true; // tray icon still shows so the Operator can retry via Status, rather than the process just vanishing
        }

        _trayIcon.ShowBalloonTip(3000, "Victory", "Starting Victory...", ToolTipIcon.Info);
        var (success, output) = await RuntimeManager.StartAsync();
        if (!success)
        {
            _trayIcon.ShowBalloonTip(5000, "Victory", "Victory could not start. Click Status for details.", ToolTipIcon.Error);
        }
        return true;
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
