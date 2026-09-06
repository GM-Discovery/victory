namespace VictoryLauncher;

/// <summary>
/// Kernel 100 §9 (ask only these two questions), §37 (product voice, not
/// jargon), §38 (truthful step-based progress, never a faked percentage).
/// </summary>
internal sealed class FirstRunWizardForm : Form
{
    private readonly TextBox _lotNameBox = new() { Width = 280 };
    private readonly TextBox _operatorHandleBox = new() { Width = 280 };
    private readonly Button _startButton = new() { Text = "Set up Victory", Width = 140 };
    private readonly ListBox _progressList = new() { Width = 380, Height = 140, Visible = false, IntegralHeight = false };
    private readonly Label _errorLabel = new() { ForeColor = Color.Firebrick, AutoSize = true, Visible = false, MaximumSize = new Size(380, 0) };

    public bool Completed { get; private set; }

    public FirstRunWizardForm()
    {
        Text = "Welcome to Victory";
        FormBorderStyle = FormBorderStyle.FixedDialog;
        MaximizeBox = false;
        MinimizeBox = false;
        StartPosition = FormStartPosition.CenterScreen;
        ClientSize = new Size(420, 320);

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

        _progressList.Location = new Point(20, 130);
        _errorLabel.Location = new Point(20, 180);

        Controls.AddRange([intro, lotNameLabel, _lotNameBox, operatorHandleLabel, _operatorHandleBox, _startButton, _progressList, _errorLabel]);
        AcceptButton = _startButton;
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

        _lotNameBox.Enabled = false;
        _operatorHandleBox.Enabled = false;
        _startButton.Enabled = false;
        _progressList.Visible = true;
        _progressList.Items.Clear();

        try
        {
            await RunStepAsync("Preparing your Victory lot", () =>
            {
                EnvGenerator.Generate(new EnvGenerator.FirstRunAnswers(lotName, operatorHandle));
                return Task.CompletedTask;
            });

            var setupResult = await RuntimeSetup.EnsureRuntimeReadyAsync(step => _progressList.Items.Add(step));
            switch (setupResult.Result)
            {
                case RuntimeSetup.Result.Ready:
                    break;
                case RuntimeSetup.Result.RebootRequired:
                    OfferRestart(setupResult.Detail!);
                    return;
                case RuntimeSetup.Result.ElevationDeclined:
                case RuntimeSetup.Result.Failed:
                default:
                    ShowError(setupResult.Detail ?? "Victory's runtime could not be set up.");
                    ResetForRetry();
                    return;
            }

            await RunStepAsync("Starting Victory", async () =>
            {
                var (success, output) = await RuntimeManager.StartAsync();
                if (!success)
                    throw new InvalidOperationException("Victory did not start:\n" + output);
            });

            await RunStepAsync("Waiting for Victory to be ready", async () =>
            {
                const int maxAttempts = 60;
                for (var attempt = 0; attempt < maxAttempts; attempt++)
                {
                    if (await RuntimeManager.IsBackendHealthyAsync())
                        return;
                    await Task.Delay(2000);
                }
                throw new TimeoutException("Victory did not report healthy in time.");
            });

            _progressList.Items.Add("Victory is ready");
            Completed = true;
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

    private async Task<T> RunStepAsync<T>(string label, Func<Task<T>> step)
    {
        _progressList.Items.Add(label + "...");
        var result = await step();
        _progressList.Items[^1] = label;
        return result;
    }

    private void OfferRestart(string reason)
    {
        // So the SAME setup resumes automatically after the Operator
        // signs back in (K100 §39) -- they haven't seen the "Start with
        // Windows" menu item yet, this wizard hasn't shown a tray icon at all.
        StartupRegistration.SetEnabled(true);
        using var restartForm = new RestartRequiredForm(reason);
        restartForm.ShowDialog();
        // Completed stays false: nothing is running yet, so the caller
        // (TrayApplicationContext.InitializeAsync) correctly treats this
        // exactly like closing the wizard without finishing, and exits
        // quietly rather than opening a browser to a Victory that isn't up.
        Close();
    }

    private void ShowError(string message)
    {
        _errorLabel.Text = message;
        _errorLabel.Visible = true;
    }

    private void ResetForRetry()
    {
        _lotNameBox.Enabled = true;
        _operatorHandleBox.Enabled = true;
        _startButton.Enabled = true;
    }
}
