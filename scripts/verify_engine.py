from pathlib import Path
import hashlib
import zipfile

root = Path(__file__).resolve().parents[1]
expected = {
    'goodbyedpi.exe': '8d412b094bb9c137ff25ba9a794d1122ecc84bb776debff6c249723a13cc31cd',
    'WinDivert.dll': '6110bfa44667405179c3e15e12af1b62037e447ed59b054b19042032995e6c7e',
    'WinDivert64.sys': 'e69b5ba3f0cd6cfb2983e442636e7f0b342b61b15264b0328317d4559c82cf50',
}
with zipfile.ZipFile(root / 'engine.zip') as z:
    for name, digest in expected.items():
        actual = hashlib.sha256(z.read(name)).hexdigest()
        if actual != digest:
            raise SystemExit(f'Engine hash mismatch: {name}')
        print(f'OK {name}: {actual}')
for name in ['GoodbyeDPI-0.2.3rc3-source.zip', 'WinDivert-2.2.0-source.zip']:
    with zipfile.ZipFile(root / 'third_party' / name) as z:
        if z.testzip() is not None or not z.namelist():
            raise SystemExit(f'Invalid corresponding source archive: {name}')
print('Engine and corresponding source archives verified.')
