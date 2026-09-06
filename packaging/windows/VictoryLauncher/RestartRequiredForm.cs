using System.Diagnostics;

namespace VictoryLauncher;

/// <summary>
/// K100 §39: tell the user why a restart is needed, offer it, never force
/// it out from under them. Shared by the first-run wizard and the
/// tray's resume-after-reboot path -- the same message either way.
/// </summary>
internal sealed class RestartRequiredForm : Form
{
    public RestartRequiredForm(string reason)
    {
        Text = "Victory needs to restart Windows";
        FormBorderStyle = FormBorderStyle.FixedDialog;
        MaximizeBox = false;
        MinimizeBox = false;
        StartPosition = FormStartPosition.CenterScreen;
        ClientSize = new Size(420, 180);

        var message = new Label
        {
            Text = reason + "\n\nVictory will pick up right where it left off after you sign back in.",
            AutoSize = false,
            Size = new Size(380, 90),
            Location = new Point(20, 16),
        };

        var restartNowButton = new Button { Text = "Restart now", Width = 130, Location = new Point(20, 120) };
        restartNowButton.Click += (_, _) =>
        {
            try
            {
                Process.Start(new ProcessStartInfo("shutdown.exe", "/r /t 10 /c \"Victory needs to finish setting up.\"") { UseShellExecute = false, CreateNoWindow = true });
            }
            catch { /* if this fails, "Restart later" is still available -- Windows' own restart UI works regardless */ }
            Close();
        };

        var restartLaterButton = new Button { Text = "Restart later", Width = 130, Location = new Point(160, 120) };
        restartLaterButton.Click += (_, _) => Close();

        Controls.AddRange([message, restartNowButton, restartLaterButton]);
        AcceptButton = restartNowButton;
    }
}
