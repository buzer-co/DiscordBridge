import hashlib
import struct
import unittest
from verify_payload import extract_payload


class PayloadVerification(unittest.TestCase):
    def fixture(self):
        header = bytearray(1024)
        header[:2] = b'MZ'
        struct.pack_into('<I', header, 0x3c, 0x80)
        header[0x80:0x84] = b'PE\0\0'
        struct.pack_into('<H', header, 0x80 + 24, 0x20b)
        payload = b'unit-test payload bytes'
        trailer = b'DBRIDGE3' + struct.pack('<Q', len(payload)) + hashlib.sha256(payload).digest()
        return header + payload + trailer, payload

    def test_unsigned_and_certificate_envelope(self):
        original, payload = self.fixture()
        self.assertEqual(extract_payload(original), payload)
        for padding in range(17):
            data = bytearray(original)
            cert_offset = len(data) + padding
            # Parser fixture only: no valid certificate/signature is created or claimed.
            certificate_fixture = b'not-a-real-certificate'
            struct.pack_into('<II', data, 0x80 + 24 + 112 + 4*8, cert_offset, len(certificate_fixture))
            data += b'\0' * padding + certificate_fixture
            self.assertEqual(extract_payload(data), payload)

    def test_corruption_and_invalid_lengths(self):
        data, _ = self.fixture()
        data[-1] ^= 1
        with self.assertRaises(ValueError):
            extract_payload(data)
        data, _ = self.fixture()
        struct.pack_into('<Q', data, len(data)-40, 1 << 50)
        with self.assertRaises(ValueError):
            extract_payload(data)
        for invalid in (b'', b'MZ', b'X'*500):
            with self.assertRaises(ValueError):
                extract_payload(invalid)


if __name__ == '__main__':
    unittest.main()
