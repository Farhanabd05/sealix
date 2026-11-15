# Bahan laporan: hasil benchmark

Sumber data: `bench/results/run1` (tabel utama) dan `bench/results/run2` (pengulangan). Median dan P95 dihitung dari 30 pengukuran setelah 5 pemanasan (keygen: 10 pengukuran). Median untuk n genap adalah rata-rata dua nilai tengah. P95 memakai metode nearest-rank, yaitu nilai pada posisi ceil(0,95 x n) dalam data terurut (posisi dimulai dari 1).

## Lingkungan

| Item | Nilai |
|---|---|
| Go | go1.27.4 (linux/amd64) |
| CPU | 11th Gen Intel(R) Core(TM) i5-1155G7 @ 2.50GHz, 4 core |
| RAM | 3.8 GiB |
| Kernel | Linux version 5.15.167.4-microsoft-standard-WSL2 |
| Pustaka | Go standard library (crypto/rsa, crypto/ecdsa, crypto/ed25519, crypto/sha256) |

## Pembangkitan kunci (median, P95; ms)

| Profil | n | Median run1 | P95 run1 | Median run2 |
|---|---:|---:|---:|---:|
| `ed25519` | 10 | 0.028 | 0.028 | 0.028 |
| `ecdsa-p256-sha2` | 10 | 0.031 | 0.032 | 0.022 |
| `rsa-pss-sha256` | 10 | 415.142 | 719.191 | 537.718 |

## Hash artefak SHA-256 (ms)

| Artefak | Median run1 | P95 run1 | Median run2 |
|---|---:|---:|---:|
| 1 KiB | 0.002 | 0.003 | 0.002 |
| 1 MiB | 1.079 | 1.350 | 1.070 |
| 64 MiB | 70.512 | 80.719 | 69.875 |

## Sign atas byte manifest (µs)

Tanpa I/O dan tanpa hashing artefak. SHA-256 internal algoritma ikut terhitung.

| Profil | Artefak | Median run1 | P95 run1 | Median run2 |
|---|---|---:|---:|---:|
| `ed25519` | 1 KiB | 47.7 | 64.6 | 34.8 |
| `ed25519` | 1 MiB | 41.2 | 54.4 | 34.8 |
| `ed25519` | 64 MiB | 41.8 | 60.1 | 34.8 |
| `ecdsa-p256-sha2` | 1 KiB | 52.2 | 67.1 | 52.6 |
| `ecdsa-p256-sha2` | 1 MiB | 52.2 | 73.2 | 52.5 |
| `ecdsa-p256-sha2` | 64 MiB | 52.3 | 64.6 | 52.3 |
| `rsa-pss-sha256` | 1 KiB | 4,617.7 | 4,766.2 | 4,614.8 |
| `rsa-pss-sha256` | 1 MiB | 4,628.3 | 4,766.8 | 4,626.8 |
| `rsa-pss-sha256` | 64 MiB | 4,613.3 | 4,731.7 | 4,620.7 |

## Verify atas byte manifest (µs)

Tanpa I/O dan tanpa hashing artefak. SHA-256 internal algoritma ikut terhitung.

| Profil | Artefak | Median run1 | P95 run1 | Median run2 |
|---|---|---:|---:|---:|
| `ed25519` | 1 KiB | 79.3 | 98.5 | 103.8 |
| `ed25519` | 1 MiB | 78.9 | 92.1 | 105.9 |
| `ed25519` | 64 MiB | 78.0 | 93.0 | 98.1 |
| `ecdsa-p256-sha2` | 1 KiB | 113.6 | 129.6 | 113.9 |
| `ecdsa-p256-sha2` | 1 MiB | 113.4 | 117.1 | 114.1 |
| `ecdsa-p256-sha2` | 64 MiB | 113.6 | 128.0 | 114.1 |
| `rsa-pss-sha256` | 1 KiB | 290.0 | 318.3 | 293.7 |
| `rsa-pss-sha256` | 1 MiB | 288.4 | 343.0 | 293.8 |
| `rsa-pss-sha256` | 64 MiB | 289.9 | 348.1 | 296.7 |

## End-to-end `sign` (ms)

Termasuk startup proses, I/O, hashing artefak, parsing, dan penulisan keluaran.

| Profil | Artefak | Median run1 | P95 run1 | Median run2 |
|---|---|---:|---:|---:|
| `ed25519` | 1 KiB | 6.651 | 8.522 | 6.483 |
| `ed25519` | 1 MiB | 7.910 | 9.667 | 7.746 |
| `ed25519` | 64 MiB | 86.372 | 97.152 | 85.985 |
| `ecdsa-p256-sha2` | 1 KiB | 4.887 | 7.169 | 4.621 |
| `ecdsa-p256-sha2` | 1 MiB | 5.839 | 7.398 | 5.729 |
| `ecdsa-p256-sha2` | 64 MiB | 84.077 | 94.995 | 83.698 |
| `rsa-pss-sha256` | 1 KiB | 10.173 | 10.829 | 10.133 |
| `rsa-pss-sha256` | 1 MiB | 11.416 | 12.268 | 11.288 |
| `rsa-pss-sha256` | 64 MiB | 88.120 | 92.795 | 89.794 |

## End-to-end `verify` (ms)

Termasuk startup proses, I/O, hashing artefak, parsing, dan penulisan keluaran.

| Profil | Artefak | Median run1 | P95 run1 | Median run2 |
|---|---|---:|---:|---:|
| `ed25519` | 1 KiB | 5.290 | 6.780 | 4.889 |
| `ed25519` | 1 MiB | 6.320 | 7.626 | 6.011 |
| `ed25519` | 64 MiB | 86.184 | 103.757 | 84.174 |
| `ecdsa-p256-sha2` | 1 KiB | 4.513 | 5.875 | 4.469 |
| `ecdsa-p256-sha2` | 1 MiB | 5.795 | 6.674 | 5.555 |
| `ecdsa-p256-sha2` | 64 MiB | 83.168 | 94.300 | 84.352 |
| `rsa-pss-sha256` | 1 KiB | 4.963 | 22.644 | 4.759 |
| `rsa-pss-sha256` | 1 MiB | 6.094 | 19.923 | 6.096 |
| `rsa-pss-sha256` | 64 MiB | 82.929 | 89.949 | 84.139 |

## Ukuran (byte)

| Profil | Kunci privat (DER PKCS#8) | Kunci publik (DER SPKI) | Tanda tangan (sebelum Base64) | Bundle 1 KiB | Bundle 1 MiB | Bundle 64 MiB |
|---|---:|---:|---:|---:|---:|---:|
| `ed25519` | 48 | 44 | 64 | 552 | 556 | 560 |
| `ecdsa-p256-sha2` | 138 | 91 | 64 | 564 | 568 | 568 |
| `rsa-pss-sha256` | 1792 | 422 | 384 | 984 | 988 | 992 |

## Angka turunan (dihitung dari tabel di atas, artefak 1 KiB)

| Perbandingan | Nilai |
|---|---:|
| Sign RSA-PSS dibanding Ed25519 | 97 kali |
| Sign RSA-PSS dibanding ECDSA | 88 kali |
| Verify RSA-PSS dibanding Ed25519 | 3.7 kali |
| Verify ECDSA dibanding Ed25519 | 1.4 kali |
| Keygen RSA-PSS dibanding Ed25519 | 14,793 kali |
| Hash 64 MiB dibanding `sign` end-to-end Ed25519 64 MiB | 70.5 ms dari 86.4 ms (82%) |
