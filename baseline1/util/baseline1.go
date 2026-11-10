package util

import (
	"baseline1/DACME"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/FISCO-BCOS/go-sdk/client"
	"github.com/FISCO-BCOS/go-sdk/conf"
	"github.com/FISCO-BCOS/go-sdk/core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"log"
	"math/big"
	"os"
	"time"
)

// Init initial an acme client
func Init() *ACMEClient {
	acmeClient := new(ACMEClient)
	// get contract address and private key(account)
	acmeClient.ContractAddress = GetContractAddress()
	log.Println("DPKI-ACME contract address: " + acmeClient.ContractAddress.Hex())
	// get a client session for smart contract
	acmeClient.ClientSession, acmeClient.Cid = GetClientSession(acmeClient.ContractAddress)
	return acmeClient
}

func DeployDACMEContract() {
	vrfCoreAddress := common.HexToAddress("0x58e3cab5c2d30558c5e01b697458b35d7fa06641")
	hashKey, _ := stringHashToBytes32("0x14d72138d742e0a53c0b00549b2bc7b169c3622eab9248760c4bde3b2850fedf")
	oracleAddress := common.HexToAddress("0xed9a4b19aff0cc6d4455de9759365ddd950d6977")
	configs, err := conf.ParseConfigFile(BlockchainConfigFilename)
	if err != nil {
		log.Fatal(err)
	}
	config := &configs[0]
	Client, err := client.Dial(config)
	address, r, _, err := DACME.DeployDACME(Client.GetTransactOpts(), Client, vrfCoreAddress, hashKey, oracleAddress)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Contract address: " + address.Hex())
	log.Println("Contract deployment gas: " + r.Gas().String())
}
func Baseline1ApplyACertificate(acmeClient *ACMEClient, domain string) {
	acmeClient.Ids, acmeClient.HttpServerType = []string{fmt.Sprintf(domain)}, 1
	acmeClient.CertificatePublicKeyString, acmeClient.CertificatePrivateKey = GenerateRSAKeyPair()

	// Generate the SSL (X.509 v3) certificate
	GenerateSSLCertificate(acmeClient.Cid, acmeClient.Ids, big.NewInt(time.Now().UnixNano()), big.NewInt(time.Now().UnixNano()+7776000), acmeClient.CertificatePrivateKey)
	var r *types.Receipt
	var err error
	_, r, err = acmeClient.ClientSession.RegisterDomain(acmeClient.Ids[0], acmeClient.CertificatePublicKeyString)
	if err != nil {
		log.Fatal(err)
	}
	if r.GetStatus() != 0 {
		log.Fatal(r.GetErrorMessage())
	}
	gasUsed, err := hexutil.DecodeUint64(r.GetGasUsed())
	if err != nil {
		return
	}
	log.Println(fmt.Sprintf("Transaction gas used: %v", gasUsed))
	log.Println("Transaction hash: " + r.GetTransactionHash())
	var n1 *big.Int
	// Listen for the event to obtain the token
	for {
		blockNumber, _ := hexutil.DecodeUint64(r.GetBlockNumber())
		_, err = acmeClient.ClientSession.WatchRandomNumberN1(&blockNumber, func(status int, logs []types.Log) {
			randomTokenEvent, err := acmeClient.ClientSession.ParseRandomNumberN1(logs[0])
			if err != nil {
				log.Fatal(err)
			}
			if randomTokenEvent.Domain == acmeClient.Ids[0] {
				n1 = randomTokenEvent.N1
			}
		})
		if err != nil {
			log.Fatal(err)
		}
		if n1 == nil {
			time.Sleep(300 * time.Millisecond)
		} else {
			log.Println("Received random validation token: " + n1.String())
			break
		}
	}
	n2, err := generateUint256Random()
	if err != nil {
		log.Fatal(err)
	}
	// Concatenate the byte
	log.Println("N2:" + n2.String())
	token := DACME_HASH(n1.String(), n2.String())
	log.Println("token is" + token)
	// Deploy the token for HTTP validation
	if acmeClient.HttpServerType == 1 { // start a build-in server
		// deploy token to http
		go StartServer(token, FileForApply)
	} else if acmeClient.HttpServerType == 2 { //use Nginx web on system
		// deploy token to http
		WriteTokenToNginxWeb(token, FileForApply)
	}

	//start a startValidation call
	_, r, err = acmeClient.ClientSession.StartValidation(acmeClient.Ids[0], n2)
	if err != nil {
		log.Fatal(err)
	}
	if r.GetStatus() != 0 {
		log.Fatal(r.GetErrorMessage())
	}
	log.Println("The smart contract started HTTP validation of the ID control right!")
	gasUsed, err = hexutil.DecodeUint64(r.GetGasUsed())
	if err != nil {
		return
	}
	log.Println(fmt.Sprintf("Transaction gas used: %v", gasUsed))
	log.Println("Transaction hash: " + r.GetTransactionHash())

	// Listen for the event to determine whether validation succeeded
	var validationResult int
	for {
		blockNumber, _ := hexutil.DecodeUint64(r.GetBlockNumber())
		_, err = acmeClient.ClientSession.WatchRegisterResult(&blockNumber, func(status int, logs []types.Log) {
			ValidationResultEvent, err := acmeClient.ClientSession.ParseRegisterResult(logs[0])
			if err != nil {
				log.Fatal(err)
			}
			if ValidationResultEvent.Author == acmeClient.Cid && ValidationResultEvent.Domain == acmeClient.Ids[0] && ValidationResultEvent.PublicKey == acmeClient.CertificatePublicKeyString {
				validationResult = 1
			} else {
				validationResult = 2
				//log.Println(ValidationResultEvent.Domain)
				//log.Println(ValidationResultEvent.PublicKey)
			}
		})
		if err != nil {
			log.Fatal(err)
		}
		if validationResult == 0 {
			time.Sleep(300 * time.Millisecond)
		} else if validationResult == 1 {
			log.Println("ID control validation succeeded!")
			break
		} else if validationResult == 2 {
			log.Fatalln("Failed")
		}
	}
	// Shut down the built-in HTTP server after control validation finishes
	if acmeClient.HttpServerType == 1 {
		ShutDownServer()
	}
	log.Println("Certificate application completed!")
	log.Println("Certificate file: ./cert.crt, certificate public key file: ./publicKey.pem, " +
		"certificate private key file: ./privateKey.pem")
}

func Baseline1UpdateCertificate(acmeClient *ACMEClient) {
	acmeClient.Ids = []string{fmt.Sprintf("www.t001.com")}
	acmeClient.CertificatePublicKeyString, acmeClient.CertificatePrivateKey = GenerateRSAKeyPair()
	// Generate the SSL (X.509 v3) certificate
	GenerateSSLCertificate(acmeClient.Cid, acmeClient.Ids, big.NewInt(time.Now().UnixNano()), big.NewInt(time.Now().UnixNano()+7776000), acmeClient.CertificatePrivateKey)
	var r *types.Receipt
	var err error
	_, r, err = acmeClient.ClientSession.UpdateCertificate(acmeClient.Ids[0], acmeClient.CertificatePublicKeyString)
	if err != nil {
		log.Fatal(err)
	}
	if r.GetStatus() != 0 {
		log.Fatal(r.GetErrorMessage())
	}
	gasUsed, err := hexutil.DecodeUint64(r.GetGasUsed())
	if err != nil {
		return
	}
	log.Println(fmt.Sprintf("Transaction gas used: %v", gasUsed))
	log.Println("Transaction hash: " + r.GetTransactionHash())
	log.Println("Certificate renewal completed!")
	log.Println("Certificate file: ./cert.crt, certificate public key file: ./publicKey.pem, " +
		"certificate private key file: ./privateKey.pem")
}

func Baseline1VerifyCertificate(acmeClient *ACMEClient) bool {
	certBytes, err := os.ReadFile(CertificateFilePath)
	if err != nil {
		log.Fatal(err)
	}
	// Try to parse the PEM-encoded certificate
	block, _ := pem.Decode(certBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		log.Fatal("Failed to decode PEM block containing certificate")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Fatal(err)
	}
	//log.Println("The certificate SerialNumber is:", cid.Hex())

	//startTime := time.Now()
	// start a verifyOrder transaction from blockchain
	pk, err := acmeClient.ClientSession.VerifyCertificate(certificate.Subject.CommonName)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Fetch the cert info from blockchain successfully!!!")
	}

	// convert public key string to publicKey type
	block, _ = pem.Decode([]byte(pk))
	if block == nil && block.Type != "PUBLIC KEY" {
		log.Fatal("failed to decode PEM block containing public key")
	}
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		log.Fatal(err)
	}
	// Check that the validity period, subject, public key, signature and SAN of the certificate match the on-chain record
	if certificate.PublicKey.(*rsa.PublicKey).Equal(publicKey.(*rsa.PublicKey)) &&
		certificate.CheckSignature(certificate.SignatureAlgorithm, certificate.RawTBSCertificate, certificate.Signature) == nil {
		return true
	} else {
		return false
	}
}
func generateUint256Random() (*big.Int, error) {
	// Create a *big.Int with a bit length of 256 (i.e., a maximum value of 2^256-1)
	maxUint256, _ := new(big.Int).SetString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)

	// Generate a random number between 0 and maxUint256
	randomBigInt, err := rand.Int(rand.Reader, maxUint256)
	if err != nil {
		return nil, err
	}

	// Return the generated random number
	return randomBigInt, nil
}

// Helper function to convert uint256 to a 32-byte array
func uint256ToBytes32(num *big.Int) [32]byte {
	var b [32]byte
	copy(b[32-len(num.Bytes()):], num.Bytes())
	return b
}
