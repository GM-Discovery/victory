namespace VictoryLauncher;

/// <summary>
/// Kernel 100 §26: human-readable status, never raw container internals
/// as the primary view. Remote access (Quick Tunnel) and updates
/// (automatic, with §33's live-Show/quiet-window scheduling for
/// backend-changing releases) are both real here now -- shown as their
/// actual current state, including "downloaded, waiting for quiet
/// hours," rather than a static label, matching §38's "do not fake"
/// doctrine applied to status as well as progress.
/// </summary>
internal sealed class StatusForm : Form
{
    private readonly Label _runningValue = new() { AutoSize = true };
    private readonly Label _databaseValue = new() { AutoSize = true };
    private readonly Label _remoteValue = new() { AutoSize = true, Text = "Not set up yet" };
    private readonly Label _urlValue = new() { AutoSize = true, Text = $"http://localhost:{RuntimeManager.PublicPort} (this device only)" };
    private readonly Label _updateValue = new() { AutoSize = true, Text = "Not set up yet" };
    private readonly Label _discordValue = new() { AutoSize = true };
    private readonly Button _copyAddressButton = new() { Text = "Copy", Width = 50, Height = 22, Visible = false };
    private readonly Button _tunnelToggleButton = new() { Width = 90, Height = 22 };
    private readonly Label _tunnelNoteLabel = new()
    {
        AutoSize = false,
        Size = new Size(380, 28),
        ForeColor = SystemColors.GrayText,
        Font = new Font(FontFamily.GenericSansSerif, 7.5f),
        Visible = false,
    };
    private readonly Button _updateToggleButton = new() { Width = 90, Height = 22 };
    // Only ever shown when TUNNEL_MODE is "quick" -- Quick Tunnel is the
    // one address that's actually ephemeral; "named" already has a
    // stable domain, "off" has no address to rediscover at all.
    private readonly Label _rediscoveryValue = new() { AutoSize = true, Visible = false };
    private readonly Button _rediscoveryToggleButton = new() { Width = 130, Height = 22, Visible = false };
    private readonly Button _copyRediscoveryButton = new() { Text = "Copy", Width = 50, Height = 22, Visible = false };
    private readonly Button _startStopButton = new() { Width = 120 };
    private readonly Button _refreshButton = new() { Text = "Refresh", Width = 100 };

    public StatusForm()
    {
        Text = "Victory Status";
        Icon = AppIcon.Shared;
        FormBorderStyle = FormBorderStyle.FixedDialog;
        MaximizeBox = false;
        MinimizeBox = false;
        StartPosition = FormStartPosition.CenterScreen;
        ClientSize = new Size(420, 330);

        AddRow("Victory:", _runningValue, 20);
        AddRow("Database:", _databaseValue, 50);
        AddRow("Remote access:", _remoteValue, 80);
        AddRow("Address:", _urlValue, 110);
        _tunnelToggleButton.Location = new Point(280, 78);
        _tunnelToggleButton.Click += async (_, _) => await OnTunnelToggleClickedAsync();
        _copyAddressButton.Location = new Point(340, 108);
        _copyAddressButton.Click += (_, _) => CopyAddressToClipboard();
        _tunnelNoteLabel.Location = new Point(20, 132);

        AddRow("Rediscovery:", _rediscoveryValue, 170);
        _rediscoveryToggleButton.Location = new Point(280, 168);
        _rediscoveryToggleButton.Click += async (_, _) => await OnRediscoveryToggleClickedAsync();
        _copyRediscoveryButton.Location = new Point(20, 190);
        _copyRediscoveryButton.Click += (_, _) => CopyRediscoveryLinkToClipboard();

        AddRow("Updates:", _updateValue, 220);
        _updateToggleButton.Location = new Point(280, 218);
        _updateToggleButton.Click += async (_, _) => await OnUpdateToggleClickedAsync();
        AddRow("Discord:", _discordValue, 250);

        _startStopButton.Location = new Point(20, 290);
        _startStopButton.Click += async (_, _) => await OnStartStopClickedAsync();

        _refreshButton.Location = new Point(150, 290);
        _refreshButton.Click += async (_, _) => await RefreshAsync();

        Controls.AddRange([
            _tunnelToggleButton, _copyAddressButton, _tunnelNoteLabel,
            _rediscoveryToggleButton, _copyRediscoveryButton,
            _updateToggleButton, _startStopButton, _refreshButton,
        ]);
        Shown += async (_, _) => await RefreshAsync();
    }

    private void CopyAddressToClipboard()
    {
        try { Clipboard.SetText(_urlValue.Text); }
        catch { /* best effort -- clipboard access can transiently fail on Windows */ }
    }

    /// <summary>
    /// The one place besides the first-run wizard an Operator can change
    /// this decision -- turning it on later downloads cloudflared if this
    /// is the first time (no separate progress UI here; the button just
    /// stays disabled and re-labeled while it works, since this only
    /// happens once per install and Status has no step-list like the
    /// wizard does).
    /// </summary>
    private async Task OnTunnelToggleClickedAsync()
    {
        _tunnelToggleButton.Enabled = false;
        try
        {
            var turningOn = _tunnelToggleButton.Text == "Turn On";
            EnvGenerator.UpdateTunnelMode(turningOn ? "quick" : "off");
            if (turningOn)
            {
                if (!RuntimeManager.IsCloudflaredInstalled())
                {
                    _tunnelToggleButton.Text = "Downloading...";
                    await RuntimeSetup.EnsureRuntimeReadyAsync(reportStep: _ => { });
                }
                else
                {
                    await RuntimeManager.StartQuickTunnelAsync();
                }
            }
            else
            {
                RuntimeManager.StopTunnel();
            }
        }
        finally
        {
            await RefreshAsync();
            _tunnelToggleButton.Enabled = true;
        }
    }

    private void AddRow(string labelText, Label valueLabel, int y)
    {
        var label = new Label { Text = labelText, AutoSize = true, Location = new Point(20, y), Font = new Font(Font, FontStyle.Bold) };
        valueLabel.Location = new Point(140, y);
        Controls.Add(label);
        Controls.Add(valueLabel);
    }

    private bool _isRunning;

    private async Task RefreshAsync()
    {
        _refreshButton.Enabled = false;
        try
        {
            var backendHealthy = await RuntimeManager.IsBackendHealthyAsync();
            var publicSiteReachable = await RuntimeManager.IsPublicSiteReachableAsync();
            // "Running" means what an Operator actually cares about:
            // can they reach Victory at all. Both need to be true --
            // a healthy backend behind a dead Caddy is exactly the
            // 404-while-healthy state this whole fix exists for.
            _isRunning = backendHealthy && publicSiteReachable;
            _runningValue.Text = _isRunning ? "Running" : "Stopped";
            _databaseValue.Text = backendHealthy ? "Healthy" : "Not running";
            _discordValue.Text = ReadDiscordConfigured() ? "Connected" : "Not configured";
            _startStopButton.Text = _isRunning ? "Stop Victory" : "Start Victory";
            RefreshTunnelDisplay();
            RefreshUpdateDisplay();
        }
        finally
        {
            _refreshButton.Enabled = true;
        }
    }

    private async Task OnStartStopClickedAsync()
    {
        _startStopButton.Enabled = false;
        try
        {
            if (_isRunning)
                await RuntimeManager.StopAsync();
            else
                await RuntimeManager.StartAsync();
        }
        finally
        {
            await RefreshAsync();
            _startStopButton.Enabled = true;
        }
    }

    /// <summary>
    /// K100: "provision an interface for the operator to switch to it and
    /// have their players be able to find the new URL easily... probably
    /// with a note suggesting the reliability of the hosting
    /// program/computer will determine the stability of the URL." Three
    /// states an Operator can actually be in: never turned it on, turned
    /// it on and it's live, or turned it on but it hasn't connected yet
    /// (still worth showing as distinct from "off," not lumped together).
    /// </summary>
    private void RefreshTunnelDisplay()
    {
        var env = EnvGenerator.EnvFileExists() ? EnvGenerator.ReadAll() : new Dictionary<string, string>();
        var tunnelMode = env.GetValueOrDefault("TUNNEL_MODE", "off");
        if (tunnelMode == "off")
        {
            _remoteValue.Text = "Off (this device only)";
            _urlValue.Text = $"http://localhost:{RuntimeManager.PublicPort} (this device only)";
            _copyAddressButton.Visible = false;
            _tunnelNoteLabel.Visible = false;
            _tunnelToggleButton.Visible = true;
            _tunnelToggleButton.Text = "Turn On";
            RefreshRediscoveryDisplay(env);
            return;
        }

        if (tunnelMode == "named")
        {
            // Set up once via the first-run wizard, using an Operator's
            // own pre-existing Cloudflare account -- Status can turn it
            // off, but there's nowhere here to collect a token/hostname
            // to turn it back on with, so no on-ramp back to "named" is
            // offered from this screen. Not a real gap in practice: an
            // Operator who set this up once still has their own token.
            var hostname = env.GetValueOrDefault("CLOUDFLARE_TUNNEL_HOSTNAME", "");
            _remoteValue.Text = RuntimeManager.IsTunnelRunning() ? "On" : "On, but not connected yet";
            _urlValue.Text = hostname.Length > 0 ? $"https://{hostname}" : "(configured, no address on record)";
            _copyAddressButton.Visible = hostname.Length > 0;
            _tunnelNoteLabel.Visible = false;
            _tunnelToggleButton.Visible = true;
            _tunnelToggleButton.Text = "Turn Off";
            RefreshRediscoveryDisplay(env);
            return;
        }

        _tunnelToggleButton.Visible = true;
        _tunnelToggleButton.Text = "Turn Off";

        var currentUrl = RuntimeManager.GetLastKnownTunnelUrl();
        if (RuntimeManager.IsTunnelRunning() && currentUrl is not null)
        {
            _remoteValue.Text = "On";
            _urlValue.Text = currentUrl;
            _copyAddressButton.Visible = true;
            _tunnelNoteLabel.Text = "This free address changes if this computer restarts -- how stable it stays depends on how reliable this machine and its connection are.";
            _tunnelNoteLabel.Visible = true;
        }
        else
        {
            _remoteValue.Text = "On, but not connected yet";
            _urlValue.Text = currentUrl ?? "(waiting for an address...)";
            _copyAddressButton.Visible = currentUrl is not null;
            _tunnelNoteLabel.Text = "Victory hasn't gotten a public address yet -- click Refresh in a moment, or check your internet connection.";
            _tunnelNoteLabel.Visible = true;
        }

        RefreshRediscoveryDisplay(env);
    }

    /// <summary>
    /// Quick Tunnel only -- an optional fix for its one real weakness
    /// (the address doesn't survive a restart). An Operator's own GitHub
    /// Gist, kept updated automatically once they provide a token
    /// (gist-scope only). Off by default; nothing for anyone but the
    /// Operator to host or be responsible for.
    /// </summary>
    private void RefreshRediscoveryDisplay(Dictionary<string, string> env)
    {
        var tunnelMode = env.GetValueOrDefault("TUNNEL_MODE", "off");
        if (tunnelMode != "quick")
        {
            _rediscoveryValue.Visible = false;
            _rediscoveryToggleButton.Visible = false;
            _copyRediscoveryButton.Visible = false;
            return;
        }

        _rediscoveryValue.Visible = true;
        _rediscoveryToggleButton.Visible = true;

        var hasToken = !string.IsNullOrWhiteSpace(env.GetValueOrDefault("GITHUB_GIST_TOKEN", ""));
        if (!hasToken)
        {
            _rediscoveryValue.Text = "Off";
            _rediscoveryToggleButton.Text = "Set Up...";
            _copyRediscoveryButton.Visible = false;
            return;
        }

        _rediscoveryToggleButton.Text = "Turn Off";
        var gistId = File.Exists(AppPaths.GistIdFile) ? File.ReadAllText(AppPaths.GistIdFile).Trim() : "";
        if (gistId.Length > 0)
        {
            _rediscoveryValue.Text = $"https://gist.github.com/{gistId}";
            _copyRediscoveryButton.Visible = true;
        }
        else
        {
            _rediscoveryValue.Text = "On -- waiting for the first address to record";
            _copyRediscoveryButton.Visible = false;
        }
    }

    private void CopyRediscoveryLinkToClipboard()
    {
        try { Clipboard.SetText(_rediscoveryValue.Text); }
        catch { /* best effort -- clipboard access can transiently fail on Windows */ }
    }

    private async Task OnRediscoveryToggleClickedAsync()
    {
        _rediscoveryToggleButton.Enabled = false;
        try
        {
            if (_rediscoveryToggleButton.Text == "Turn Off")
            {
                EnvGenerator.UpdateGistToken("");
                return;
            }

            var token = PromptForToken(
                "Set Up Address Rediscovery",
                "Paste a GitHub token with only the \"gist\" scope. Create one at github.com/settings/tokens -- Victory will use it solely to keep one private Gist updated with your current address.");
            if (string.IsNullOrWhiteSpace(token))
                return;
            EnvGenerator.UpdateGistToken(token.Trim());
        }
        finally
        {
            await RefreshAsync();
            _rediscoveryToggleButton.Enabled = true;
        }
    }

    /// <summary>
    /// A minimal, self-contained prompt -- this is the one place Status
    /// needs to collect a secret rather than just toggle a mode, and it
    /// isn't worth a whole separate form file for one text field.
    /// </summary>
    private static string? PromptForToken(string title, string message)
    {
        using var prompt = new Form
        {
            Text = title,
            FormBorderStyle = FormBorderStyle.FixedDialog,
            MaximizeBox = false,
            MinimizeBox = false,
            StartPosition = FormStartPosition.CenterParent,
            ClientSize = new Size(380, 160),
        };
        var messageLabel = new Label { Text = message, AutoSize = false, Size = new Size(340, 70), Location = new Point(20, 16) };
        var tokenBox = new TextBox { Location = new Point(20, 92), Width = 340, UseSystemPasswordChar = true };
        var okButton = new Button { Text = "Save", Location = new Point(200, 122), DialogResult = DialogResult.OK };
        var cancelButton = new Button { Text = "Cancel", Location = new Point(285, 122), DialogResult = DialogResult.Cancel };
        prompt.Controls.AddRange([messageLabel, tokenBox, okButton, cancelButton]);
        prompt.AcceptButton = okButton;
        prompt.CancelButton = cancelButton;
        return prompt.ShowDialog() == DialogResult.OK ? tokenBox.Text : null;
    }

    /// <summary>
    /// Grant: "people should be able to opt out of updates." Off stops
    /// the periodic automatic check entirely -- the tray's own "Check for
    /// Updates" menu item still works regardless, since a manual click is
    /// inherently consensual. Also shows honestly when a backend-changing
    /// update is downloaded but deliberately holding for the 2:30am-local
    /// window rather than pretending nothing is happening.
    /// </summary>
    private void RefreshUpdateDisplay()
    {
        var updateMode = EnvGenerator.EnvFileExists() ? EnvGenerator.ReadAll().GetValueOrDefault("UPDATE_MODE", "automatic") : "automatic";
        _updateToggleButton.Text = updateMode == "off" ? "Turn On" : "Turn Off";
        _updateValue.Text = updateMode == "off"
            ? "Off"
            : UpdateChecker.HasPendingBackendUpdate
                ? "Update ready -- applying overnight (~2:30am)"
                : "Automatic";
    }

    private async Task OnUpdateToggleClickedAsync()
    {
        _updateToggleButton.Enabled = false;
        try
        {
            var turningOff = _updateToggleButton.Text == "Turn Off";
            EnvGenerator.UpdateUpdateMode(turningOff ? "off" : "automatic");
        }
        finally
        {
            await RefreshAsync();
            _updateToggleButton.Enabled = true;
        }
    }

    private static bool ReadDiscordConfigured()
    {
        if (!File.Exists(AppPaths.EnvFile))
            return false;
        foreach (var line in File.ReadLines(AppPaths.EnvFile))
        {
            if (line.StartsWith("DISCORD_BOT_TOKEN=", StringComparison.Ordinal) && line.Length > "DISCORD_BOT_TOKEN=".Length)
                return true;
        }
        return false;
    }
}
