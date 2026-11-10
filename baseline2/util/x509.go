package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"github.com/ethereum/go-ethereum/common"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
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
			Type:  "PRIVATE KEY", // note: "PRIVATE KEY", not "RSA PRIVATE KEY"
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
	log.Println("RSA key pair generated and saved to file.")
	return string(publicKeyPEM), privateKey
}

// isIPAddress check a string is ip or not
func isIPAddress(s string) bool {
	// Parse IP addresses with the net package
	return net.ParseIP(s) != nil
}

// classifyStrings classify the string array to ip array and domain array
func classifyStrings(stringArray []string) (ips []net.IP, domains []string) {
	for _, str := range stringArray {
		if isIPAddress(str) {
			ips = append(ips, net.ParseIP(str))
		} else {
			domains = append(domains, str)
		}
	}
	return
}

// GenerateSSLCertificate generate self-sign ssl certificate
func GenerateSSLCertificate(accountAddress common.Address, ids []string, timeNotBefore *big.Int, timeNotAfter *big.Int, certificatePrivateKey *rsa.PrivateKey) {
	// change big.Int to time.time
	timeNoteBeforeInt64 := timeNotBefore.Int64()
	timeNotBeforeNanoseconds := timeNoteBeforeInt64 * int64(time.Millisecond)
	timeNotBeforeUnix := time.Unix(0, timeNotBeforeNanoseconds).Truncate(time.Second).UTC()
	timeNoteAfterInt64 := timeNotAfter.Int64()
	timeNotAfterNanoseconds := timeNoteAfterInt64 * int64(time.Millisecond)
	timeNotAfterUnix := time.Unix(0, timeNotAfterNanoseconds).Truncate(time.Second).UTC()
	IPAddresses, dnsNames := classifyStrings(ids)
	// create x509 certificate template
	template := x509.Certificate{
		SerialNumber: accountAddress.Big(), //SerialNumber is accountAddress
		Subject: pkix.Name{
			CommonName: ids[0], //Subject is first ID
		},
		NotBefore:             timeNotBeforeUnix, // Smart contract return TimeNotBefore
		NotAfter:              timeNotAfterUnix,  // Smart contract return TimeNotAfter
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           IPAddresses,
	}
	//fmt.Println(template.NotAfter, template.NotBefore)
	// create x509 self-sign certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &certificatePrivateKey.PublicKey, certificatePrivateKey)
	if err != nil {
		log.Fatal(err)
	}
	// save certificate to file
	certFile, err := os.Create(CertificateFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer certFile.Close()

	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Self-Signed certificate has been generated!!!")
}

// GeneratePreAuthInfo calculate a sha256 hash of random number and encrypt result of random number
func GeneratePreAuthInfo(privateKey *rsa.PrivateKey) ([32]byte, string) {
	// generate 256-bit random number
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Fatalf("Failed to generate random number: %v", err)
	}
	randomString := hex.EncodeToString(randomBytes)
	log.Println("random string: " + randomString)
	// calculate sha256 hash value
	hash := sha256.Sum256([]byte(randomString))
	log.Printf("SHA-256 Hash: %x\n", hash)
	publicKey := &privateKey.PublicKey
	// encrypt the random string
	encryptedBytes, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, []byte(randomString), nil)
	if err != nil {
		log.Printf("Failed to encrypt: %v\n", err)
	}
	encryptedString := base64.StdEncoding.EncodeToString(encryptedBytes)
	log.Println("The RSA encrypted result:" + encryptedString)
	return hash, encryptedString
}

// DecryptPreAuthInfo Decrypt the preAuthText
func DecryptPreAuthInfo(encryptedString string, privateKey *rsa.PrivateKey) string {
	decodeString, err := base64.StdEncoding.DecodeString(encryptedString)
	if err != nil {
		log.Fatal(err)
	}
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, decodeString, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("The plainText is: ", string(plaintext))
	return string(plaintext)
}

// ReadOldPrivateKey read the old private key from the specific path
func ReadOldPrivateKey(i int) *rsa.PrivateKey {
	oldPrivateKeyPem, err := os.ReadFile(OldPrivateKeyFilename + strconv.Itoa(i))
	if err != nil {
		log.Fatal(err)
	}
	block, _ := pem.Decode(oldPrivateKeyPem)
	if block == nil && block.Type != "PRIVATE KEY" {
		log.Fatal("failed to decode PEM block containing private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		log.Fatal(err)
	}
	return privateKey.(*rsa.PrivateKey)
}
