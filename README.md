# Discord Bridge

A small native Windows control center for GoodbyeDPI, with a soft Discord-inspired interface, saved connection profiles, DNS backup/restore and a live network graph.

**[Download v4.6.0](https://github.com/buzer-co/DiscordBridge/releases/tag/v4.6.0)** · [Türkçe](README.tr.md) · [Build and test](https://github.com/buzer-co/DiscordBridge/actions)

Windows 10/11, x64. One executable; Go and Python are needed only for building from source. This is an independent project, unaffiliated with Discord or the GoodbyeDPI/WinDivert authors.

## Use

1. Exit an older Discord Bridge instance through its system tray menu.
2. Download the Windows ZIP from Releases, extract it, and run `DiscordBridge_4_6_Stable.exe`.
3. Select **Automatic** or **Superonline + ExitLag**, then click **Start connection**.
4. Click **Open Discord** when you want to launch Discord.

Closing the window hides it to the system tray. Use the tray menu to exit or stop the engine. Settings and logs are stored under `%LOCALAPPDATA%\DiscordBridge`.

## What's new in 4.6

- Traffic samples repaint only the small network panel. They do not refresh the entire window or main button.
- Buffered window and button painting, correct rounded-corner backgrounds, and bounded 60-sample history.
- Native 64-bit Windows traffic counters, without launching PowerShell for every sample.
- The window scales to fit compact desktops without clipping controls.
- Soft rounded controls, compact profile choices, manual Discord launch and persisted preferences.
- The dedicated Superonline + ExitLag profile preserves `-5 --native-frag --blacklist discord-domains.txt`. ExitLag settings are left untouched.

The graph measures **all physical-adapter traffic**, not Discord-only traffic. DNS changes are backed up before applying Cloudflare; restore is available in the app.

## Release status

The initial public release is marked **pre-release** while ISP-specific end-to-end testing continues. Test results and the native Windows UI capture are available in the Actions run associated with the release commit. A successful runner test does not establish that a profile works on every ISP, ExitLag route or Windows installation.

The executable is **unsigned**. SignPath approval or Authenticode signing is not claimed. The filename `Stable` distinguishes this repaint-fix build from earlier local iterations; the GitHub pre-release flag is the authoritative public release status.

## Build

With Go 1.23+ and Python 3:

```sh
python build.py dist/DiscordBridge_4_6_Stable.exe
python scripts/package_release.py
```

The release workflow pins Go 1.27.3 and GitHub Actions to commit SHAs. It builds directly from the tagged source, runs the state tests on Linux and Windows, performs native Windows repaint/resource/control checks, and publishes the EXE, Windows ZIP, full source package and SHA-256 checksums after those checks pass.

```sh
go test -buildvcs=false .
# On Linux with a C compiler:
go test -buildvcs=false -race .
```

The embedded engine is unchanged from the official GoodbyeDPI 0.2.3rc3 release package. Its three binary hashes are verified by `scripts/verify_engine.py`. Corresponding third-party source archives, original license texts and provenance are included in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

## License

Discord Bridge code: [MIT](LICENSE). GoodbyeDPI: Apache-2.0. WinDivert: LGPL-3.0. Third-party licenses do not change under this repository's MIT license.

## Code signing policy

Signing preparation is documented in [Code signing policy](docs/CODE_SIGNING_POLICY.md) and [Privacy policy](docs/PRIVACY.md). The SignPath Foundation eligibility application was submitted on 29 October 2026; provider approval is pending. Existing release files remain unsigned.
