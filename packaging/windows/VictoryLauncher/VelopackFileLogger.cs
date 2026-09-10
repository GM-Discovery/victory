using Microsoft.Extensions.Logging;

namespace VictoryLauncher;

/// <summary>
/// Routes Velopack's own internal diagnostic trace into launcher.log.
/// Added specifically because ApplyUpdatesAndRestart has failed silently on
/// real hardware multiple times -- no exception ever reached this app's own
/// try/catch, no AV quarantine, no SmartScreen prompt, and the app just
/// never came back. Every one of those checks ruled out something in
/// Victory's own code or in Windows' own gatekeeping; what's left is
/// Velopack's own update/restart mechanism, which this app had zero
/// visibility into. UpdateManager accepts an optional ILogger -- this
/// plugs into it, whether or not Velopack has anything useful to say about
/// the specific failure. Every UpdateManager construction site should pass
/// one of these.
/// </summary>
internal sealed class VelopackFileLogger : ILogger
{
    public IDisposable? BeginScope<TState>(TState state) where TState : notnull => null;

    public bool IsEnabled(LogLevel logLevel) => true;

    public void Log<TState>(LogLevel logLevel, EventId eventId, TState state, Exception? exception, Func<TState, Exception?, string> formatter)
    {
        try
        {
            Directory.CreateDirectory(AppPaths.LogsDir);
            var line = $"{DateTime.UtcNow:O} [velopack:{logLevel}] {formatter(state, exception)}";
            if (exception is not null)
                line += Environment.NewLine + exception;
            // AppendAllText, not a buffered writer: this needs to survive a
            // process that might die moments later without ever flushing
            // anything else cleanly.
            File.AppendAllText(Path.Combine(AppPaths.LogsDir, "launcher.log"), line + Environment.NewLine);
        }
        catch
        {
            // Best effort -- a missed diagnostic line isn't worth failing
            // the actual update-check/apply flow over.
        }
    }
}
