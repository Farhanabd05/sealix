# Kunci khusus pengujian

Pasangan kunci di folder ini dibuat hanya untuk pengujian, satu pasang per
profil algoritma:

| Profil | Kunci privat | Kunci publik |
|---|---|---|
| `ed25519` | `ed25519.key.pem` | `ed25519.pub.pem` |
| `ecdsa-p256-sha2` | `ecdsa-p256-sha2.key.pem` | `ecdsa-p256-sha2.pub.pem` |
| `rsa-pss-sha256` | `rsa-pss-sha256.key.pem` | `rsa-pss-sha256.pub.pem` |

Kunci privat di sini sengaja ikut disimpan di repositori karena spesifikasi
meminta pasangan kunci uji. Jangan percayai kunci ini di luar pengujian, dan
jangan memakainya untuk menandatangani rilis sungguhan. Kunci untuk pipeline CI
dibuat terpisah dan kunci privatnya hanya ada di secret repositori.

Format berkas dan cara pembuatannya dijelaskan di `README.md` pada root
repositori:

```sh
sealix keygen --algorithm ed25519 --out test/keys/ed25519
```
