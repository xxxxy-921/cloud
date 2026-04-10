#!/usr/bin/env python3
# pyright: reportMissingImports=false
import argparse
import base64
import hashlib
import json
from pathlib import Path

from cryptography.hazmat.primitives.ciphers.aead import AESGCM


def normalize_license_file_token(product_name: str) -> str:
    trimmed = product_name.strip()
    if not trimmed:
        return "LICENSE1"

    token = "".join(ch for ch in trimmed.upper()
                    if ch.isascii() and ch.isalnum())[:24]
    if token:
        return f"{token}1"

    digest = hashlib.sha256(trimmed.encode("utf-8")).digest()
    return f"LIC{digest[:4].hex().upper()}1"


def derive_license_file_key(registration_code: str, file_token: str) -> bytes:
    return hashlib.sha256(f"{file_token}:{registration_code}".encode("utf-8")).digest()


def decode_activation_code(code: str) -> dict:
    padding = "=" * (-len(code) % 4)
    data = base64.urlsafe_b64decode(code + padding)
    return json.loads(data.decode("utf-8"))


def _decrypt_base64_payload(encoded: str, key: bytes) -> dict:
    padding = "=" * (-len(encoded) % 4)
    data = base64.urlsafe_b64decode(encoded + padding)
    nonce, encrypted = data[:12], data[12:]
    plaintext = AESGCM(key).decrypt(nonce, encrypted, None)
    return json.loads(plaintext.decode("utf-8"))


def decrypt_license_text(ciphertext: str, registration_code: str) -> dict:
    normalized = ciphertext.strip()

    if "." not in normalized:
        raise ValueError("invalid license file format")

    file_token, encoded = normalized.split(".", 1)
    return _decrypt_base64_payload(
        encoded, derive_license_file_key(registration_code, file_token)
    )


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Decode a Metis .lic file and print activation payload")
    parser.add_argument("lic_file", help="Path to the .lic file")
    parser.add_argument("-r", "--registration-code",
                        help="Customer registration code for decrypting encrypted .lic")
    args = parser.parse_args()

    lic_path = Path(args.lic_file)
    raw_text = lic_path.read_text(encoding="utf-8").strip()

    if raw_text.startswith("{"):
        lic_data = json.loads(raw_text)
    else:
        if not args.registration_code:
            raise SystemExit("Encrypted .lic requires --registration-code")
        lic_data = decrypt_license_text(raw_text, args.registration_code)

    print("# Decrypted license JSON")
    print(json.dumps(lic_data, ensure_ascii=False, indent=2, sort_keys=True))

    activation_code = lic_data["activationCode"]
    decoded = decode_activation_code(activation_code)

    print("\n# Decoded activation payload")
    print(json.dumps(decoded, ensure_ascii=False, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
