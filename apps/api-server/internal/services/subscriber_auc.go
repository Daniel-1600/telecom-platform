package services

import (
	"context"
	"crypto/aes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/nutcas3/telecom-platform/apps/api-server/internal/models"
)

// defaultAMF is the Authentication Management Field (2 bytes).
// Bit 15 (separation bit) = 1 for 3G/4G per 3GPP TS 33.102 §6.3.3.
var defaultAMF = []byte{0x80, 0x00}

// AuthVector is a 3GPP EPS authentication vector (AV) returned to the MME/AMF.
type AuthVector struct {
	RAND string `json:"rand"` // 16 bytes hex — random challenge sent to UE
	XRES string `json:"xres"` // 8 bytes hex  — expected response from UE (f2)
	CK   string `json:"ck"`   // 16 bytes hex — cipher key (f3)
	IK   string `json:"ik"`   // 16 bytes hex — integrity key (f4)
	AUTN string `json:"autn"` // 16 bytes hex — (SQN XOR AK) || AMF || MAC-A
}

// GenerateAuthVector generates a fresh 3GPP EPS authentication vector for the
// subscriber identified by imsi. It atomically increments the subscriber's SQN
// in the database before computing the vector to prevent replay attacks.
func (s *SubscriberService) GenerateAuthVector(ctx context.Context, imsi models.IMSI) (*AuthVector, error) {
	// 1. Load subscriber — need K (AuthKey) and OPc
	sub, err := s.db.GetSubscriberByIMSI(ctx, imsi)
	if err != nil {
		return nil, fmt.Errorf("subscriber not found: %w", err)
	}

	// 2. Decode K and OPc from hex storage
	k, err := hex.DecodeString(sub.AuthKey)
	if err != nil || len(k) != 16 {
		return nil, fmt.Errorf("invalid AuthKey for subscriber %s: must be 32 hex chars", imsi)
	}
	opc, err := hex.DecodeString(sub.OPc)
	if err != nil || len(opc) != 16 {
		return nil, fmt.Errorf("invalid OPc for subscriber %s: must be 32 hex chars", imsi)
	}

	// 3. Atomically increment SQN — prevents replay attacks
	sqn, err := s.incrementSQN(ctx, sub)
	if err != nil {
		return nil, fmt.Errorf("failed to increment SQN: %w", err)
	}

	// 4. Generate RAND (16 cryptographically random bytes)
	randBytes := make([]byte, 16)
	if _, err := rand.Read(randBytes); err != nil {
		return nil, fmt.Errorf("failed to generate RAND: %w", err)
	}

	// 5. Run Milenage and return the auth vector
	return runMilenage(k, opc, randBytes, sqn, defaultAMF)
}

// runMilenage executes the full Milenage algorithm (3GPP TS 35.206) and
// returns a complete authentication vector.
func runMilenage(k, opc, randBytes []byte, sqn uint64, amf []byte) (*AuthVector, error) {
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, fmt.Errorf("AES cipher init failed: %w", err)
	}

	// Encode SQN as 6 bytes (big-endian, 48-bit counter)
	sqnBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(sqnBuf, sqn)
	sqn6 := sqnBuf[2:] // lower 6 bytes

	// Shared intermediate value: temp = AES-128(K, RAND XOR OPc)
	temp := milenageTemp(block, randBytes, opc)

	// f2 + f5: RES (expected response) and AK (anonymity key)
	res, ak := milenageF2F5(block, temp, opc)

	// f3: CK (cipher key)
	ck := milenageF3(block, temp, opc)

	// f4: IK (integrity key)
	ik := milenageF4(block, temp, opc)

	// f1: MAC-A (network authentication code) — uses raw SQN, not SQN XOR AK
	macA := milenageF1(block, temp, opc, sqn6, amf)

	// AUTN = (SQN XOR AK)[6] || AMF[2] || MAC-A[8]  — total 16 bytes
	sqnXorAK := xorBytes(sqn6, ak)
	autn := make([]byte, 16)
	copy(autn[0:6], sqnXorAK)
	copy(autn[6:8], amf)
	copy(autn[8:16], macA)

	return &AuthVector{
		RAND: hex.EncodeToString(randBytes),
		XRES: hex.EncodeToString(res),
		CK:   hex.EncodeToString(ck),
		IK:   hex.EncodeToString(ik),
		AUTN: hex.EncodeToString(autn),
	}, nil
}

// incrementSQN atomically increments the subscriber's SQN in PostgreSQL and
// returns the new value. Using UPDATE … RETURNING ensures no two concurrent
// requests can receive the same SQN.
func (s *SubscriberService) incrementSQN(ctx context.Context, sub *models.Subscriber) (uint64, error) {
	var newSQN int64
	err := s.db.DB.WithContext(ctx).
		Raw("UPDATE subscribers SET sqn = sqn + 1 WHERE id = ? RETURNING sqn", sub.ID).
		Scan(&newSQN).Error
	if err != nil {
		return 0, fmt.Errorf("SQN increment failed: %w", err)
	}
	return uint64(newSQN), nil
}
