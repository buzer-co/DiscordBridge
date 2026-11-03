## Discord Bridge 4.7.0 — selective encrypted DNS

- Start leaves Windows and modem DNS configuration unchanged. The previous automatic adapter-DNS change has been removed.
- Only the five Discord domain families and their subdomains use the in-process Cloudflare/Google DNS-over-HTTPS bridge. Other ordinary DNS queries keep their existing resolver.
- The bridge uses the existing original WinDivert driver, verified by hash, with normal TLS certificate validation and resolver bootstrap independent of system DNS. No hosts/NRPT/proxy/route changes.
- X/background mode keeps DNS and DPI alive in the tray. Stop/exit drains and closes the DNS filter. The old manual DNS button and restore backup remain available with explicit confirmation.
- The sidebar now navigates to visible tool pages with actions, progress, results and a working Overview return. Theme spacing, profile cards and selected navigation are polished; buffered drawing and narrow traffic repaints are retained.
- Persistent session/engine logs, language/path preferences and PID-file writes are removed. Only the last successful profile remains. Tool output is transient and cleared on navigation; old app-owned log files are removed.
- Duplicate modes are consolidated; original missing modes 1, 7 and 8 are added with readable profile names. The last successful method is remembered. The UDP/443-restricting method remains optional and is skipped in fresh automatic discovery.

Scope: ordinary IPv4/IPv6 UDP/53 only; loopback DNS proxies, caller-managed DoH/DoT, DNS over TCP and IPv6 extension headers are not intercepted. Native tests use controlled responses and do not establish compatibility with every ISP or ExitLag configuration. This is an unsigned test candidate, not a trusted signed release.

[Usage and limitations](https://github.com/buzer-co/DiscordBridge/blob/main/README.tr.md) · [Privacy policy](https://github.com/buzer-co/DiscordBridge/blob/main/docs/PRIVACY.md) · [Signing policy](https://github.com/buzer-co/DiscordBridge/blob/main/docs/CODE_SIGNING_POLICY.md)
