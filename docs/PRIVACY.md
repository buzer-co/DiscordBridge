# Privacy policy and system changes

Maintainer: [Burhan Uzer / buzer-co](https://github.com/buzer-co). This policy covers the native Discord Bridge Windows application, not GitHub or SignPath websites.

## Data and network requests

Discord Bridge contains no intentional telemetry, advertising, account collection or automatic upload of local logs. Its network graph reads physical-adapter byte counters locally and does not identify the applications or content producing that traffic. Startup does not automatically launch Discord or initiate a connection test.

When the user requests a connection or a connectivity test, the app resolves Discord domains through the configured system DNS resolver and makes HTTPS requests to `updates.discord.com` and `discord.com`. Choosing Open Discord launches the installed Discord application or opens `https://discord.com/app`; Discord then operates under its own [privacy policy](https://discord.com/privacy). If the user chooses a DNS operation, or starts the connection workflow and the existing resolver returns an invalid/blocked result, the app may apply Cloudflare DNS after backing up the previous configuration. Cloudflare's handling of public-resolver requests is described in its [1.1.1.1 privacy documentation](https://developers.cloudflare.com/1.1.1.1/privacy/public-dns-resolver/).

The application does not send browsing history, packet contents, local files or logs to the maintainer or SignPath. GoodbyeDPI/WinDivert intercept and modify matching network traffic locally; they do not create a VPN tunnel or a maintainer-operated proxy.

## Administrator privileges and changes

Starting the app requests Windows administrator permission. A user-initiated connection workflow may extract the bundled engine, run GoodbyeDPI with WinDivert, modify DNS settings where a blocked resolver is detected, and clear the DNS cache. It does not change ExitLag configuration. The tray menu can stop the engine launched by this application. Closing the window with X only hides it; the connection process may continue until stopped.

Settings, DNS backup and local diagnostic logs are stored under `%LOCALAPPDATA%\DiscordBridge`. Logs may contain local paths, process information, adapter names and connection diagnostics. They are not automatically uploaded. Inspect them before sharing them with anyone.

## Restore and removal

1. Use **DNS geri al / Restore DNS** to restore a saved DNS configuration, if one was changed.
2. Use the tray menu **DPI'ı Durdur ve Çık / Stop DPI and Exit** to stop the application's engine and exit.
3. Delete the portable EXE and its extracted distribution folder. Delete `%LOCALAPPDATA%\DiscordBridge` if settings and logs are no longer needed; keep the DNS backup until restoration is complete.

Do not delete a driver or service used by another WinDivert application. If Windows still reports the driver in use after all consumers have exited, restart before removing the extracted application folder. Removing files alone does not restore changed DNS settings, so restore DNS first.

## External sites and signing

GitHub hosts the source and releases. Applying for signing requires voluntarily sending the maintainer's name, email and public project details to SignPath, under SignPath's own privacy policy and conditions. The signing preparation files do not grant SignPath access to private repositories, credentials or application users' data.
