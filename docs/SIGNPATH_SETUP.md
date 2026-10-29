# SignPath setup after eligibility approval

Current state: application preparation only. The [public application](https://signpath.org/apply.html) requires the maintainer's own acceptance of its Code of Conduct and personal-data processing. No approval, certificate, API credential or GitHub App installation is claimed.

## Approval first

Disclose the DPI-circumvention purpose, administrator privileges, DNS changes, unmodified third-party payload and the project's early stage. If SignPath finds the project ineligible, do not hide those functions or change the classification to evade the decision. Do not enable the signing workflow without provider approval.

## Provider configuration

After acceptance, create the provider-assigned organization/project, connect the specific public repository as a trusted GitHub build system and configure a **manual approval** production signing policy. Use `signing/artifact-configuration.xml` to sign only the project's executable. Do not select nested upstream binaries for signing. Confirm MFA on the maintainer's GitHub and SignPath accounts.

Any SignPath GitHub App installation or access grant must be separately reviewed by the owner and limited to the required repository. Private keys remain with the signing service; they must not be committed or exported to this repository.

## GitHub configuration

Add the real provider values, without placeholder credentials:

| Type | Name | Value |
| --- | --- | --- |
| Secret | `SIGNPATH_API_TOKEN` | A provider token restricted to submitting requests for this project/policy |
| Variable | `SIGNPATH_ORGANIZATION_ID` | Assigned organization ID |
| Variable | `SIGNPATH_PROJECT_SLUG` | Approved project slug |
| Variable | `SIGNPATH_SIGNING_POLICY_SLUG` | Production policy requiring the maintainer's manual approval |
| Variable | `SIGNPATH_ARTIFACT_CONFIGURATION_SLUG` | Uploaded configuration slug |
| Variable | `SIGNPATH_ENABLED` | Set to `true` only after all approvals and checks above |

Run the **Prepare and sign Windows candidate** workflow manually on `main`. It builds a new artifact in the same trusted run, submits it for SignPath approval, checks Authenticode status and publisher, checks payload integrity, and creates a signed Windows ZIP and checksum candidates. It does not publish a release automatically. Review the returned request URL and verify the output before deciding to publish.

```powershell
Get-AuthenticodeSignature .\DiscordBridge_4_6_Stable.exe | Format-List Status,StatusMessage,SignerCertificate,TimeStamperCertificate
```

Never modify an executable after signing. ZIP packaging and SHA-256 generation must use the final signed bytes. Existing v4.6.0 assets stay unchanged until an explicitly reviewed signed release is published.

Official references: [Foundation conditions](https://signpath.org/terms), [GitHub trusted builds](https://docs.signpath.io/trusted-build-systems/github), [artifact configuration](https://docs.signpath.io/artifact-configuration/reference).
