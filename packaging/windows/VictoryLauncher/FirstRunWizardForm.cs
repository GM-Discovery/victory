namespace VictoryLauncher;

/// <summary>
/// Kernel 100 §9 (ask only these two questions), §37 (product voice, not
/// jargon), §38 (truthful step-based progress, never a faked percentage).
///
/// Two full-size panels swapped by visibility, not individual controls
/// toggled/disabled in place: an earlier version disabled the question
/// controls but left them positioned exactly where the progress list
/// also rendered, so the two visibly overlapped once setup started
/// (confirmed on real hardware -- looked exactly as broken as it sounds).
/// Two clean panels can't overlap by construction.
/// </summary>
internal sealed class FirstRunWizardForm : Form
{
    private readonly Panel _questionsPanel = new() { Dock = DockStyle.Fill };
    private readonly Panel _progressPanel = new() { Dock = DockStyle.Fill, Visible = false };

    private readonly TextBox _lotNameBox = new() { Width = 280 };
    private readonly TextBox _operatorHandleBox = new() { Width = 280 };
    // Asked right here, immediately after the two required questions --
    // not something surfaced later in Status for the Operator to
    // stumble onto. Nobody should have to monitor the whole install
    // waiting to be prompted for something; if there's a real decision
    // to make, it belongs up front, before setup itself starts.
    private readonly RadioButton _tunnelOffRadio = new() { Text = "Just this computer, for now", Width = 340, Checked = true };
    private readonly RadioButton _tunnelQuickRadio = new() { Text = "Give people outside this computer a free address automatically", Width = 340 };
    private readonly RadioButton _tunnelNamedRadio = new() { Text = "I already have my own domain and Cloudflare Tunnel token", Width = 380 };
    private readonly Label _tunnelNote = new()
    {
        Text = "That free address changes if this computer restarts -- how stable it stays depends on how reliable this machine and its connection are.",
        AutoSize = false,
        Size = new Size(380, 32),
        ForeColor = SystemColors.GrayText,
        Visible = false,
    };
    private readonly Label _tunnelTokenLabel = new() { Text = "Your Cloudflare Tunnel token", AutoSize = true, Visible = false };
    private readonly TextBox _tunnelTokenBox = new() { Width = 380, UseSystemPasswordChar = true, Visible = false };
    private readonly Label _tunnelHostnameLabel = new() { Text = "What address is it? (e.g. victory.theater)", AutoSize = true, Visible = false };
    private readonly TextBox _tunnelHostnameBox = new() { Width = 380, Visible = false };
    private readonly Button _startButton = new() { Text = "Set up Victory", Width = 140 };
    private readonly Label _errorLabel = new() { ForeColor = Color.Firebrick, AutoSize = true, Visible = false, MaximumSize = new Size(380, 0) };

    private readonly Label _progressStatusLabel = new() { AutoSize = false, Size = new Size(380, 24), Location = new Point(20, 16) };
    private readonly ProgressBar _overallProgressBar = new() { Width = 380, Height = 18, Location = new Point(20, 46), Minimum = 0, Maximum = 1, Style = ProgressBarStyle.Continuous };
    private readonly ListBox _progressList = new() { Width = 380, Height = 166, Location = new Point(20, 72), IntegralHeight = false };

    // A visible clock, ticking every second from the moment setup starts
    // regardless of which step is running or how chatty it is: real
    // per-step progress (the ListBox, the byte counts) proves *what* is
    // happening, but a step that legitimately blocks for a few seconds
    // with no callback of its own (pg_ctl start, createdb) still needs
    // *something* moving on screen, or it reads exactly like a hang --
    // which is the whole thing "so I don't prematurely click" was about.
    private readonly System.Windows.Forms.Timer _elapsedTimer = new() { Interval = 1000 };
    private DateTime _setupStartedAt;
    private string _currentPhaseLabel = "Setting up Victory";
    // Setup finishing used to auto-open the browser and close this
    // window on a timer -- easy to miss entirely (confirmed on real
    // hardware: the Operator never saw it happen, double-clicked the
    // tray icon out of habit instead, and landed on the map with no
    // account). An explicit button the Operator has to click is
    // impossible to miss and ties the browser launch to a real user
    // gesture instead of a background timer race.
    private readonly Button _continueButton = new() { Text = "Continue to Create Your Login →", Width = 300, Location = new Point(20, 246), Visible = false };

    public bool Completed { get; private set; }
    public string? OperatorHandle { get; private set; }

    // Guards against closing the window mid-setup. Without this, closing
    // while OnStartClickedAsync is still awaiting something disposes
    // every control the background continuation later tries to touch
    // (_progressList.Items.Add, ShowError, ...) -- an ObjectDisposedException
    // thrown out of an async-void event handler with nothing to catch it,
    // which crashes the entire process, not just this dialog. Confirmed
    // on real hardware: closing mid-setup took the whole tray app down.
    private bool _setupInProgress;

    public FirstRunWizardForm()
    {
        Text = "Welcome to Victory";
        Icon = AppIcon.Shared;
        FormBorderStyle = FormBorderStyle.FixedDialog;
        MaximizeBox = false;
        MinimizeBox = false;
        StartPosition = FormStartPosition.CenterScreen;
        ClientSize = new Size(420, 440);

        BuildQuestionsPanel();
        BuildProgressPanel();
        _progressStatusLabel.Font = new Font(Font, FontStyle.Bold);

        Controls.Add(_progressPanel);
        Controls.Add(_questionsPanel);
        AcceptButton = _startButton;

        FormClosing += OnFormClosing;
        _elapsedTimer.Tick += (_, _) => UpdateElapsedLabel();
    }

    private void UpdateElapsedLabel()
    {
        var elapsedSeconds = (int)(DateTime.UtcNow - _setupStartedAt).TotalSeconds;
        _progressStatusLabel.Text = $"{_currentPhaseLabel} ({elapsedSeconds}s)";
    }

    private void OnFormClosing(object? sender, FormClosingEventArgs e)
    {
        if (!_setupInProgress)
            return;
        e.Cancel = true;
        MessageBox.Show(
            this,
            "Victory is still setting itself up, so this window can't close yet -- doing so partway through (especially during a download) " +
            "could leave things in a broken, half-finished state that's harder to recover from than just waiting.\n\n" +
            "If a step genuinely seems stuck rather than just slow, it will time out and let you retry within a few minutes -- " +
            "you don't need to force this window shut to get unstuck.",
            "Victory is setting up",
            MessageBoxButtons.OK,
            MessageBoxIcon.Information);
    }

    private void BuildQuestionsPanel()
    {
        var intro = new Label
        {
            Text = "Let's get your Victory lot running.",
            AutoSize = false,
            Size = new Size(380, 24),
            Location = new Point(20, 16),
        };

        var lotNameLabel = new Label { Text = "What should this Victory lot be called?", AutoSize = true, Location = new Point(20, 50) };
        _lotNameBox.Location = new Point(20, 70);

        var operatorHandleLabel = new Label { Text = "What should your Operator handle be?", AutoSize = true, Location = new Point(20, 106) };
        _operatorHandleBox.Location = new Point(20, 126);

        var tunnelLabel = new Label { Text = "Should people outside this computer be able to reach it?", AutoSize = true, Location = new Point(20, 162) };
        _tunnelOffRadio.Location = new Point(20, 182);
        _tunnelQuickRadio.Location = new Point(20, 204);
        _tunnelNamedRadio.Location = new Point(20, 226);

        // Detail area shares one vertical slot -- whichever option is
        // picked shows its own honest tradeoff/fields there, not three
        // permanently-stacked blocks most of which never apply.
        var detailFont = new Font(Font.FontFamily, 7.5f);
        _tunnelNote.Font = detailFont;
        _tunnelNote.Location = new Point(20, 250);
        _tunnelTokenLabel.Location = new Point(20, 250);
        _tunnelTokenBox.Location = new Point(20, 270);
        _tunnelHostnameLabel.Location = new Point(20, 300);
        _tunnelHostnameBox.Location = new Point(20, 320);

        _tunnelOffRadio.CheckedChanged += (_, _) => UpdateTunnelDetailVisibility();
        _tunnelQuickRadio.CheckedChanged += (_, _) => UpdateTunnelDetailVisibility();
        _tunnelNamedRadio.CheckedChanged += (_, _) => UpdateTunnelDetailVisibility();

        _startButton.Location = new Point(20, 362);
        _startButton.Click += async (_, _) => await OnStartClickedAsync();

        _errorLabel.Location = new Point(20, 400);

        _questionsPanel.Controls.AddRange([
            intro, lotNameLabel, _lotNameBox, operatorHandleLabel, _operatorHandleBox,
            tunnelLabel, _tunnelOffRadio, _tunnelQuickRadio, _tunnelNamedRadio,
            _tunnelNote, _tunnelTokenLabel, _tunnelTokenBox, _tunnelHostnameLabel, _tunnelHostnameBox,
            _startButton, _errorLabel,
        ]);
    }

    private void UpdateTunnelDetailVisibility()
    {
        _tunnelNote.Visible = _tunnelQuickRadio.Checked;
        _tunnelTokenLabel.Visible = _tunnelNamedRadio.Checked;
        _tunnelTokenBox.Visible = _tunnelNamedRadio.Checked;
        _tunnelHostnameLabel.Visible = _tunnelNamedRadio.Checked;
        _tunnelHostnameBox.Visible = _tunnelNamedRadio.Checked;
    }

    private void BuildProgressPanel()
    {
        _continueButton.Click += (_, _) => OnContinueClicked();
        _progressPanel.Controls.AddRange([_progressStatusLabel, _overallProgressBar, _progressList, _continueButton]);
    }

    private void OnContinueClicked()
    {
        var url = string.IsNullOrEmpty(OperatorHandle)
            ? $"http://localhost:{RuntimeManager.PublicPort}/signup/"
            : $"http://localhost:{RuntimeManager.PublicPort}/signup/?handle=" + Uri.EscapeDataString(OperatorHandle);
        try
        {
            System.Diagnostics.Process.Start(new System.Diagnostics.ProcessStartInfo(url) { UseShellExecute = true });
        }
        catch
        {
            MessageBox.Show(this, "Could not open your browser. Visit " + url + " manually.", "Victory", MessageBoxButtons.OK, MessageBoxIcon.Warning);
        }
        DialogResult = DialogResult.OK;
        Close();
    }

    private async Task OnStartClickedAsync()
    {
        var lotName = _lotNameBox.Text.Trim();
        var operatorHandle = _operatorHandleBox.Text.Trim();

        _errorLabel.Visible = false;
        if (lotName.Length == 0 || operatorHandle.Length == 0)
        {
            ShowError("Both a lot name and an Operator handle are required.");
            return;
        }
        var tunnelToken = _tunnelTokenBox.Text.Trim();
        var tunnelHostname = _tunnelHostnameBox.Text.Trim();
        if (_tunnelNamedRadio.Checked && (tunnelToken.Length == 0 || tunnelHostname.Length == 0))
        {
            ShowError("Both the Cloudflare Tunnel token and your address are required for this option.");
            return;
        }

        _currentPhaseLabel = "Setting up Victory";
        _progressStatusLabel.Text = _currentPhaseLabel;
        _overallProgressBar.Maximum = 1;
        _overallProgressBar.Value = 0;
        _progressList.Items.Clear();
        _questionsPanel.Visible = false;
        _progressPanel.Visible = true;
        _setupInProgress = true;
        _setupStartedAt = DateTime.UtcNow;
        _elapsedTimer.Start();

        try
        {
            var tunnelMode = _tunnelNamedRadio.Checked ? "named" : _tunnelQuickRadio.Checked ? "quick" : "off";
            await RunStepAsync("Preparing your Victory lot", () =>
            {
                EnvGenerator.Generate(new EnvGenerator.FirstRunAnswers(lotName, operatorHandle, tunnelMode, tunnelToken, tunnelHostname));
                return Task.CompletedTask;
            });

            var setupResult = await RuntimeSetup.EnsureRuntimeReadyAsync(
                reportStep: step =>
                {
                    _progressList.Items.Add(step.Label);
                    _currentPhaseLabel = $"Setting up Victory -- step {step.Current} of {step.Total}";
                    _overallProgressBar.Maximum = step.Total;
                    _overallProgressBar.Value = step.Current;
                },
                updateProgress: detail =>
                {
                    if (_progressList.Items.Count > 0)
                        _progressList.Items[^1] = detail;
                });
            if (setupResult.Result == RuntimeSetup.Result.Failed)
            {
                ShowError(setupResult.Detail ?? "Victory's runtime could not be set up.");
                ResetForRetry();
                return;
            }

            _currentPhaseLabel = "Almost ready -- waiting for Victory to finish starting";
            _overallProgressBar.Value = _overallProgressBar.Maximum;
            await RunStepAsync("Waiting for Victory to be ready", async () =>
            {
                const int maxAttempts = 60;
                for (var attempt = 0; attempt < maxAttempts; attempt++)
                {
                    // The public port (through Caddy), not just the
                    // backend's own /health -- that's what "Open Victory"
                    // actually opens, and what genuinely being ready means
                    // to the Operator waiting on this screen.
                    if (await RuntimeManager.IsBackendHealthyAsync() && await RuntimeManager.IsPublicSiteReachableAsync())
                        return;
                    await Task.Delay(2000);
                }
                throw new TimeoutException("Victory did not report healthy in time.");
            });

            _elapsedTimer.Stop();
            _progressStatusLabel.Text = "Victory is ready -- one more step";
            _progressList.Items.Add("Victory is ready");
            Completed = true;
            OperatorHandle = operatorHandle;
            _setupInProgress = false;
            // Stays open here rather than closing itself: clicking the
            // button (OnContinueClicked) is what actually opens the
            // browser and closes this window, not a timer.
            _continueButton.Visible = true;
        }
        catch (Exception ex)
        {
            ShowError(ex.Message);
            ResetForRetry();
        }
    }

    private async Task RunStepAsync(string label, Func<Task> step)
    {
        _progressList.Items.Add(label + "...");
        await step();
        _progressList.Items[^1] = label;
    }

    private void ShowError(string message)
    {
        _errorLabel.Text = message;
        _errorLabel.Visible = true;
    }

    private void ResetForRetry()
    {
        // Back to the questions panel entirely, not just re-enabling
        // controls in place -- the whole point of two panels is that
        // "which one is showing" is never ambiguous.
        _elapsedTimer.Stop();
        _setupInProgress = false;
        _progressPanel.Visible = false;
        _questionsPanel.Visible = true;
    }
}
