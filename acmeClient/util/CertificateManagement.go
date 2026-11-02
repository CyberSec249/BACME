package util

import (
	"crypto/rand"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"github.com/A1andNS/newCrypto/ecdsa"
	"github.com/A1andNS/newCrypto/x509"
	"github.com/FISCO-BCOS/go-sdk/core/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	ipfsApi "github.com/ipfs/go-ipfs-api"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Init initial an acme client
func Init(needServer bool) *ACMEClient {
	acmeClient := new(ACMEClient)
	// get contract address and private key(account)
	acmeClient.ContractAddress = GetContractAddress()
	log.Println("DPKI-ACME contract address: " + acmeClient.ContractAddress.Hex())
	// get a client session for smart contract
	acmeClient.Config, acmeClient.ClientSession, acmeClient.Account = GetClientSession(acmeClient.ContractAddress)
	acmeClient.IPFSClient = ipfsApi.NewShell("http://192.168.136.128:9001")
	acmeClient.DB = InitDB()
	if needServer {
		go StartFileServer()
	}
	return acmeClient
}

// ControlValidation 控制权验证接口
func ControlValidation(acmeClient *ACMEClient, ids string) bool {
	var receipt *types.Receipt
	var err error
	acmeClient.Ids = ids
	acmeClient.HttpServerType = 1
	// 控制权验证交易
	_, receipt, err = acmeClient.ClientSession.ControlValidate(acmeClient.Ids)
	if err != nil {
		log.Println(err)
		return false
	}
	if receipt.GetStatus() != 0 {
		log.Println(receipt.GetErrorMessage())
		return false
	}
	gasUsed, err := hexutil.DecodeUint64(receipt.GetGasUsed())
	if err != nil {
		return false
	}
	log.Println(fmt.Sprintf("交易Gas消耗：%v", gasUsed))
	log.Println("交易hash：" + receipt.GetTransactionHash())
	// 监听事件获取token1
	var token1 *big.Int
	for {
		blockNumber, _ := hexutil.DecodeUint64(receipt.GetBlockNumber())
		//log.Println(blockNumber)
		_, err = acmeClient.ClientSession.WatchRandomToken(&blockNumber, func(status int, logs []types.Log) {
			randomTokenEvent, err := acmeClient.ClientSession.ParseRandomToken(logs[0])
			//log.Println(randomTokenEvent.RandomToken, randomTokenEvent.ManagementAccount)
			if err != nil {
				log.Println(err)
				return
			}
			if randomTokenEvent.ManagementAccount == acmeClient.Account {
				token1 = randomTokenEvent.RandomToken
			}
		})
		if err != nil {
			log.Println(err)
			return false
		}
		if token1 == nil {
			time.Sleep(300 * time.Millisecond)
		} else {
			log.Println("获取到随机验证Token1为：" + token1.String())
			break
		}
	}
	// 生成token2和计算token
	token2 := generateUint256Random()
	log.Println("Token2:" + token2.String())
	token := DACME_HASH(token1.String(), token2.String())
	log.Println("Token:" + token)
	//部署token进行http验证
	if acmeClient.HttpServerType == 1 { // 启动内置服务器
		WriteTokenToBuiltinWeb(token)
	} else if acmeClient.HttpServerType == 2 { //使用系统中的nginx web服务器
		WriteTokenToNginxWeb(token)
	}
	// 发送开始验证交易
	_, receipt, err = acmeClient.ClientSession.StartValidation(token2)
	if err != nil {
		log.Println(err)
		return false
	}
	if receipt.GetStatus() != 0 {
		log.Println(receipt.GetErrorMessage())
		return false
	}
	gasUsed, err = hexutil.DecodeUint64(receipt.GetGasUsed())
	if err != nil {
		log.Println(err)
		return false

	}
	log.Println("智能合约开始进行HTTP验证ID控制权！")
	log.Println(fmt.Sprintf("交易Gas消耗：%v", gasUsed))
	log.Println("交易hash：" + receipt.GetTransactionHash())
	// 监听事件，判断是否验证成功
	var validationResult int
	for {
		blockNumber, _ := hexutil.DecodeUint64(receipt.GetBlockNumber())
		_, err = acmeClient.ClientSession.WatchValidationResult(&blockNumber, func(status int, logs []types.Log) {
			ValidationResultEvent, err := acmeClient.ClientSession.ParseValidationResult(logs[0])
			if err != nil {
				log.Println(err)
				return
			}
			if ValidationResultEvent.ManagementAccount == acmeClient.Account && ValidationResultEvent.Result {
				validationResult = 1
			} else if ValidationResultEvent.ManagementAccount == acmeClient.Account && !ValidationResultEvent.Result {
				validationResult = 2
			}
		})
		if err != nil {
			log.Println(err)
			return false
		}
		if validationResult == 0 {
			time.Sleep(300 * time.Millisecond)
		} else if validationResult == 1 {
			log.Println("ID控制权验证成功！")
			break
		} else if validationResult == 2 {
			log.Println("ID控制权验证失败！")
			return false
		}
	}
	log.Println("控制权验证完成")
	return true
}

// ApplyCertificate ACME-Client apply a certificate
func ApplyCertificate(acmeClient *ACMEClient, ids string) (bool, string) {
	var receipt *types.Receipt
	var err error
	var ok bool
	//获取ID列表和HTTP验证使用的服务器类型（1：内置http服务器，2. 本地nginx）
	acmeClient.Ids = ids
	acmeClient.HttpServerType = 1
	idList := strings.Split(ids, ",")
	filenameBase := strings.Replace(idList[0]+"_"+strconv.Itoa(len(idList)), ".", "_", -1)
	privateKeyPath := OutputPath + "/" + filenameBase + "_private.pem"
	publicKeyPath := OutputPath + "/" + filenameBase + "_public.pem"
	certPath := OutputPath + "/" + filenameBase + ".crt"
	// 生成证书公公钥和私钥
	acmeClient.CertificatePrivateKey, acmeClient.CertificatePublicKeyHash = GenerateECCKeyPair(privateKeyPath, publicKeyPath)
	// 生成SSL（x.509 v3）证书
	validTime := GenerateSSLCertificate(acmeClient.Ids, acmeClient.CertificatePrivateKey)
	log.Printf("证书文件在%v，证书公钥文件在%v，证书私钥文件在%v", certPath, publicKeyPath, privateKeyPath)
	// 上传自签名证书到IPFS
	file, err := os.Open(certPath)
	if err != nil {
		log.Println(err)
		return false, ""
	}
	acmeClient.IPFSCid, err = acmeClient.IPFSClient.Add(file, ipfsApi.CidVersion(1))
	if err != nil {
		log.Println(err)
		return false, ""
	}
	log.Printf("IPFS CID: %s", acmeClient.IPFSCid)
	// 开始新订单请求
	ok, _, receipt, err = acmeClient.ClientSession.ApplyCertificate(acmeClient.Ids, acmeClient.IPFSCid, acmeClient.CertificatePublicKeyHash)
	if err != nil {
		log.Println(err)
		return false, ""
	}
	if receipt.GetStatus() != 0 {
		log.Println(receipt.GetErrorMessage())
		return false, ""
	}
	gasUsed, err := hexutil.DecodeUint64(receipt.GetGasUsed())
	if err != nil {
		return false, ""
	}
	log.Println(fmt.Sprintf("交易Gas消耗：%v", gasUsed))
	log.Println("交易hash：" + receipt.GetTransactionHash())
	if ok {
		log.Println("证书申请完成")
		return true, validTime
	} else {
		log.Println("证书申请失败")
		return false, ""
	}
}

// ParserCertificate
func ParserOldCertificate(oldCertificateFilePath string) (string, string) {
	// 解析证书
	certBytes, err := os.ReadFile(oldCertificateFilePath)
	if err != nil {
		log.Println(err)
		return "", ""
	}
	// 尝试解析PEM编码的证书
	block1, _ := pem.Decode(certBytes)
	if block1 == nil || block1.Type != "CERTIFICATE" {
		log.Println("无法解码PEM块")
		return "", ""
	}
	certificate, err := x509.ParseCertificate(block1.Bytes)
	if err != nil {
		log.Println(err)
		return "", ""
	}
	domains := strings.Join(certificate.DNSNames, ",")
	for i := 0; i < len(certificate.IPAddresses); i++ {
		domains += "," + certificate.IPAddresses[i].String()
	}
	log.Println("这是旧证书中的ID", domains)
	oldCertCID := CalculateIPFSCid(oldCertificateFilePath)
	return domains, oldCertCID
}

// UpdateCertificate 更新证书
func UpdateCertificate(acmeClient *ACMEClient, oldCertCID string, ids string) (bool, string) {
	var err error
	var receipt *types.Receipt
	var ok bool
	// 生成新的公私密钥
	acmeClient.Ids = ids
	idList := strings.Split(acmeClient.Ids, ",")
	filenameBase := strings.Replace(idList[0]+"_"+strconv.Itoa(len(idList)), ".", "_", -1)
	privateKeyPath := OutputPath + "/" + filenameBase + "_private.pem"
	publicKeyPath := OutputPath + "/" + filenameBase + "_public.pem"
	certPath := OutputPath + "/" + filenameBase + ".crt"
	acmeClient.CertificatePrivateKey, acmeClient.CertificatePublicKeyHash = GenerateECCKeyPair(privateKeyPath, publicKeyPath)
	// 生成SSL（x.509 v3）证书
	validTime := GenerateSSLCertificate(acmeClient.Ids, acmeClient.CertificatePrivateKey)
	log.Printf("证书文件在%v，证书公钥文件在%v，证书私钥文件在%v", certPath, publicKeyPath, privateKeyPath)
	// 上传自签名证书到IPFS
	file, err := os.Open(certPath)
	if err != nil {
		log.Println(err)
		return false, ""
	}
	acmeClient.IPFSCid, err = acmeClient.IPFSClient.Add(file, ipfsApi.CidVersion(1))
	if err != nil {
		log.Println(err)
		return false, ""
	}
	log.Printf("IPFS CID: %s", acmeClient.IPFSCid)
	// 根据不同情况进行证书更新
	ok, _, receipt, err = acmeClient.ClientSession.UpdateCertificate(oldCertCID, acmeClient.IPFSCid, acmeClient.CertificatePublicKeyHash)
	if err != nil {
		log.Println(err)
		return false, ""
	}
	if receipt.GetStatus() != 0 {
		log.Println("交易失败")
		return false, ""
	}
	gasUsed, err := hexutil.DecodeUint64(receipt.GetGasUsed())
	if err != nil {
		log.Println(err)
		return false, ""
	}
	log.Println(fmt.Sprintf("交易Gas消耗：%v", gasUsed))
	log.Println("交易hash：" + receipt.GetTransactionHash())
	if ok {
		log.Println("证书更新成功")
		return true, validTime
	} else {
		log.Println("证书更新失败")
		return false, ""
	}
}

// RevokeCertificate 基于账户撤销证书
func RevokeCertificate(acmeClient *ACMEClient, oldCertCID string) bool {
	var err error
	var receipt *types.Receipt
	var ok bool

	ok, _, receipt, err = acmeClient.ClientSession.RevokeCertificate(oldCertCID)
	if err != nil {
		log.Println(err)
		return false
	}
	if receipt.GetStatus() != 0 {
		log.Println("交易失败")
		return false
	}
	gasUsed, err := hexutil.DecodeUint64(receipt.GetGasUsed())
	if err != nil {
		log.Println(err)
		return false
	}
	log.Println(fmt.Sprintf("交易Gas消耗：%v", gasUsed))
	log.Println("交易hash：" + receipt.GetTransactionHash())
	if ok {
		log.Println("证书撤销成功")
		return true
	} else {
		log.Println("证书撤销失败")
		return false
	}
}

// RevokeCertificateWithPrivateKey 基于私钥证明撤销证书
func RevokeCertificateWithPrivateKey(acmeClient *ACMEClient, oldCertCID string, oldCertPrivateKey *ecdsa.PrivateKey) bool {
	var err error
	var receipt *types.Receipt
	var ok bool
	r, s, v := PrivateKeySignature(oldCertPrivateKey, oldCertCID)
	ok, _, receipt, err = acmeClient.ClientSession.RevokeCertificateWithPrivateKey(oldCertCID, v, r, s)
	if err != nil {
		log.Println(err)
		return false
	}
	if receipt.GetStatus() != 0 {
		log.Println("交易失败")
		return false
	}
	gasUsed, err := hexutil.DecodeUint64(receipt.GetGasUsed())
	if err != nil {
		log.Println(err)
		return false
	}
	log.Println(fmt.Sprintf("交易Gas消耗：%v", gasUsed))
	log.Println("交易hash：" + receipt.GetTransactionHash())
	if ok {
		log.Println("证书撤销成功")
		return true
	} else {
		log.Println("证书撤销失败")
		return false
	}
}

// RevokeCertificateWithDV 基于控制权验证撤销证书
func RevokeCertificateWithDV(acmeClient *ACMEClient, oldCertCID string) bool {
	var err error
	var receipt *types.Receipt
	var ok bool
	ok, _, receipt, err = acmeClient.ClientSession.RevokeCertificateWithDV(oldCertCID)
	if err != nil {
		log.Println(err)
		return false
	}
	if receipt.GetStatus() != 0 {
		log.Println("交易失败")
		return false
	}
	gasUsed, err := hexutil.DecodeUint64(receipt.GetGasUsed())
	if err != nil {
		log.Println(err)
		return false
	}
	log.Println(fmt.Sprintf("交易Gas消耗：%v", gasUsed))
	log.Println("交易hash：" + receipt.GetTransactionHash())
	if ok {
		log.Println("证书撤销成功")
		return true
	} else {
		log.Println("证书撤销失败")
		return false
	}
}

// VerifyCertificate verify a certificate
func VerifyCertificate(acmeClient *ACMEClient, certificatePath string) (bool, *x509.Certificate, string) {
	// 解析证书
	certBytes, err := os.ReadFile(certificatePath)
	if err != nil {
		log.Println(err)
		return false, nil, ""
	}
	// 尝试解析PEM编码的证书
	block1, _ := pem.Decode(certBytes)
	if block1 == nil || block1.Type != "CERTIFICATE" {
		log.Println("无法解码PEM块")
		return false, nil, ""
	}
	certificate, err := x509.ParseCertificate(block1.Bytes)
	if err != nil {
		log.Println(err)
		return false, nil, ""
	}
	//log.Println("11111")
	// 检查证书是否是DPKI-ACNE证书还是传统PKI证书
	if !IsDPKICertificate(certificate) {
		// 如果是传统PKI证书，走传统验证渠道
		// 读取 Mozilla 的 CA 证书文件
		caCert, err := os.ReadFile("./conf/cacert.pem")
		if err != nil {
			fmt.Printf("读取根证书失败: %v\n", err)
		}

		// 创建证书池
		rootPool := x509.NewCertPool()
		if !rootPool.AppendCertsFromPEM(caCert) {
			fmt.Println("加载根证书失败")
		}
		// 配置验证选项
		opts := x509.VerifyOptions{
			Roots: rootPool, // 使用系统根证书池
			// 可以添加其他选项，如检查证书用途
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		}
		// 验证证书(不包括撤销验证)
		if _, err := certificate.Verify(opts); err != nil {
			fmt.Printf("无法验证证书: %v\n", err)
			return false, certificate, "无法验证证书"
		}
		return true, certificate, "验证成功"
	}
	// 如果是DPKI-ACME证书则走区块链验证
	// 计算证书CID
	CertCID := CalculateIPFSCid(certificatePath)
	// 区块链上验证证书信息
	ids, certPKHash, status, err := acmeClient.ClientSession.VerifyCertificate(CertCID)
	if err != nil {
		log.Println(err)
		return false, certificate, err.Error()
	}
	if !status {
		log.Println("证书已经被撤销")
		return false, certificate, "证书已经被撤销"
	} else {
		log.Println("证书未被撤销")
	}
	ips, domains, firstId := classifyStrings(ids)

	// 1. 检查证书公钥
	certPK := certificate.PublicKey.(*ecdsa.PublicKey)
	stdPublicKey := toStdPublicKey(certPK)
	//log.Println(certPKHash.String())
	if crypto.PubkeyToAddress(*stdPublicKey) != certPKHash {
		log.Println("公钥hash不匹配")
		return false, certificate, "公钥hash不匹配"
	}
	// 2. 检查证书主体
	if !checkSubject(firstId, certificate.Subject) {
		log.Println(firstId, certificate.Subject)
		log.Println("主体不合规")
		return false, certificate, "主体不合规"
	}
	// 3. 检查颁发者，需要和证书主体一致
	if !checkIssuer(firstId, certificate.Issuer) {
		log.Println("颁发者不合规")
		return false, certificate, "颁发者不合规"
	}
	// 4. 检查证书有效期是否为30日
	if certificate.NotAfter.Sub(certificate.NotBefore) != (30 * 24 * time.Hour) {
		log.Println("有效期不合规")
		return false, certificate, "有效期不合规"
	}
	// 5. 检查SAN字段是否正确
	if !checkSANIP(ips, certificate.IPAddresses) || !checkSANDomain(domains, certificate.DNSNames) {
		log.Println("SAN不合规")
		return false, certificate, "SAN不合规"
	}
	// 6. 检查自签名
	if certificate.CheckSignature(certificate.SignatureAlgorithm, certificate.RawTBSCertificate, certificate.Signature) != nil {
		log.Println("签名不合规")
		return false, certificate, "签名不合规"
	}
	// 验证其他禁用字段是否符合规定
	if !checkOtherBanField(certificate) {
		log.Println("禁用字段不合规")
		return false, certificate, "禁用字段不合规"
	}
	return true, certificate, "验证通过"
}

// checkSubject even if check the subject field is matched
func checkSubject(firstId string, subject pkix.Name) bool {
	return subject.CommonName == firstId && subject.Names[0].Type.Equal(asn1.ObjectIdentifier{2, 5, 4, 3}) && subject.Names[0].Value == firstId && subject.Country == nil && subject.Organization == nil && subject.OrganizationalUnit == nil && subject.Locality == nil && subject.Province == nil && subject.StreetAddress == nil && subject.PostalCode == nil && subject.SerialNumber == "" && subject.ExtraNames == nil
}

// checkIssuer even if check the subject field is matched
func checkIssuer(issuerCommonName string, issuer pkix.Name) bool {
	return issuer.CommonName == issuerCommonName && issuer.Names[0].Type.Equal(asn1.ObjectIdentifier{2, 5, 4, 3}) && issuer.Names[0].Value == issuerCommonName && issuer.Country == nil && issuer.Organization == nil && issuer.OrganizationalUnit == nil && issuer.Locality == nil && issuer.Province == nil && issuer.StreetAddress == nil && issuer.PostalCode == nil && issuer.SerialNumber == "" && issuer.ExtraNames == nil
}

// checkSANIP even if check the SANIP is match.
func checkSANIP(IPAddresses []net.IP, certIPAddresses []net.IP) bool {
	for i, v := range IPAddresses {
		if !v.Equal(certIPAddresses[i]) {
			return false
		}
	}
	return true
}

// checkSANDomain even if check the SANDomain is match
func checkSANDomain(domains []string, certDomain []string) bool {
	for i, v := range domains {
		if v != certDomain[i] {
			return false
		}
	}
	return true
}

// checkOtherBanField 主要是检测除规定使用的字段外，其他字段是否禁用，关键拓展是否按照默认规定使用。
func checkOtherBanField(cert *x509.Certificate) bool {
	return cert.Version == 3 &&
		cert.KeyUsage == (x509.KeyUsageDigitalSignature+x509.KeyUsageKeyEncipherment) &&
		cert.ExtraExtensions == nil &&
		cert.Extensions[len(cert.Extensions)-1].Id.Equal(asn1.ObjectIdentifier{1, 2, 3, 4, 5}) &&
		cert.UnhandledCriticalExtensions == nil &&
		cert.ExtKeyUsage[0] == x509.ExtKeyUsageServerAuth &&
		cert.UnknownExtKeyUsage == nil &&
		cert.BasicConstraintsValid &&
		!cert.IsCA &&
		cert.MaxPathLen == -1 &&
		!cert.MaxPathLenZero &&
		cert.SubjectKeyId == nil &&
		cert.AuthorityKeyId == nil &&
		cert.OCSPServer == nil &&
		cert.IssuingCertificateURL == nil &&
		cert.EmailAddresses == nil &&
		cert.URIs == nil &&
		!cert.PermittedDNSDomainsCritical &&
		cert.PermittedDNSDomains == nil &&
		cert.ExcludedDNSDomains == nil &&
		cert.PermittedIPRanges == nil &&
		cert.ExcludedIPRanges == nil &&
		cert.PermittedEmailAddresses == nil &&
		cert.ExcludedEmailAddresses == nil &&
		cert.PermittedURIDomains == nil &&
		cert.ExcludedURIDomains == nil &&
		cert.CRLDistributionPoints == nil &&
		cert.PolicyIdentifiers == nil
}

func generateUint256Random() *big.Int {
	// 创建一个*big.Int，其位长度设置为256（即最大值为2^256-1）
	maxUint256, _ := new(big.Int).SetString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)

	// 生成一个0到maxUint256之间的随机数
	randomBigInt, _ := rand.Int(rand.Reader, maxUint256)

	// 返回生成的随机数
	return randomBigInt
}
