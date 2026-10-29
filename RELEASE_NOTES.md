## Discord Bridge 4.6.0

Türkçe: İlk herkese açık GitHub sürümü. Ağ ölçümünde tüm pencerenin ve ana düğmenin sürekli yenilenmesi kaldırıldı; yalnızca küçük ağ paneli güncellenir. Arayüz yuvarlak profil seçimleri ve yumuşak düğmelerle yenilendi. GoodbyeDPI / WinDivert motoru EXE içine gömülüdür. ExitLag ayarları değiştirilmez; Discord yalnızca düğmesine basıldığında açılır.

English: First public GitHub release. Traffic updates repaint only the network panel, with buffered native drawing and no periodic PowerShell process. The window scales to fit compact desktops. Includes rounded profile choices, saved settings, DNS backup/restore and explicit Discord launch.

### Downloads

- `DiscordBridge_4_6_Stable.exe`: self-contained Windows 10/11 x64 application.
- `DiscordBridge_4_6_Windows.zip`: executable, usage instructions and licenses.
- `DiscordBridge_4_6_KaynakKod.zip`: full app source plus corresponding third-party source archives.
- `SHA256SUMS.txt`: integrity checksums.

### Validation and scope

Publication is gated on Linux and Windows state tests, native Windows repaint/resource/control checks, engine provenance verification and a Windows build. The associated Actions run records the results and Windows UI capture. ISP-specific Superonline/ExitLag connectivity is not exercised on a GitHub runner, so this initial public version is marked **pre-release**.

The application is unsigned; no SignPath approval or Authenticode signature is claimed. The `Stable` filename is retained from the local repaint-fix build. GoodbyeDPI 0.2.3rc3 and WinDivert retain their original licenses; their source archives and notices are included.

## Code signing policy

Signing preparation is documented in [Code signing policy](docs/CODE_SIGNING_POLICY.md) and [Privacy policy](docs/PRIVACY.md). The SignPath Foundation eligibility application was submitted on 29 October 2026; provider approval is pending. Existing release files remain unsigned.
