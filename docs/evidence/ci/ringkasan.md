# Bukti CI/CD

Semua bukti di folder ini berasal dari image rilis `v1.0.0`.

| Item | Nilai |
|---|---|
| Tag repositori | `v1.0.0` (commit `d684de7`) |
| Tag image | `ghcr.io/farhanabd05/sealix:v1.0.0` |
| Image yang dipakai workflow | `ghcr.io/farhanabd05/sealix@sha256:157c26627908855d2a290ba676f960c66a37b4f5f46f72310933423a3ca1693a` |
| Platform | `linux/amd64` |

## Workflow run

| Workflow | Tautan |
|---|---|
| Build and publish sealix (tag `v1.0.0`) | https://github.com/Farhanabd05/sealix/actions/runs/43724487136 |
| Container integration C01-C05 | https://github.com/Farhanabd05/sealix/actions/runs/43724647967 |
| Release verification demo | https://github.com/Farhanabd05/sealix/actions/runs/43724708103 |

## Hasil C01 sampai C05

| Skenario | Profil | Exit code | Status | Penanda deployment |
|---|---|---|---|---|
| C01 | `ed25519` | 0 | `OK` | ada |
| C01 | `ecdsa-p256-sha2` | 0 | `OK` | ada |
| C01 | `rsa-pss-sha256` | 0 | `OK` | ada |
| C02 (satu byte artefak diubah) | `ed25519` | 1 | `ARTIFACT_MISMATCH` | tidak ada |
| C03 (kunci tidak diizinkan untuk paket) | `ed25519` | 1 | `KEY_NOT_TRUSTED` | tidak ada |
| C04 (artefak tidak ada) | `ed25519` | 2 | `INPUT_ERROR` | tidak ada |
| C05 (UID runner, tanpa TTY, `--network none`) | `ed25519` | 0 | `OK` | berkas keluaran milik `uid=1001` |

Exit code pada tabel dibaca dari proses `verify` yang sebenarnya, bukan dari
status job.

## Alur pemakai (sign, verify, deploy)

Job `sign` memakai secret `SIGNING_KEY_PEM`. Job `verify` tidak menerima secret
dan memakai `ci/trust.json` dari repositori. Job `deploy` hanya berjalan jika
`verify` berhasil.

- `verification.json`: `status` `OK`, `algorithm` `ed25519`, `keyid`
  `f216f1c5fa04b09d6c78e69a864543be15ed8400881e7abb855b0e964c7c5241`
  (kunci publik yang sama dengan entri di `ci/trust.json`).
- `deployed.marker` memuat SHA-256 artefak yang diverifikasi.

Kunci pada alur ini dibuat khusus untuk pengujian. Kunci privatnya hanya ada di
secret repositori, dan kunci publiknya ada di `ci/trust.json`.

## Isi folder

| Path | Isi |
|---|---|
| `integration/evidence-<profil>/` | exit code, keluaran JSON, stderr, dan penanda deployment per skenario |
| `integration/workflow.log` | log run integrasi |
| `pipeline-demo/` | artefak, bundle, hasil verifikasi, dan penanda deployment |
| `pipeline-demo/workflow.log` | log run alur pemakai |

Log disalin dari GitHub Actions. Nilai secret ditampilkan GitHub sebagai `***`,
dan tidak ada teks `PRIVATE KEY` di folder ini.
