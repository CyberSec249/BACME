package util

import (
	"encoding/hex"
	"golang.org/x/crypto/sha3"
	"math/big"
)

func DACME_HASH(n1 string, n2 string) string {
	// Example uint256 values
	a := new(big.Int)
	b := new(big.Int)

	a.SetString(n1, 10)
	b.SetString(n2, 10)
	// Convert uint256 to bytes32
	aBytes := uint256ToBytes32(a)
	bBytes := uint256ToBytes32(b)

	// Concatenate the bytes
	var concatenated [64]byte
	copy(concatenated[:32], aBytes[:])
	copy(concatenated[32:], bBytes[:])

	// Hash the concatenated bytes using keccak256
	hash := sha3.NewLegacyKeccak256()
	hash.Write(concatenated[:])
	hashedBytes := hash.Sum(nil)

	// Convert to bytes32
	var result [32]byte
	copy(result[:], hashedBytes[:32])

	return hex.EncodeToString(result[:])
}
