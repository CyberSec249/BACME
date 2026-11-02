package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"github.com/A1andNS/newCrypto/ecdsa"
	"github.com/A1andNS/newCrypto/elliptic"
	"github.com/A1andNS/newCrypto/x509"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// GenerateRSAKeyPair generate RSA key pair for certificate apply
func GenerateRSAKeyPair() (string, *rsa.PrivateKey) {
	bitSize := 2048
	// generate rsa private key
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		log.Fatal(err)
	}
	// serialize private key to PKCS8 format
	privateKeyPKCS8, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		log.Fatal(err)
	}
	// save private key to pem file
	privateKeyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PRIVATE KEY", // 注意这里使用的是 "PRIVATE KEY" 而不是 "RSA PRIVATE KEY"
			Bytes: privateKeyPKCS8,
		},
	)
	err = os.WriteFile(CertificatePrivateKeyFilePath, privateKeyPEM, 0600)
	if err != nil {
		log.Fatal(err)
	}
	// generate public key
	publicKey := &privateKey.PublicKey
	// converts a public key to PKIX, ASN.1 DER form. The encoded public key is a SubjectPublicKeyInfo structure (see RFC 5280, Section 4.1).
	publicKeyASN1, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		log.Fatal(err)
	}
	// save the public key to pem file
	publicKeyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyASN1,
		},
	)
	err = os.WriteFile(CertificatePublicKeyFilePath, publicKeyPEM, 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("RSA密钥对已成功生成并保存到文件。")
	return string(publicKeyPEM), privateKey
}

// GenerateECCKeyPair 生成ECC密钥
func GenerateECCKeyPair(skPath string, pkPath string) (*ecdsa.PrivateKey, common.Address) {
	// 这个函数默认使用secp256k1曲线
	privateKey, err := ecdsa.GenerateKey(elliptic.Secp256k1(), rand.Reader)
	if err != nil {
		fmt.Println("错误生成私钥:", err)
		return nil, common.Address{0}
	}
	// 打印私钥和公钥
	fmt.Println("私钥:", privateKey.D.String())
	publicKey := privateKey.PublicKey
	// 打印公钥
	publicKeyHex := fmt.Sprintf("0x%x", publicKey.X.Bytes())
	fmt.Println("公钥:", publicKeyHex)
	// 保存私钥和公钥到文件
	err = savePrivateKeyToFile(privateKey, skPath)
	if err != nil {
		fmt.Println("无法保存私钥:", err)
		return nil, common.Address{0}
	}
	err = savePublicKeyToFile(&publicKey, pkPath)
	if err != nil {
		fmt.Println("无法保存公钥:", err)
		return nil, common.Address{0}
	}
	fmt.Println("密钥成功保存")

	//公钥转bytes
	stdPublicKey := toStdPublicKey(&publicKey)
	address := crypto.PubkeyToAddress(*stdPublicKey)
	return privateKey, address
}

// 保存PEM格式的私钥到文件
func savePrivateKeyToFile(privateKey *ecdsa.PrivateKey, filename string) error {
	privBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return err
	}

	privFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer privFile.Close()

	privBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privBytes,
	}
	return pem.Encode(privFile, privBlock)
}

// 保存PEM格式的公钥到文件
func savePublicKeyToFile(publicKey *ecdsa.PublicKey, filename string) error {
	pubBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}

	pubFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer pubFile.Close()

	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	return pem.Encode(pubFile, pubBlock)
}

// isIPAddress check a string is ip or not
func isIPAddress(s string) bool {
	// 使用 net 包解析 IP 地址
	return net.ParseIP(s) != nil
}

// classifyStrings classify the string array to ip array and domain array
func classifyStrings(ids string) (ips []net.IP, domains []string, firstId string) {
	idArray := strings.Split(ids, ",")
	firstId = idArray[0]
	for _, str := range idArray {
		if isIPAddress(str) {
			ips = append(ips, net.ParseIP(str))
		} else {
			domains = append(domains, str)
		}
	}
	return
}

// GenerateSSLCertificate generate self-sign ssl certificate
func GenerateSSLCertificate(ids string, certificatePrivateKey *ecdsa.PrivateKey) string {
	// 证书生成时使用P256曲线
	//certificatePrivateKey.PublicKey.Curve = elliptic.P256()
	IPAddresses, dnsNames, firstId := classifyStrings(ids)
	timeNotBefore := time.Now()
	timeNotAfter := timeNotBefore.AddDate(0, 0, 30)
	// 创建自定义扩展
	customOID := asn1.ObjectIdentifier{1, 2, 3, 4, 5}
	customValue := "dpki-acme"
	customExtension := pkix.Extension{
		Id:    customOID,
		Value: []byte(customValue),
	}
	// create x509 certificate template
	template := x509.Certificate{
		SerialNumber:          big.NewInt(20250109), //SerialNumber is accountAddress
		Subject:               pkix.Name{CommonName: firstId},
		NotBefore:             timeNotBefore,
		NotAfter:              timeNotAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           IPAddresses,
		ExtraExtensions:       []pkix.Extension{customExtension},
	}
	//fmt.Println(template.NotAfter, template.NotBefore)
	// create x509 self-sign certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &certificatePrivateKey.PublicKey, certificatePrivateKey)
	if err != nil {
		log.Println(err)
		return ""
	}
	// save certificate to file
	certNameBase := "/" + firstId + "_" + strconv.Itoa(len(dnsNames)+len(IPAddresses))
	certPath := OutputPath + strings.Replace(certNameBase, ".", "_", -1) + ".crt"
	certFile, err := os.Create(certPath)
	if err != nil {
		log.Println(err)
		return ""
	}
	defer certFile.Close()

	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err != nil {
		log.Println(err)
		return ""
	}
	log.Println("自签名证书已经生成!!!")
	return timeNotAfter.Format("2006-01-02 15:04:05")
}

// ReadOldPrivateKey read the old private key from the specific path
func ReadOldPrivateKey(OldPrivateKeyPath string) *ecdsa.PrivateKey {
	oldPrivateKeyPem, err := os.ReadFile(OldPrivateKeyPath)
	if err != nil {
		log.Fatal(err)
	}
	block, _ := pem.Decode(oldPrivateKeyPem)
	if block == nil && block.Type != "PRIVATE KEY" {
		log.Fatal("无法解码PEM私钥")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		log.Fatal(err)
	}
	return privateKey
}

func IsDPKICertificate(certificate *x509.Certificate) bool {
	// 查找自定义标记扩展
	customOID := asn1.ObjectIdentifier{1, 2, 3, 4, 5}
	for _, ext := range certificate.Extensions {
		if ext.Id.Equal(customOID) && string(ext.Value) == "dpki-acme" {
			log.Printf("找到了DPKI-ACME证书标记扩展")
			return true
		}
	}
	return false
}
