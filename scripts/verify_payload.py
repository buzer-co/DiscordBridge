"""Verify the engine trailer both before and after Authenticode appends a certificate."""
from pathlib import Path
import hashlib
import struct
import sys


def extract_payload(data):
    if len(data) < 48 or data[:2] != b'MZ':
        raise ValueError('Invalid executable')
    offsets = [len(data) - 48]
    if len(data) >= 0x40:
        pe = struct.unpack_from('<I', data, 0x3c)[0]
        opt = pe + 24
        if opt + 152 <= len(data) and data[pe:pe+4] == b'PE\0\0' and struct.unpack_from('<H', data, opt)[0] == 0x20b:
            cert, size = struct.unpack_from('<II', data, opt + 112 + 4*8)
            if size > 0 and 48 < cert <= len(data) and cert + size <= len(data):
                offsets.extend(cert - 48 - pad for pad in range(17))
    for end in offsets:
        if end < 0 or end + 48 > len(data) or data[end:end+8] != b'DBRIDGE3':
            continue
        length = struct.unpack_from('<Q', data, end + 8)[0]
        if not 0 < length <= min(40 << 20, end):
            continue
        payload = data[end-length:end]
        if hashlib.sha256(payload).digest() == data[end+16:end+48]:
            return payload
    raise ValueError('Embedded engine missing or corrupted')


def verify_payload_file(path):
    expected = (Path(__file__).resolve().parents[1] / 'engine.zip').read_bytes()
    if extract_payload(Path(path).read_bytes()) != expected:
        raise ValueError('Embedded engine differs from the approved source payload')


if __name__ == '__main__':
    verify_payload_file(sys.argv[1])
    print('Embedded engine intact; executable bytes were not modified.')
