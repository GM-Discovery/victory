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
        ClientSize = new Size(420, 320);

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
            Text = "Let's get your Victory lot running. Two questions, then Victory sets itself up.",
            AutoSize = false,
            Size = new Size(380, 40),
            Location = new Point(20, 16),
        };

        var lotNameLabel = new Label { Text = "What should this Victory lot be called?", AutoSize = true, Location = new Point(20, 66) };
        _lotNameBox.Location = new Point(20, 86);

        var operatorHandleLabel = new Label { Text = "What should your Operator handle be?", AutoSize = true, Location = new Point(20, 122) };
        _operatorHandleBox.Location = new Point(20, 142);

        _startButton.Location = new Point(20, 180);
        _startButton.Click += async (_, _) => await OnStartClickedAsync();

        _errorLabel.Location = new Point(20, 220);

        _questionsPanel.Controls.AddRange([intro, lotNameLabel, _lotNameBox, operatorHandleLabel, _operatorHandleBox, _startButton, _errorLabel]);
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
            await RunStepAsync("Preparing your Victory lot", () =>
            {
                EnvGenerator.Generate(new EnvGenerator.FirstRunAnswers(lotName, operatorHandle));
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
