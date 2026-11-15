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

## Image kontainer

TODO: dokumentasikan cara menarik image setelah tag rilis dan digest image
tersedia.