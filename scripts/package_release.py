from pathlib import Path
import hashlib
import struct
import subprocess
import zipfile

root = Path(__file__).resolve().parents[1]
dist = root / 'dist'
exe = dist / 'DiscordBridge_4_6_Stable.exe'
b = exe.read_bytes()
assert b.startswith(b'MZ') and b[-48:-40] == b'DBRIDGE3'
length = struct.unpack('<Q', b[-40:-32])[0]
assert b[-48-length:-48] == (root / 'engine.zip').read_bytes()
assert hashlib.sha256(b[-48-length:-48]).digest() == b[-32:]
with zipfile.ZipFile(dist / 'DiscordBridge_4_6_Windows.zip', 'w', zipfile.ZIP_DEFLATED) as z:
    for path in [exe, root/'README.tr.md', root/'LICENSE', root/'THIRD_PARTY_NOTICES.md', root/'GOODBYEDPI_APACHE_2_LICENSE.txt', root/'WINDIVERT_LGPL_3_LICENSE.txt']:
        z.write(path, path.name)
# Tracked source only: never package settings, logs, temporary files or credentials.
tracked = subprocess.check_output(['git', 'ls-files', '-z'], cwd=root).decode().split('\0')
with zipfile.ZipFile(dist / 'DiscordBridge_4_6_KaynakKod.zip', 'w', zipfile.ZIP_DEFLATED) as z:
    for name in sorted(filter(None, tracked)):
        z.write(root / name, name)
    capture = root/'test-results/windows-ui.png'
    if capture.exists():
        z.write(capture, 'test-results/windows-ui.png')
with (dist / 'SHA256SUMS.txt').open('w', encoding='utf-8', newline='\n') as f:
    for path in sorted(dist.iterdir()):
        if path.name != 'SHA256SUMS.txt' and path.is_file():
            f.write(f'{hashlib.sha256(path.read_bytes()).hexdigest()}  {path.name}\n')
print('Windows executable, ZIP, complete source ZIP and checksums ready.')
