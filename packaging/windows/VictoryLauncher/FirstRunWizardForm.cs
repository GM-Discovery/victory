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
    private readonly ListBox _progressList = new() { Width = 380, Height = 220, Location = new Point(20, 46), IntegralHeight = false };

    public bool Completed { get; private set; }

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
    }

    private void OnFormClosing(object? sender, FormClosingEventArgs e)
    {
        if (!_setupInProgress)
            return;
        e.Cancel = true;
        MessageBox.Show(
            this,
            "Victory is still setting itself up. Closing this window now could leave things half-configured -- please wait for it to finish.",
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
        _progressPanel.Controls.AddRange([_progressStatusLabel, _progressList]);
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

        _progressStatusLabel.Text = "Setting up Victory...";
        _progressList.Items.Clear();
        _questionsPanel.Visible = false;
        _progressPanel.Visible = true;
        _setupInProgress = true;

        try
        {
            await RunStepAsync("Preparing your Victory lot", () =>
            {
                EnvGenerator.Generate(new EnvGenerator.FirstRunAnswers(lotName, operatorHandle));
                return Task.CompletedTask;
            });

            var setupResult = await RuntimeSetup.EnsureRuntimeReadyAsync(step => _progressList.Items.Add(step));
            if (setupResult.Result == RuntimeSetup.Result.Failed)
            {
                ShowError(setupResult.Detail ?? "Victory's runtime could not be set up.");
                ResetForRetry();
                return;
            }

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

            _progressStatusLabel.Text = "Victory is ready";
            _progressList.Items.Add("Victory is ready");
            Completed = true;
            _setupInProgress = false;
            await Task.Delay(700); // let the Operator actually see "ready" before the window closes
            DialogResult = DialogResult.OK;
            Close();
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
        _setupInProgress = false;
        _progressPanel.Visible = false;
        _questionsPanel.Visible = true;
    }
}
