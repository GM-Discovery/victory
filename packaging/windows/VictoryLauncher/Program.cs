using System.Threading;
using Velopack;

namespace VictoryLauncher;

internal static class Program
{
    private const string SingleInstanceMutexName = "Global\\VictoryLauncher-SingleInstance";

    [STAThread]
    private static void Main()
    {
        // Must be the very first thing that runs (Velopack docs): handles
        // install/update/uninstall hooks (e.g. creating the Start Menu
        // shortcut on first install, per K100 §28) before any of this
        // app's own code executes.
        VelopackApp.Build().Run();

        using var singleInstanceMutex = new Mutex(initiallyOwned: true, SingleInstanceMutexName, out var isNewInstance);
        if (!isNewInstance)
        {
            // K100 §28: "do not spawn duplicate server stacks" -- a second
            // launch means the Operator wants the running instance, not a
            // second launcher. Best-effort open, then exit immediately.
            try
            {
                System.Diagnostics.Process.Start(new System.Diagnostics.ProcessStartInfo("http://localhost:8081/") { UseShellExecute = true });
            }
            catch { /* best effort; the already-running tray icon is still there regardless */ }
            return;
        }

        // Not ApplicationConfiguration.Initialize(): that's SDK-generated
        // and depends on template scaffolding this hand-authored project
        // doesn't have. The explicit calls below are what it expands to.
        Application.SetHighDpiMode(HighDpiMode.SystemAware);
        Application.EnableVisualStyles();
        Application.SetCompatibleTextRenderingDefault(false);

        var trayContext = new TrayApplicationContext();

        // Blocking on this Task before Application.Run is deliberate, not
        // an oversight: FirstRunWizardForm.ShowDialog() pumps its own
        // nested message loop regardless of whether the main Application
        // message loop has started, so this is the standard way to do
        // "async setup, possibly a modal dialog, then enter the tray
        // loop" without a WindowsFormsSynchronizationContext existing yet.
        var initialized = trayContext.InitializeAsync().GetAwaiter().GetResult();
        if (!initialized)
        {
            // First-run wizard was closed without completing setup --
            // nothing is running, nothing to keep the process alive for.
            return;
        }

        Application.Run(trayContext);
        GC.KeepAlive(singleInstanceMutex);
    }
}
