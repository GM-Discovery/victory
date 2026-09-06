using System.Threading;
using Velopack;

namespace VictoryLauncher;

internal static class Program
{
    private const string SingleInstanceMutexName = "Global\\VictoryLauncher-SingleInstance";

    [STAThread]
    private static int Main()
    {
        // Must be the very first thing that runs (Velopack docs): handles
        // install/update/uninstall hooks (e.g. creating the Start Menu
        // shortcut on first install, per K100 §28) before any of this
        // app's own code executes.
        //
        // OnBeforeUninstallFastCallback: without this, uninstalling never
        // stopped Postgres/backend/Caddy first, and victory-backend.exe
        // specifically lives inside the same folder Velopack's uninstaller
        // has to remove -- a still-running process holding that file open
        // blocks the whole folder from being deleted. Confirmed on real
        // hardware: uninstall got stuck on "file in use" with no obviously
        // Victory-named process visible in Task Manager's simple view.
        // 15-second budget for this hook (Velopack's own limit); StopAsync
        // (Caddy, then the backend, then `pg_ctl stop -m fast -w`) is a
        // local shutdown of small processes and comfortably fits that.
        VelopackApp.Build()
            .OnBeforeUninstallFastCallback(_ => RuntimeManager.StopAsync().GetAwaiter().GetResult())
            .Run();

        using var singleInstanceMutex = new Mutex(initiallyOwned: true, SingleInstanceMutexName, out var isNewInstance);
        if (!isNewInstance)
        {
            // K100 §28: "do not spawn duplicate server stacks" -- a second
            // launch means the Operator wants the running instance, not a
            // second launcher. Best-effort open, then exit immediately.
            try
            {
                System.Diagnostics.Process.Start(new System.Diagnostics.ProcessStartInfo($"http://localhost:{RuntimeManager.PublicPort}/") { UseShellExecute = true });
            }
            catch { /* best effort; the already-running tray icon is still there regardless */ }
            return 0;
        }

        // Not ApplicationConfiguration.Initialize(): that's SDK-generated
        // and depends on template scaffolding this hand-authored project
        // doesn't have. The explicit calls below are what it expands to.
        Application.SetHighDpiMode(HighDpiMode.SystemAware);
        Application.EnableVisualStyles();
        Application.SetCompatibleTextRenderingDefault(false);

        InstallGlobalExceptionHandling();

        var trayContext = new TrayApplicationContext();

        // Application.Run starts FIRST, before InitializeAsync's runtime
        // setup (downloading Postgres on first run, initdb, starting both
        // processes) ever runs -- none of that pumps a message loop by
        // itself outside FirstRunWizardForm's own modal dialog. Running
        // setup before Application.Run left the tray icon visible but
        // completely inert (created, but nothing dispatching its
        // click/menu messages) for that whole stretch on the Podman/WSL2
        // branch this was first built on -- a real bug that reproduced on
        // real hardware, not a hypothetical one. Idle+= runs
        // InitializeAsync once the loop is actually pumping, so the icon
        // stays responsive throughout, and the WindowsFormsSynchronizationContext
        // Application.Run installs is what lets InitializeAsync's own
        // awaits safely marshal back to touch _trayIcon/forms afterward.
        void OnceIdle(object? s, EventArgs e)
        {
            Application.Idle -= OnceIdle;
            _ = RunInitializeAsync(trayContext);
        }
        Application.Idle += OnceIdle;

        Application.Run(trayContext);
        GC.KeepAlive(singleInstanceMutex);
        return 0;
    }

    private static async Task RunInitializeAsync(TrayApplicationContext trayContext)
    {
        var initialized = await trayContext.InitializeAsync();
        if (!initialized)
        {
            // First-run wizard was closed without completing setup --
            // nothing is running, no reason to keep the tray icon up.
            Application.Exit();
        }
    }

    /// <summary>
    /// Before this, any unhandled exception (a disposed-control access,
    /// a bug nobody caught yet) just made the whole tray app vanish with
    /// no explanation -- confirmed on real hardware (closing the
    /// first-run wizard mid-setup did exactly this, separately fixed in
    /// FirstRunWizardForm's own close guard). This is the backstop for
    /// whatever the NEXT uncaught one turns out to be: log it somewhere
    /// findable and tell the Operator plainly, instead of Victory just
    /// disappearing -- an unexplained silent death is exactly the kind
    /// of thing that makes unsigned, unfamiliar software feel untrustworthy.
    /// </summary>
    private static void InstallGlobalExceptionHandling()
    {
        Application.SetUnhandledExceptionMode(UnhandledExceptionMode.CatchException);
        Application.ThreadException += (_, e) => ReportUnhandledException(e.Exception, canContinue: true);
        AppDomain.CurrentDomain.UnhandledException += (_, e) =>
            ReportUnhandledException(e.ExceptionObject as Exception, canContinue: false);
    }

    private static void ReportUnhandledException(Exception? ex, bool canContinue)
    {
        try
        {
            Directory.CreateDirectory(AppPaths.LogsDir);
            File.AppendAllText(
                Path.Combine(AppPaths.LogsDir, "launcher.log"),
                $"{DateTime.UtcNow:O} unhandled exception (continuing={canContinue}): {ex}\n");
        }
        catch { /* the message box below is the fallback if even logging fails */ }

        MessageBox.Show(
            "Victory ran into a problem it wasn't expecting" + (canContinue ? "." : " and has to close.") +
            "\n\nDetails were saved to:\n" + Path.Combine(AppPaths.LogsDir, "launcher.log") +
            "\n\n" + ex?.Message,
            "Victory",
            MessageBoxButtons.OK,
            canContinue ? MessageBoxIcon.Warning : MessageBoxIcon.Error);
    }
}
