package util

import (
	ecdsa2 "crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/A1andNS/newCrypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	"io"
	"log"
	"math/big"
	"os"
)

// Helper function to convert uint256 to a 32-byte array
func uint256ToBytes32(num *big.Int) [32]byte {
	var b [32]byte
	copy(b[32-len(num.Bytes()):], num.Bytes())
	return b
}

// 控制权验证hash token计算函数
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
	hash := sha256.New()
	//hash := sha3.NewLegacyKeccak256()
	hash.Write(concatenated[:])
	hashedBytes := hash.Sum(nil)

	// Convert to bytes32
	var result [32]byte
	copy(result[:], hashedBytes[:32])

	return hex.EncodeToString(result[:])
}

// 计算IPFS CID
func CalculateIPFSCid(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Failed to open file: %s\n", err)
		return ""
	}
	defer file.Close()

	// 计算文件的SHA-256哈希
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		fmt.Printf("Failed to calculate hash: %s\n", err)
		return ""
	}
	hash := hasher.Sum(nil)

	// 将哈希转换为Multihash
	mhash, err := mh.Encode(hash, mh.SHA2_256)
	if err != nil {
		fmt.Printf("Failed to encode multihash: %s\n", err)
		return ""
	}

	// 生成CID
	c := cid.NewCidV1(cid.Raw, mhash)

	// 输出CID
	fmt.Println("File CID:", c.String())
	return c.String()
}

// PrivateKeySignature 证书私钥签名证明
func PrivateKeySignature(privKey *ecdsa.PrivateKey, messageString string) ([32]byte, [32]byte, uint8) {
	// 要签名的消息
	message := []byte(messageString)
	// 对消息进行哈希（通常区块链使用 Keccak-256）
	hash := crypto.Keccak256Hash(message)
	// 使用私钥对哈希值进行签名
	stdPrivateKey := toStdPrivateKey(privKey)
	signature, err := crypto.Sign(hash[:], stdPrivateKey)
	if err != nil {
		log.Printf("Error signing message: %v\n", err)
		return [32]byte{}, [32]byte{}, 0
	}

	// 解析签名为 r, s, v
	r, s, v := parseSignature(signature)
	// 打印出签名的 r, s 和 v，用于 ecrecover() 验证
	fmt.Printf("Message: %s\n", message)
	fmt.Printf("Signature (r, s, v):\n")
	fmt.Printf("r: %s\n", r.Text(16)) // 打印 r 为十六进制
	fmt.Printf("s: %s\n", s.Text(16)) // 打印 s 为十六进制
	fmt.Printf("v: %d\n", v)
	fmt.Println(crypto.PubkeyToAddress(stdPrivateKey.PublicKey).String())
	return uint256ToBytes32(r), uint256ToBytes32(s), v
}

//0xDA5A26d837d8123D579217D5f0fb8351AcfF8dE2

func parseSignature(signature []byte) (*big.Int, *big.Int, uint8) {
	// 以太坊签名的长度是 65 字节，前 32 字节是 r，接下来的 32 字节是 s，最后一个字节是 v
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:64])
	v := signature[64]

	// 确保 r 和 s 都是 32 字节（如果它们小于 32 字节，则补零）
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// 如果 r 的字节长度小于 32 字节，则补零
	if len(rBytes) < 32 {
		rBytes = append(make([]byte, 32-len(rBytes)), rBytes...)
	}

	// 如果 s 的字节长度小于 32 字节，则补零
	if len(sBytes) < 32 {
		sBytes = append(make([]byte, 32-len(sBytes)), sBytes...)
	}

	// 重新设置 r 和 s 为补齐后的值
	r.SetBytes(rBytes)
	s.SetBytes(sBytes)

	// 以太坊规范中的 v 值通常是 27 或 28
	if v < 27 {
		v += 27
	}
	return r, s, v
}

func toStdPrivateKey(fiscoKey *ecdsa.PrivateKey) *ecdsa2.PrivateKey {
	return &ecdsa2.PrivateKey{
		PublicKey: ecdsa2.PublicKey{
			Curve: crypto.S256(),
			X:     fiscoKey.X,
			Y:     fiscoKey.Y,
		},
		D: fiscoKey.D,
	}
}

// 将 FISCO-BCOS 的公钥转换为标准库的公钥
func toStdPublicKey(fiscoKey *ecdsa.PublicKey) *ecdsa2.PublicKey {
	return &ecdsa2.PublicKey{
		Curve: crypto.S256(),
		X:     fiscoKey.X,
		Y:     fiscoKey.Y,
	}
}
