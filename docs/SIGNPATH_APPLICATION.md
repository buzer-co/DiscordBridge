# SignPath Foundation eligibility application

Status: **submitted on 29 October 2026; eligibility review pending**. Submitted through the official application form with the maintainer's explicit authorization of the mandatory Code of Conduct and personal-data processing consents. The form confirmed "Form submitted" and "Thank you, we'll be in touch soon." This is a submission receipt, not provider approval or an issued certificate.

- Project: Discord Bridge
- Repository/homepage: https://github.com/buzer-co/DiscordBridge
- Download: https://github.com/buzer-co/DiscordBridge/releases/tag/v4.6.0
- License: MIT application; Apache-2.0 GoodbyeDPI and LGPL-3.0 WinDivert
- Policy: https://github.com/buzer-co/DiscordBridge/blob/main/docs/CODE_SIGNING_POLICY.md
- Privacy: https://github.com/buzer-co/DiscordBridge/blob/main/docs/PRIVACY.md
- Category: Other / networking utility
- Maintainer type: Individual maintainer
- Build system: GitHub Actions
- Maintainer/approver: [Burhan Uzer, buzer-co](https://github.com/buzer-co)

## Tagline

An open-source Windows desktop app for managing Discord connectivity with GoodbyeDPI.

## Description and eligibility disclosure

Discord Bridge is a free, MIT-licensed native Windows application that gives users explicit controls for Discord connectivity, saved GoodbyeDPI profiles, DNS backup and restoration, and manual Discord launch. It intentionally uses GoodbyeDPI/WinDivert to work around ISP DPI filtering and requires administrator privileges. The bundled upstream binaries are unchanged, have documented hashes and licenses, and are not proposed for re-signing as our own code. Please assess this functionality against your eligibility rules, including the restriction on security-circumvention tools. It is an independent project with no affiliation to Discord.

## Reputation

This is a new, early-stage project, first published on 28 October 2026. We do not yet claim an established user base, independent media coverage, or a widely verified reputation. Public source and MIT license are in the repository. The initial public release and public Linux/native Windows UI test/build results are available through Releases and Actions. We request an eligibility review and understand that project age/reputation and the DPI-circumvention functionality may make the project ineligible.

## Build and signing scope

The wrapper is built on public GitHub-hosted runners with version metadata. Third-party hashes and corresponding source archives are published. Only the wrapper EXE is proposed for approval-controlled signing; upstream executables, libraries and drivers retain their original signatures/licenses. No signing subscription, certificate, approved project, token or GitHub access grant is assumed by this application draft.
