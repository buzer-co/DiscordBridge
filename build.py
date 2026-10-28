"""Reproducible offline packager. Requires Go 1.23+ and Python 3 to build only."""
from pathlib import Path
import hashlib
import os
import struct
import subprocess
import sys
import tempfile
import zipfile

root = Path(__file__).resolve().parent
out = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else root / "DiscordBridge_4_6_Stable.exe"
payload = (root / "engine.zip").read_bytes()
with zipfile.ZipFile(root / "engine.zip") as archive:
    required = {"goodbyedpi.exe", "WinDivert.dll", "WinDivert64.sys"}
    assert required.issubset(archive.namelist()), "Missing bundled engine component"
    for name in required:
        assert archive.read(name).startswith(b"MZ"), f"Invalid PE component: {name}"

env = dict(os.environ, GOOS="windows", GOARCH="amd64", CGO_ENABLED="0")
with tempfile.TemporaryDirectory(prefix="discordbridge-build-") as folder:
    raw, icon = Path(folder) / "raw.exe", Path(folder) / "icon.exe"
    subprocess.run(["go", "build", "-buildvcs=false", "-trimpath", "-ldflags=-s -w -H windowsgui", "-o", str(raw), "."], cwd=root, env=env, check=True)
    subprocess.run([sys.executable, str(root / "add_icon.py"), str(raw), str(icon), str(root / "DiscordBridge_Icon.ico")], check=True)
    trailer = b"DBRIDGE3" + struct.pack("<Q", len(payload)) + hashlib.sha256(payload).digest()
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_bytes(icon.read_bytes() + payload + trailer)
print(out.name, out.stat().st_size, "bytes")
print("SHA256", hashlib.sha256(out.read_bytes()).hexdigest())
