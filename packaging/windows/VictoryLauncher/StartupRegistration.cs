using Microsoft.Win32;

namespace VictoryLauncher;

/// <summary>
/// K100 §27's "start with Windows if configured" toggle. Shared between
/// TrayApplicationContext's user-facing menu item and RuntimeSetup's
/// automatic "turn this on before a required reboot" behavior (K100
/// §39: resume automatically after login/reboot where practical --
/// resuming needs the app to actually relaunch after that reboot).
/// </summary>
internal static class StartupRegistration
{
    private const string RunRegistryKey = @"Software\Microsoft\Windows\CurrentVersion\Run";
    private const string RunValueName = "VictoryLauncher";

    public static bool IsEnabled()
    {
        using var key = Registry.CurrentUser.OpenSubKey(RunRegistryKey, writable: false);
        return key?.GetValue(RunValueName) is not null;
    }

    public static void SetEnabled(bool enabled)
    {
        using var key = Registry.CurrentUser.OpenSubKey(RunRegistryKey, writable: true)
            ?? Registry.CurrentUser.CreateSubKey(RunRegistryKey);
        if (enabled)
            key.SetValue(RunValueName, "\"" + Environment.ProcessPath + "\"");
        else
            key.DeleteValue(RunValueName, throwOnMissingValue: false);
    }
}
