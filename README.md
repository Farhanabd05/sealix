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