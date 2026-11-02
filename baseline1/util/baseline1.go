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
	log.Println("合约地址为：" + address.Hex())
	log.Println("合约部署Gas消耗为：" + r.Gas().String())
}
func Baseline1ApplyACertificate(acmeClient *ACMEClient, domain string) {
	acmeClient.Ids, acmeClient.HttpServerType = []string{fmt.Sprintf(domain)}, 1
	acmeClient.CertificatePublicKeyString, acmeClient.CertificatePrivateKey = GenerateRSAKeyPair()

	// 生成SSL（x.509 v3）证书
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
	log.Println(fmt.Sprintf("交易Gas消耗：%v", gasUsed))
	log.Println("交易hash：" + r.GetTransactionHash())
	var n1 *big.Int
	// 监听事件获取token
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
			log.Println("获取到随机验证Token为：" + n1.String())
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
	//部署token进行http验证
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
	log.Println("智能合约开始进行HTTP验证ID控制权！")
	gasUsed, err = hexutil.DecodeUint64(r.GetGasUsed())
	if err != nil {
		return
	}
	log.Println(fmt.Sprintf("交易Gas消耗：%v", gasUsed))
	log.Println("交易hash：" + r.GetTransactionHash())

	// 监听事件，判断是否验证成功
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
			log.Println("ID控制权验证成功！")
			break
		} else if validationResult == 2 {
			log.Fatalln("失败")
		}
	}
	//控制权验证结束后，关闭内置的http服务器
	if acmeClient.HttpServerType == 1 {
		ShutDownServer()
	}
	log.Println("证书申请已经完成！")
	log.Println("证书文件在./cert.crt，证书公钥文件在./publicKey.pem，" +
		"证书私钥文件在./privateKey.pem")
}

func Baseline1UpdateCertificate(acmeClient *ACMEClient) {
	acmeClient.Ids = []string{fmt.Sprintf("www.t001.com")}
	acmeClient.CertificatePublicKeyString, acmeClient.CertificatePrivateKey = GenerateRSAKeyPair()
	// 生成SSL（x.509 v3）证书
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
	log.Println(fmt.Sprintf("交易Gas消耗：%v", gasUsed))
	log.Println("交易hash：" + r.GetTransactionHash())
	log.Println("证书更新已经完成！")
	log.Println("证书文件在./cert.crt，证书公钥文件在./publicKey.pem，" +
		"证书私钥文件在./privateKey.pem")
}

func Baseline1VerifyCertificate(acmeClient *ACMEClient) bool {
	certBytes, err := os.ReadFile(CertificateFilePath)
	if err != nil {
		log.Fatal(err)
	}
	// 尝试解析PEM编码的证书
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
	// 验证证书有效期是否和链上一致，主体字段是否匹配，公钥字段是否匹配，签名是否合法，SAN是否匹配,主要是检测是否和链上一致
	if certificate.PublicKey.(*rsa.PublicKey).Equal(publicKey.(*rsa.PublicKey)) &&
		certificate.CheckSignature(certificate.SignatureAlgorithm, certificate.RawTBSCertificate, certificate.Signature) == nil {
		return true
	} else {
		return false
	}
}
func generateUint256Random() (*big.Int, error) {
	// 创建一个*big.Int，其位长度设置为256（即最大值为2^256-1）
	maxUint256, _ := new(big.Int).SetString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)

	// 生成一个0到maxUint256之间的随机数
	randomBigInt, err := rand.Int(rand.Reader, maxUint256)
	if err != nil {
		return nil, err
	}

	// 返回生成的随机数
	return randomBigInt, nil
}

// Helper function to convert uint256 to a 32-byte array
func uint256ToBytes32(num *big.Int) [32]byte {
	var b [32]byte
	copy(b[32-len(num.Bytes()):], num.Bytes())
	return b
}
