# Discord Bridge

**Status: early beta — unsigned and not yet approved for SignPath Foundation.**

Discord Bridge is an independent Windows application that manages GoodbyeDPI and provides Discord connectivity diagnostics. It is not an official Discord product.

## Planned functionality

- Embedded GoodbyeDPI / WinDivert components (third-party licenses and provenance must be verified before release)
- A dedicated **Superonline + ExitLag** profile, preserving `-5 --native-frag --blacklist discord-domains.txt`
- Alternative profiles for other network conditions, with no guarantee of compatibility
- Optional DNS management with backup and restore
- Manual Discord launch (never automatic), tray icon, and connectivity diagnostics

## Current release status

No executable has been published or signed through this GitHub repository. Windows SmartScreen may warn about unsigned executables. Do not disable Windows security checks. The source and GitHub Actions workflow are being prepared for review and reproducible builds before any public binary release.

## Acknowledgements

- [GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI) (Apache-2.0)
- [WinDivert](https://github.com/basil00/WinDivert) (LGPL-3.0)

The project is unaffiliated with Discord, GoodbyeDPI, or WinDivert authors.
