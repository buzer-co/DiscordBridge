# Code signing policy

Status: **eligibility application submitted on 29 October 2026; provider approval pending; no certificate issued and no signing service approved.** Existing v4.6.0 release assets remain unsigned.

## Proposed provider and eligibility disclosure

We are requesting an eligibility review from SignPath Foundation. Attribution to be activated **only after approval and valid signed releases**:

> Free code signing provided by SignPath.io, certificate by SignPath Foundation.

Discord Bridge intentionally uses GoodbyeDPI/WinDivert to work around ISP DPI filtering. It runs with administrator privileges and can change DNS settings as documented in the [privacy and system changes policy](PRIVACY.md). These functions are disclosed to SignPath for assessment under its [conditions](https://signpath.org/terms). Approval is not assumed. The project is new and does not claim an established public reputation or broad user adoption.

## Team roles

| Role | Maintainer |
| --- | --- |
| Authors / committers | [Burhan Uzer — buzer-co](https://github.com/buzer-co) |
| Reviewer of external contributions | [buzer-co](https://github.com/buzer-co) |
| Signing approver | [buzer-co](https://github.com/buzer-co) |

External contributions must be reviewed before merge. Before signing is activated, all participating accounts must use multifactor authentication on GitHub and SignPath. Every production signing request requires a separate manual approval by the designated maintainer in SignPath; a successful CI run alone is insufficient.

## Scope and build provenance

Only the Discord Bridge executable built from this repository by GitHub-hosted runners may be submitted. Local binaries, arbitrary uploads and third-party GoodByeDPI/WinDivert executables, DLLs and drivers are excluded from re-signing under this project's certificate. Third-party components remain unmodified within the application's payload; their provenance and source archives are documented in [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).

The planned artifact configuration restricts ProductName, ProductVersion, FileVersion, CompanyName and OriginalFilename. The version comes from the repository's application version and must agree with the artifact submitted for signing. Signing is disabled until the provider has approved the project and the owner has configured real organization, project, policy and API credentials. No signing credential or repository access is granted by these preparation files.

## Verification and distribution

The manual signing workflow builds and tests on a GitHub-hosted Windows runner, uploads the unsigned application artifact, requests approval-controlled Authenticode signing, and verifies the returned signature and embedded payload. The signed EXE must not be edited or re-packed internally after signing; external ZIP packaging and checksum generation happen afterwards. The workflow produces candidate artifacts only and does not automatically replace existing release assets.

Until approval and signature verification succeed, the public status stays unsigned. Self-signed certificates, repository checksums and CI badges are not presented as trusted Windows publisher signatures. A valid signature authenticates a publisher and file integrity; it does not guarantee compatibility, ISP connectivity or absence of future SmartScreen reputation warnings.

## Privacy

See [Privacy policy and system changes](PRIVACY.md), including network requests, local logs, DNS changes, and removal instructions.
