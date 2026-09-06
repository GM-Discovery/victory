namespace VictoryLauncher;

/// <summary>
/// Kernel 100 §26: human-readable status, never raw container internals
/// as the primary view. Remote access and update status genuinely don't
/// exist as features yet (Cloudflare Tunnel and the Velopack update
/// check are later K100 phases) -- shown honestly as "not set up yet"
/// rather than faked, matching §38's "do not fake" doctrine applied to
/// status as well as progress.
/// </summary>
internal sealed class StatusForm : Form
{
    private readonly Label _runningValue = new() { AutoSize = true };
    private readonly Label _databaseValue = new() { AutoSize = true };
    private readonly Label _remoteValue = new() { AutoSize = true, Text = "Not set up yet" };
    private readonly Label _urlValue = new() { AutoSize = true, Text = $"http://localhost:{RuntimeManager.PublicPort} (this device only)" };
    private readonly Label _updateValue = new() { AutoSize = true, Text = "Not set up yet" };
    private readonly Label _discordValue = new() { AutoSize = true };
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
        ClientSize = new Size(420, 260);

        AddRow("Victory:", _runningValue, 20);
        AddRow("Database:", _databaseValue, 50);
        AddRow("Remote access:", _remoteValue, 80);
        AddRow("Address:", _urlValue, 110);
        AddRow("Updates:", _updateValue, 140);
        AddRow("Discord:", _discordValue, 170);

        _startStopButton.Location = new Point(20, 210);
        _startStopButton.Click += async (_, _) => await OnStartStopClickedAsync();

        _refreshButton.Location = new Point(150, 210);
        _refreshButton.Click += async (_, _) => await RefreshAsync();

        Controls.AddRange([_startStopButton, _refreshButton]);
        Shown += async (_, _) => await RefreshAsync();
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
