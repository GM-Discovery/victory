using System.Threading;
using Velopack;

namespace VictoryLauncher;

internal static class Program
{
    private const string SingleInstanceMutexName = "Global\\VictoryLauncher-SingleInstance";

    [STAThread]
    private static int Main(string[] args)
    {
        // Checked before Velopack's own bootstrap and before anything
        // else: this is PrivilegedSetup's elevated child process (see its
        // header comment), not a normal launch. It runs one narrow
        // operation and exits -- no tray icon, no update-apply lifecycle,
        // no first-run wizard.
        if (args.Length == 2 && args[0] == "--elevated-step" && Enum.TryParse<PrivilegedSetup.ElevatedStep>(args[1], out var step))
        {
            return step switch
            {
                PrivilegedSetup.ElevatedStep.EnableWsl2 => ElevatedSteps.EnableWsl2Async().GetAwaiter().GetResult(),
                PrivilegedSetup.ElevatedStep.InstallPodman => ElevatedSteps.InstallPodmanAsync().GetAwaiter().GetResult(),
                _ => -1,
            };
        }

        // Must be the very first thing that runs in the normal app
        // lifecycle (Velopack docs): handles install/update/uninstall
        // hooks (e.g. creating the Start Menu shortcut on first install,
        // per K100 §28) before any of this app's own code executes.
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
            return 0;
        }

        // Not ApplicationConfiguration.Initialize(): that's SDK-generated
        // and depends on template scaffolding this hand-authored project
        // doesn't have. The explicit calls below are what it expands to.
        Application.SetHighDpiMode(HighDpiMode.SystemAware);
        Application.EnableVisualStyles();
        Application.SetCompatibleTextRenderingDefault(false);

        var trayContext = new TrayApplicationContext();

        // Application.Run starts FIRST, before InitializeAsync's runtime
        // setup (WSL2 checks, elevation, waiting on winget) ever runs --
        // that setup can take minutes and involves waiting on a UAC
        // prompt, and none of it pumps a message loop by itself outside
        // FirstRunWizardForm's own modal dialog. Running it before
        // Application.Run left the tray icon visible but completely
        // inert (created, but nothing dispatching its click/menu
        // messages) for that whole stretch -- a real bug this fixes, not
        // a hypothetical one: it reproduced on real hardware. Idle+= runs
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
            // First-run wizard was closed without completing setup, or a
            // reboot is now required -- nothing is running, no reason to
            // keep the tray icon up.
            Application.Exit();
        }
    }
}
