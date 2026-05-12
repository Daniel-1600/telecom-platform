package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
)

// generateAuthKeys generates a fresh Ki (K) and derives OPc for a new subscriber.
// Returns (K_hex, OPc_hex, error).
func (s *SubscriberService) generateAuthKeys() (string, string, error) {
	k := make([]byte, 16)
	if _, err := rand.Read(k); err != nil {
		return "", "", fmt.Errorf("failed to generate K: %w", err)
	}
	op := s.getOperatorVariant()
	opc, err := deriveOPc(k, op)
	if err != nil {
		return "", "", fmt.Errorf("failed to derive OPc: %w", err)
	}
	return hex.EncodeToString(k), hex.EncodeToString(opc), nil
}

// getOperatorVariant returns the 16-byte OP from environment configuration.
// OP must be kept secret and identical across all subscribers for the same operator.
func (s *SubscriberService) getOperatorVariant() []byte {
	opStr := os.Getenv("OPERATOR_VARIANT")
	if opStr == "" {
		opStr = "TelecomOP1234567" // 16 bytes — override in production via env
	}
	op := make([]byte, 16)
	copy(op, []byte(opStr))
	return op
}

// deriveOPc computes OPc = AES-128(K, OP) XOR OP per 3GPP TS 35.206 §3.
// BUG FIX: the previous implementation was missing the final XOR OP step.
func deriveOPc(k, op []byte) ([]byte, error) {
	if len(k) != 16 || len(op) != 16 {
		return nil, fmt.Errorf("K and OP must each be exactly 16 bytes")
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, fmt.Errorf("AES cipher init failed: %w", err)
	}
	opc := make([]byte, 16)
	block.Encrypt(opc, op)
	xorInPlace(opc, op) // OPc = AES-128(K, OP) XOR OP  ← was missing
	return opc, nil
}

// ─── Milenage f-functions (3GPP TS 35.206) ───────────────────────────────────

// milenageTemp computes the shared intermediate value used by all f-functions:
//
//	temp = AES-128(K, RAND XOR OPc)
func milenageTemp(block cipher.Block, rand16, opc []byte) []byte {
	out := make([]byte, 16)
	block.Encrypt(out, xorBytes(rand16, opc))
	return out
}

// milenageF1 computes MAC-A (8 bytes) — the network authentication code.
// 3GPP TS 35.206 §2.3: r1=64 bits (8 bytes), c1=0x00…00 (no-op XOR).
func milenageF1(block cipher.Block, temp, opc, sqn6, amf2 []byte) []byte {
	// in1 = SQN[0..5] || AMF[0..1] || SQN[0..5] || AMF[0..1]
	in1 := make([]byte, 16)
	copy(in1[0:6], sqn6)
	copy(in1[6:8], amf2)
	copy(in1[8:14], sqn6)
	copy(in1[14:16], amf2)

	x := xorBytes(rotLeft(xorBytes(temp, opc), 8), in1) // rot r1=8 bytes, c1=0 (no-op)
	out := make([]byte, 16)
	block.Encrypt(out, x)
	return xorBytes(out, opc)[:8] // MAC-A = out[0..7]
}

// milenageF2F5 computes RES (8 bytes) and AK (6 bytes) in one pass.
// 3GPP TS 35.206 §2.4 (f2) and §2.7 (f5): r2=0, c2=0x00…01.
func milenageF2F5(block cipher.Block, temp, opc []byte) (res, ak []byte) {
	x := xorBytes(temp, opc) // rot r2=0 bits — no rotation
	x[15] ^= 0x01            // XOR c2
	out := make([]byte, 16)
	block.Encrypt(out, x)
	result := xorBytes(out, opc)
	return append([]byte{}, result[8:16]...), append([]byte{}, result[0:6]...)
}

// milenageF3 computes CK (16 bytes) — the cipher key.
// 3GPP TS 35.206 §2.5: r3=32 bits (4 bytes), c3=0x00…02.
func milenageF3(block cipher.Block, temp, opc []byte) []byte {
	x := rotLeft(xorBytes(temp, opc), 4)
	x[15] ^= 0x02
	out := make([]byte, 16)
	block.Encrypt(out, x)
	return xorBytes(out, opc)
}

// milenageF4 computes IK (16 bytes) — the integrity key.
// 3GPP TS 35.206 §2.6: r4=64 bits (8 bytes), c4=0x00…04.
func milenageF4(block cipher.Block, temp, opc []byte) []byte {
	x := rotLeft(xorBytes(temp, opc), 8)
	x[15] ^= 0x04
	out := make([]byte, 16)
	block.Encrypt(out, x)
	return xorBytes(out, opc)
}

// ─── Byte utilities ───────────────────────────────────────────────────────────

// rotLeft rotates a 16-byte slice left by byteCount positions.
func rotLeft(x []byte, byteCount int) []byte {
	out := make([]byte, 16)
	for i := 0; i < 16; i++ {
		out[i] = x[(i+byteCount)%16]
	}
	return out
}

// xorBytes returns a XOR b (slices must be the same length).
func xorBytes(a, b []byte) []byte {
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}

// xorInPlace XORs b into a in place.
func xorInPlace(a, b []byte) {
	for i := range a {
		a[i] ^= b[i]
	}
}
