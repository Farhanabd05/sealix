package sigalg

import "crypto"

const (
	ProfileRSAPSS  = "rsa-pss-sha256"
	ProfileECDSA   = "ecdsa-p256-sha2"
	ProfileEd25519 = "ed25519"
)

type Algorithm interface {
	Name() string
	SigLen() int
	Generate() (crypto.Signer, error)
	CheckPublic(pub crypto.PublicKey) error
	Sign(k crypto.Signer, msg []byte) ([]byte, error)
	Verify(pub crypto.PublicKey, msg, sig []byte) bool
}
