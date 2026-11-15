# sealix

`sealix` adalah CLI untuk menandatangani dan memverifikasi rilis perangkat lunak:
artefak dipetakan ke release manifest, ditandatangani secara digital, lalu
dikemas dalam bundle. Verifikasi menggunakan trust store kunci publik yang
dikelola pengguna.

Profil tanda tangan yang didukung:

- `rsa-pss-sha256`
- `ecdsa-p256-sha2`
- `ed25519`

`sealix` ditujukan untuk digunakan sebagai alat dalam pipeline CI/CD dan
didistribusikan dalam image kontainer. Tanda tangan membuktikan keaslian dan
integritas artefak, bukan bahwa perangkat lunak tersebut bebas malware.

## Tech stack

- Go, dengan versi minimum 1.24 pada [`app/go.mod`](app/go.mod), dikembangkan
  menggunakan Go 1.27.4.
- Docker multi-stage: image build menggunakan
  [`golang:1.24.13`](Dockerfile) dan image runtime akhir berbasis `scratch`.
- GitHub Actions untuk otomasi.
- GitHub Container Registry untuk distribusi image.
- OpenSSL 3.0.13 dan Python 3.12.3 dengan `cryptography` 41.0.7 digunakan
  hanya untuk pengujian interoperabilitas, bukan sebagai dependensi image.

## Dependensi

Program menggunakan standard library Go dan tidak memiliki dependensi pihak
ketiga. Tidak ada `go.sum`.

| Paket | Kegunaan | Dokumentasi |
|---|---|---|
| `crypto` | Tipe `crypto.Signer` dan `crypto.PublicKey` | [pkg.go.dev/crypto](https://pkg.go.dev/crypto) |
| `crypto/rsa` | Kunci dan operasi RSA-PSS | [pkg.go.dev/crypto/rsa](https://pkg.go.dev/crypto/rsa) |
| `crypto/ecdsa` | Kunci dan operasi ECDSA | [pkg.go.dev/crypto/ecdsa](https://pkg.go.dev/crypto/ecdsa) |
| `crypto/ed25519` | Kunci dan tanda tangan Ed25519 | [pkg.go.dev/crypto/ed25519](https://pkg.go.dev/crypto/ed25519) |
| `crypto/elliptic` | Kurva eliptik P-256 | [pkg.go.dev/crypto/elliptic](https://pkg.go.dev/crypto/elliptic) |
| `crypto/x509` | Serialisasi dan parsing kunci DER | [pkg.go.dev/crypto/x509](https://pkg.go.dev/crypto/x509) |
| `crypto/sha256` | Hash artefak dan manifest | [pkg.go.dev/crypto/sha256](https://pkg.go.dev/crypto/sha256) |
| `crypto/rand` | Sumber keacakan kriptografis | [pkg.go.dev/crypto/rand](https://pkg.go.dev/crypto/rand) |
| `math/big` | Bilangan `r` dan `s` pada tanda tangan ECDSA | [pkg.go.dev/math/big](https://pkg.go.dev/math/big) |
| `encoding/pem` | Encoding kunci PEM | [pkg.go.dev/encoding/pem](https://pkg.go.dev/encoding/pem) |
| `encoding/asn1` | Konversi signature ECDSA DER | [pkg.go.dev/encoding/asn1](https://pkg.go.dev/encoding/asn1) |
| `encoding/base64` | Encoding isi bundle | [pkg.go.dev/encoding/base64](https://pkg.go.dev/encoding/base64) |
| `encoding/hex` | Fingerprint dan hash dalam heksadesimal | [pkg.go.dev/encoding/hex](https://pkg.go.dev/encoding/hex) |
| `encoding/json` | Manifest, bundle, dan trust store | [pkg.go.dev/encoding/json](https://pkg.go.dev/encoding/json) |
| `flag` | Parsing argumen CLI | [pkg.go.dev/flag](https://pkg.go.dev/flag) |

Paket utilitas umum lain dari standard library (misalnya `fmt`, `os`, `io`,
`regexp`, dan `strings`) juga dipakai, tetapi tidak dicantumkan satu per satu.

## Menjalankan di luar runner CI/CD

Build binary lokal:

```sh
cd app
go build -o ../bin/sealix ./cmd/sealix
../bin/sealix --help
```

Menjalankan tes:

```sh
cd app
go test ./... -count=1
```

## Contoh penggunaan

Contoh berikut memakai artefak `contoh-app.zip`, paket `contoh-app`, versi `1.0.0`.

### keygen

```sh
sealix keygen --algorithm ed25519 --out keys/publisher
```

Menulis `keys/publisher.key.pem` dan `keys/publisher.pub.pem`, lalu mencetak
fingerprint kunci publik. Profil lain: `ecdsa-p256-sha2` dan `rsa-pss-sha256`.
Berkas yang sudah ada tidak ditimpa kecuali dengan `--force`.

### trust add

```sh
sealix trust add --key keys/publisher.pub.pem --package contoh-app --store trust.json
```

Membuat `trust.json` jika belum ada, lalu mencetak fingerprint, algoritma, dan
paket yang diizinkan. Menjalankan perintah yang sama dua kali tidak mengubah
apa pun. Kunci yang sama untuk paket lain menambah daftar paket pada entri yang
ada. Opsi `--keyid` menolak kunci jika fingerprint-nya berbeda dari yang
diharapkan.

### sign

```sh
sealix sign --file contoh-app.zip --package contoh-app --version 1.0.0 \
  --key keys/publisher.key.pem --out release.json
```

Menulis bundle ke `release.json`. Profil algoritma ditentukan dari kunci privat.
Bundle yang sudah ada tidak ditimpa kecuali dengan `--force`.

### verify

```sh
sealix verify --file contoh-app.zip --bundle release.json \
  --trust-store trust.json --package contoh-app --version 1.0.0 --json
```

Keluarannya satu objek JSON di stdout:

```json
{"valid":true,"status":"OK","message":"signature, metadata, dan artefak valid","algorithm":"ed25519","keyid":"<fingerprint>","package":"contoh-app","version":"1.0.0"}
```

Tanpa `--json`, hasilnya satu baris teks. Pesan kesalahan ditulis ke stderr.
Jika `--version` tidak sama dengan isi manifest, status menjadi
`METADATA_MISMATCH`.

### trust list dan trust remove

```sh
sealix trust list --store trust.json
sealix trust remove --keyid <fingerprint> --store trust.json
```

`trust remove` dengan keyid yang tidak ada keluar dengan kode 2. Di dalam
kontainer, mount direktori yang memuat trust store, bukan berkas tunggal,
karena penyimpanan trust store memakai `rename`.

### Exit code

| Kode | Arti |
|---|---|
| 0 | Berhasil. Pada `verify`: `status` `OK`. |
| 1 | Verifikasi ditolak: `FORMAT_ERROR`, `UNSUPPORTED_ALGORITHM`, `KEY_NOT_TRUSTED`, `INVALID_SIGNATURE`, `METADATA_MISMATCH`, atau `ARTIFACT_MISMATCH`. |
| 2 | Kesalahan pemakaian, I/O, atau lingkungan (`INPUT_ERROR` pada `verify`). |

## Format berkas

### Manifest

JSON dalam satu baris UTF-8, tanpa spasi dan tanpa baris baru, dengan urutan
field tetap dan ukuran maksimal 16 KiB. Contoh di bawah diberi indentasi hanya
supaya terbaca:

```json
{
  "schema": "minisign-manifest/v1",
  "package": "contoh-app",
  "version": "1.0.0",
  "algorithm": "ed25519",
  "keyid": "<64 karakter heksadesimal>",
  "artifact": {
    "name": "contoh-app.zip",
    "size": 1048576,
    "sha256": "<64 karakter heksadesimal>"
  }
}
```

| Field | Aturan |
|---|---|
| `schema` | `minisign-manifest/v1` |
| `package` | `[a-z0-9][a-z0-9._-]{0,63}` |
| `version` | `[A-Za-z0-9][A-Za-z0-9._+-]{0,63}` |
| `algorithm` | nama profil |
| `keyid` | SHA-256 dari DER SPKI kunci publik, heksadesimal huruf kecil |
| `artifact.name` | nama berkas tanpa direktori, `[A-Za-z0-9][A-Za-z0-9._-]{0,127}` |
| `artifact.size` | bilangan bulat 0 sampai 2^53 - 1 |
| `artifact.sha256` | SHA-256 isi artefak, heksadesimal huruf kecil |

Field tambahan, field ganda, nilai `null`, dan tipe yang salah ditolak. Saat
verifikasi, manifest diurai lalu diserialisasi ulang dan dibandingkan
byte per byte dengan manifest di bundle.

### Bundle

```json
{"schema":"minisign-bundle/v1","manifest_b64":"...","signature_b64":"..."}
```

Maksimal 64 KiB. Isi `manifest_b64` dan `signature_b64` memakai Base64 standar
(RFC 4648) dengan padding. Urutan field dan spasi pada bundle bebas, selama
kedua string Base64 tidak berubah.

### Kunci

| Jenis | Format |
|---|---|
| Kunci privat | PKCS#8 PEM, label `PRIVATE KEY` |
| Kunci publik | SPKI PEM, label `PUBLIC KEY` |
| RSA | modulus 3072 bit, eksponen publik 65537 |
| ECDSA | kurva P-256 dengan named curve |
| Ed25519 | RFC 8410 |

`keyid` dihitung dari DER SPKI hasil ekspor ulang kunci publik, jadi perbedaan
baris baru atau lebar baris pada berkas PEM tidak mengubahnya.

### Trust store

```json
{
  "schema": "minisign-trust/v1",
  "keys": [
    {
      "keyid": "<fingerprint>",
      "algorithm": "ed25519",
      "public_key_pem": "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----\n",
      "packages": ["contoh-app"]
    }
  ]
}
```

`public_key_pem` berisi teks PEM, bukan path berkas. Daftar `packages` tidak
boleh kosong, berisi duplikat, atau memakai wildcard. Ukuran berkas maksimal
1 MiB. Isi trust store hanya berubah lewat `trust add` dan `trust remove`.

## Pemetaan profil ke API pustaka

Semua fungsi berasal dari standard library Go. SHA-256 atas manifest dihitung
tepat satu kali, kecuali pada Ed25519 yang menerima manifest langsung.

| Profil | Keygen | Sign | Verify |
|---|---|---|---|
| `rsa-pss-sha256` | `rsa.GenerateKey(rand.Reader, 3072)` | `rsa.SignPSS` dengan `crypto.SHA256` dan `SaltLength: 32` | `rsa.VerifyPSS` dengan `SaltLength: 32` |
| `ecdsa-p256-sha2` | `ecdsa.GenerateKey(elliptic.P256(), rand.Reader)` | `ecdsa.SignASN1`, lalu DER diurai dengan `asn1.Unmarshal` menjadi `r` dan `s` (masing-masing 32 byte) | `r` dan `s` disusun kembali dengan `asn1.Marshal`, lalu `ecdsa.VerifyASN1` |
| `ed25519` | `ed25519.GenerateKey(rand.Reader)` | `ed25519.Sign` | `ed25519.Verify` |

Tanda tangan RSA-PSS berukuran 384 byte. Tanda tangan ECDSA disimpan sebagai
`r` diikuti `s` (64 byte), bukan DER. ECDSA dan RSA-PSS memakai `crypto/rand`
sebagai sumber keacakan (nonce dan salt), sedangkan Ed25519 bersifat
deterministik.

Kunci dan fingerprint:

| Operasi | Fungsi |
|---|---|
| Simpan kunci privat | `x509.MarshalPKCS8PrivateKey` dan `pem.EncodeToMemory` |
| Simpan kunci publik | `x509.MarshalPKIXPublicKey` dan `pem.EncodeToMemory` |
| Baca kunci privat | `pem.Decode` dan `x509.ParsePKCS8PrivateKey` |
| Baca kunci publik | `pem.Decode` dan `x509.ParsePKIXPublicKey` |
| Fingerprint | `sha256.Sum256` atas hasil `x509.MarshalPKIXPublicKey` |

Setelah dibaca, parameter kunci diperiksa lagi (RSA 3072 bit dengan eksponen
65537, ECDSA pada P-256), karena parser x509 menerima kunci dengan parameter
lain.

## Mengulang pengujian

| Pengujian | Perintah |
|---|---|
| Unit test, U01, dan U14 | `cd app && go test ./... -count=1` |
| Interop dengan OpenSSL dan Python | `bash test/interop/interop.sh` |
| Tes negatif U02 sampai U13 | `make negative` |
| Benchmark | `cd app && go run ./cmd/bench -bin ../bin/sealix -out ../bench/results/run1` |
| Integrasi C01 sampai C05 pada image | `bash test/integration/run.sh <image> <algoritma> <folder-keluaran>` |

## Image kontainer

TODO: dokumentasikan cara menarik image setelah tag rilis dan digest image
tersedia.