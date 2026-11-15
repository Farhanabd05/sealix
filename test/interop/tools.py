#!/usr/bin/env python3

import argparse
import base64
import hashlib
import json
import sys
from pathlib import Path

from cryptography import exceptions
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ed25519, ec, padding, rsa, utils


def extract(bundle_path: Path, outdir: Path) -> None:
    bundle = json.loads(bundle_path.read_bytes())
    manifest = base64.b64decode(bundle["manifest_b64"], validate=True)
    signature = base64.b64decode(bundle["signature_b64"], validate=True)
    outdir.mkdir(parents=True, exist_ok=True)
    (outdir / "manifest.bin").write_bytes(manifest)
    (outdir / "sig.bin").write_bytes(signature)


def verify(profile: str, public_key_path: Path, manifest_path: Path, signature_path: Path) -> int:
    public_key = serialization.load_pem_public_key(public_key_path.read_bytes())
    manifest = manifest_path.read_bytes()
    signature = signature_path.read_bytes()

    try:
        if profile == "ed25519":
            if not isinstance(public_key, ed25519.Ed25519PublicKey):
                return 1
            public_key.verify(signature, manifest)
        elif profile == "ecdsa-p256-sha2":
            if len(signature) != 64:
                return 1
            if not isinstance(public_key, ec.EllipticCurvePublicKey):
                return 1
            r = int.from_bytes(signature[:32], "big")
            s = int.from_bytes(signature[32:], "big")
            der_signature = utils.encode_dss_signature(r, s)
            public_key.verify(der_signature, manifest, ec.ECDSA(hashes.SHA256()))
        elif profile == "rsa-pss-sha256":
            if not isinstance(public_key, rsa.RSAPublicKey):
                return 1
            public_key.verify(
                signature,
                manifest,
                padding.PSS(
                    mgf=padding.MGF1(hashes.SHA256()),
                    salt_length=32,
                ),
                hashes.SHA256(),
            )
        else:
            raise ValueError(f"profil tidak dikenal: {profile}")
    except exceptions.InvalidSignature:
        return 1
    return 0


def rs2der(signature_path: Path, out_path: Path) -> None:
    signature = signature_path.read_bytes()
    if len(signature) != 64:
        raise ValueError("signature r||s harus berukuran 64 byte")
    r = int.from_bytes(signature[:32], "big")
    s = int.from_bytes(signature[32:], "big")
    out_path.write_bytes(utils.encode_dss_signature(r, s))


def der2rs(signature_path: Path, out_path: Path) -> None:
    r, s = utils.decode_dss_signature(signature_path.read_bytes())
    out_path.write_bytes(r.to_bytes(32, "big") + s.to_bytes(32, "big"))


def write_manifest(
    package: str,
    version: str,
    profile: str,
    key_id: str,
    artifact_path: Path,
    out_path: Path,
) -> None:
    artifact = artifact_path.read_bytes()
    obj = {
        "schema": "minisign-manifest/v1",
        "package": package,
        "version": version,
        "algorithm": profile,
        "keyid": key_id,
        "artifact": {
            "name": artifact_path.name,
            "size": len(artifact),
            "sha256": hashlib.sha256(artifact).hexdigest(),
        },
    }
    out_path.write_text(json.dumps(obj, separators=(",", ":")), encoding="utf-8")


def write_bundle(manifest_path: Path, signature_path: Path, out_path: Path) -> None:
    obj = {
        "schema": "minisign-bundle/v1",
        "manifest_b64": base64.b64encode(manifest_path.read_bytes()).decode("ascii"),
        "signature_b64": base64.b64encode(signature_path.read_bytes()).decode("ascii"),
    }
    out_path.write_text(json.dumps(obj, separators=(",", ":")), encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest="command", required=True)

    extract_parser = subparsers.add_parser("extract")
    extract_parser.add_argument("bundle", type=Path)
    extract_parser.add_argument("outdir", type=Path)

    verify_parser = subparsers.add_parser("verify")
    verify_parser.add_argument("profile")
    verify_parser.add_argument("public_key", type=Path)
    verify_parser.add_argument("manifest", type=Path)
    verify_parser.add_argument("signature", type=Path)

    rs2der_parser = subparsers.add_parser("rs2der")
    rs2der_parser.add_argument("signature_rs", type=Path)
    rs2der_parser.add_argument("out", type=Path)

    der2rs_parser = subparsers.add_parser("der2rs")
    der2rs_parser.add_argument("signature_der", type=Path)
    der2rs_parser.add_argument("out", type=Path)

    manifest_parser = subparsers.add_parser("manifest")
    manifest_parser.add_argument("package")
    manifest_parser.add_argument("version")
    manifest_parser.add_argument("profile")
    manifest_parser.add_argument("keyid")
    manifest_parser.add_argument("artifact", type=Path)
    manifest_parser.add_argument("out", type=Path)

    bundle_parser = subparsers.add_parser("bundle")
    bundle_parser.add_argument("manifest", type=Path)
    bundle_parser.add_argument("signature", type=Path)
    bundle_parser.add_argument("out", type=Path)

    args = parser.parse_args()
    if args.command == "extract":
        extract(args.bundle, args.outdir)
    elif args.command == "verify":
        return verify(args.profile, args.public_key, args.manifest, args.signature)
    elif args.command == "rs2der":
        rs2der(args.signature_rs, args.out)
    elif args.command == "der2rs":
        der2rs(args.signature_der, args.out)
    elif args.command == "manifest":
        write_manifest(args.package, args.version, args.profile, args.keyid, args.artifact, args.out)
    elif args.command == "bundle":
        write_bundle(args.manifest, args.signature, args.out)
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError, KeyError, json.JSONDecodeError) as error:
        print(f"error: {error}", file=sys.stderr)
        sys.exit(2)
