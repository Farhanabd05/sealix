#!/usr/bin/env bash
set -u

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
TOOLS="$ROOT/test/interop/tools.py"
W=$(mktemp -d)
trap 'rm -rf "$W"' EXIT

passed=0
failed=0

check() {
    local name=$1
    shift

    if "$@" >/dev/null 2>&1; then
        printf 'PASS %s\n' "$name"
        passed=$((passed + 1))
    else
        printf 'FAIL %s\n' "$name"
        failed=$((failed + 1))
    fi
}

check_status_ok() {
    local name=$1
    shift
    local output

    if output=$("$@" 2>/dev/null) &&
        [[ "$output" == *'"status":"OK"'* ]]; then
        printf 'PASS %s\n' "$name"
        passed=$((passed + 1))
    else
        printf 'FAIL %s\n' "$name"
        failed=$((failed + 1))
    fi
}

check_rejects_der() {
    local sealix=$1
    local tools=$2
    local manifest=$3
    local signature=$4
    local bundle=$5
    local trust=$6
    local artifact=$7
    local output
    local status

    python3 "$tools" bundle "$manifest" "$signature" "$bundle" \
        >/dev/null 2>&1 || return 1

    output=$("$sealix" verify \
        --file "$artifact" \
        --bundle "$bundle" \
        --trust-store "$trust" \
        --package contoh-app \
        --version 1.0.0 \
        --json 2>/dev/null)
    status=$?

    [[ $status -eq 1 && "$output" == *'"status":"INVALID_SIGNATURE"'* ]]
}

printf 'OpenSSL: '
openssl version
printf 'cryptography: '
python3 -c 'import cryptography; print(cryptography.__version__)'
printf 'Go: '
go version

(cd "$ROOT/app" && go build -o "$W/sealix" ./cmd/sealix) || exit 1

printf 'artefak interop\n' > "$W/artefak.bin"

keyid_from_public() {
    openssl pkey -in "$1" -pubout -outform DER |
        sha256sum |
        awk '{print $1}'
}

for profile in ed25519 ecdsa-p256-sha2 rsa-pss-sha256; do
    case "$profile" in
    ed25519)
        external_algorithm=ED25519
        ;;
    ecdsa-p256-sha2)
        external_algorithm=EC
        ;;
    rsa-pss-sha256)
        external_algorithm=RSA
        ;;
    esac

    ours="$W/ours-$profile"
    external="$W/external-$profile"
    mkdir -p "$ours" "$external"

    fingerprint=$("$W/sealix" keygen \
        --algorithm "$profile" \
        --out "$ours/k" 2>/dev/null)
    if [[ -n "$fingerprint" ]]; then
        printf 'PASS %s ours keygen\n' "$profile"
        passed=$((passed + 1))
    else
        printf 'FAIL %s ours keygen\n' "$profile"
        failed=$((failed + 1))
    fi

    check "$profile ours keyid" \
        test "$fingerprint" = "$(keyid_from_public "$ours/k.key.pem")"
    check "$profile ours sign" \
        "$W/sealix" sign \
        --file "$W/artefak.bin" \
        --package contoh-app \
        --version 1.0.0 \
        --key "$ours/k.key.pem" \
        --out "$ours/b.json"
    check "$profile extract ours bundle" \
        python3 "$TOOLS" extract "$ours/b.json" "$ours"
    check "$profile Python verifies ours" \
        python3 "$TOOLS" verify "$profile" \
        "$ours/k.pub.pem" "$ours/manifest.bin" "$ours/sig.bin"

    case "$profile" in
    ed25519)
        check "$profile OpenSSL verifies ours" \
            openssl pkeyutl -verify -rawin \
            -pubin -inkey "$ours/k.pub.pem" \
            -in "$ours/manifest.bin" -sigfile "$ours/sig.bin"
        ;;
    ecdsa-p256-sha2)
        check "$profile DER conversion" \
            python3 "$TOOLS" rs2der "$ours/sig.bin" "$ours/sig.der"
        check "$profile OpenSSL verifies ours" \
            openssl dgst -sha256 -verify "$ours/k.pub.pem" \
            -signature "$ours/sig.der" "$ours/manifest.bin"
        ;;
    rsa-pss-sha256)
        check "$profile OpenSSL verifies ours" \
            openssl dgst -sha256 -verify "$ours/k.pub.pem" \
            -sigopt rsa_padding_mode:pss \
            -sigopt rsa_pss_saltlen:32 \
            -sigopt rsa_mgf1_md:sha256 \
            -signature "$ours/sig.bin" "$ours/manifest.bin"
        ;;
    esac

    case "$external_algorithm" in
    ED25519)
        check "$profile external keygen" \
            openssl genpkey -algorithm ED25519 \
            -out "$external/ext.key.pem"
        ;;
    EC)
        check "$profile external keygen" \
            openssl genpkey -algorithm EC \
            -pkeyopt ec_paramgen_curve:P-256 \
            -pkeyopt ec_param_enc:named_curve \
            -out "$external/ext.key.pem"
        ;;
    RSA)
        check "$profile external keygen" \
            openssl genpkey -algorithm RSA \
            -pkeyopt rsa_keygen_bits:3072 \
            -pkeyopt rsa_keygen_pubexp:65537 \
            -out "$external/ext.key.pem"
        ;;
    esac

    check "$profile external public key" \
        openssl pkey -in "$external/ext.key.pem" \
        -pubout -out "$external/ext.pub.pem"
    external_keyid=$(keyid_from_public "$external/ext.key.pem")
    check "$profile external manifest" \
        python3 "$TOOLS" manifest \
        contoh-app 1.0.0 "$profile" "$external_keyid" \
        "$W/artefak.bin" "$external/manifest.bin"

    case "$profile" in
    ed25519)
        check "$profile external sign" \
            openssl pkeyutl -sign -rawin \
            -inkey "$external/ext.key.pem" \
            -in "$external/manifest.bin" \
            -out "$external/sig.bin"
        ;;
    ecdsa-p256-sha2)
        check "$profile external sign" \
            openssl dgst -sha256 -sign "$external/ext.key.pem" \
            -out "$external/sig.der" "$external/manifest.bin"
        check "$profile external signature conversion" \
            python3 "$TOOLS" der2rs \
            "$external/sig.der" "$external/sig.bin"
        ;;
    rsa-pss-sha256)
        check "$profile external sign" \
            openssl dgst -sha256 -sign "$external/ext.key.pem" \
            -sigopt rsa_padding_mode:pss \
            -sigopt rsa_pss_saltlen:32 \
            -sigopt rsa_mgf1_md:sha256 \
            -out "$external/sig.bin" "$external/manifest.bin"
        ;;
    esac

    check "$profile external bundle" \
        python3 "$TOOLS" bundle \
        "$external/manifest.bin" "$external/sig.bin" \
        "$external/b.json"
    check "$profile external trust" \
        "$W/sealix" trust add \
        --key "$external/ext.pub.pem" \
        --package contoh-app \
        --store "$external/trust.json"

    if [[ "$profile" == ecdsa-p256-sha2 ]]; then
        check "$profile rejects DER signature" \
            check_rejects_der "$W/sealix" "$TOOLS" \
            "$external/manifest.bin" "$external/sig.der" \
            "$external/der.json" "$external/trust.json" \
            "$W/artefak.bin"
    fi

    check_status_ok "$profile sealix verifies external" \
        "$W/sealix" verify \
        --file "$W/artefak.bin" \
        --bundle "$external/b.json" \
        --trust-store "$external/trust.json" \
        --package contoh-app \
        --version 1.0.0 \
        --json
done

printf '%d lulus, %d gagal\n' "$passed" "$failed"
((failed == 0))
