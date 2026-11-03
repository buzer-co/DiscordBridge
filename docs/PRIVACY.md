# Privacy policy and system changes

Maintainer: [Burhan Uzer / buzer-co](https://github.com/buzer-co). This policy covers the native Discord Bridge Windows application, not GitHub or SignPath websites.

## Data and network requests

Discord Bridge contains no intentional telemetry, advertising, account collection or automatic upload of local logs. Its network graph reads physical-adapter byte counters locally and does not identify the applications or content producing that traffic. Startup does not automatically launch Discord or initiate a connection test.

When the user starts a connection, the app opens a local WinDivert filter and sends standard UDP DNS questions only for `discord.com`, `discord.gg`, `discordapp.com`, `discordapp.net`, `discord.media` and their subdomains to Cloudflare DNS over HTTPS, falling back to Google Public DNS when needed. Those providers receive the selected DNS question, the public source IP address and ordinary HTTPS metadata; no query log is kept by Discord Bridge. All other domain questions pass through to their existing resolver. The providers' policies are described in [Cloudflare's resolver privacy documentation](https://developers.cloudflare.com/1.1.1.1/privacy/public-dns-resolver/) and [Google Public DNS privacy documentation](https://developers.google.com/speed/public-dns/privacy).

Connection tests make HTTPS requests to `updates.discord.com` and `discord.com`. Choosing Open Discord launches the installed Discord application or opens `https://discord.com/app`; Discord then operates under its own [privacy policy](https://discord.com/privacy). Start never applies adapter DNS automatically. The separate optional DNS button can still apply Cloudflare adapter DNS after its explicit confirmation and backup. The DNS bridge does not change adapter addresses, hosts files, NRPT rules, system proxy settings or IP routing.

The application does not send browsing history, packet contents, local files or logs to the maintainer or SignPath. Selected DNS questions are sent only to the DNS providers described above. GoodbyeDPI/WinDivert intercept and modify matching network traffic locally; they do not create a VPN tunnel or a maintainer-operated proxy.

## Administrator privileges and changes

Starting the app requests Windows administrator permission. A user-initiated connection workflow may extract the bundled engine, run GoodbyeDPI with WinDivert, start the selective Discord DNS filter, and clear the DNS cache. It does not change ExitLag configuration. The tray menu can stop the engine launched by this application. Closing the window with X only hides it; the connection process may continue until stopped.

The only saved user preference is the last successful connection profile, stored in `%LOCALAPPDATA%\DiscordBridge\settings.json`. Language and the engine path are not saved. No session/engine log files or persistent process-ID file are written. The old app-owned `session.log` and `goodbyedpi.log` files are removed at startup. Tool details are held temporarily while the page is open, cleared on leaving it, and never uploaded. Only the latest status message remains in memory until exit.

The bundled engine and domain list are local program resources. A DNS restore backup is retained only for an explicit manual DNS change (or when inherited from an older version) so settings can be restored safely; successful restoration removes it. A Windows job object tracks the app's child engine in memory and ends that process when the owning app closes; it does not manage ExitLag or unrelated programs.

## Restore and removal

1. Use **DNS geri al / Restore DNS** to restore a saved DNS configuration, if one was changed.
2. Use the tray menu **DPI'ı Durdur ve Çık / Stop DPI and Exit** to stop the application's engine and exit.
3. Delete the portable EXE and its extracted distribution folder. Delete `%LOCALAPPDATA%\DiscordBridge` if profile settings and program resources are no longer needed; keep the DNS backup until restoration is complete.

Do not delete a driver or service used by another WinDivert application. If Windows still reports the driver in use after all consumers have exited, restart before removing the extracted application folder. Removing files alone does not restore changed DNS settings, so restore DNS first.

## External sites and signing

GitHub hosts the source and releases. Applying for signing requires voluntarily sending the maintainer's name, email and public project details to SignPath, under SignPath's own privacy policy and conditions. The signing preparation files do not grant SignPath access to private repositories, credentials or application users' data.
