package util

import (
	"baseline1/DACME"
	"crypto/rsa"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

const PrivateKeyFilename string = "./conf/account/account"
const ContractAddressFilename string = "./conf/contractAddress"
const GroupID string = "group0"
const BlockchainConfigFilename string = "./conf/config.toml"
const NginxApplyCertificateFilePath string = "/var/www/html/well-known/pki-validation"
const NginxUpdateCertificateFilePath string = "/var/www/html/acme-challenge/update-certificate"
const NginxRevokeCertificateFilePath string = "/var/www/html/acme-challenge/revoke-certificate"
const OldPrivateKeyFilename string = "./output/privateKey.pem"
const CertificatePublicKeyFilePath = "./output/publicKey.pem"
const CertificatePrivateKeyFilePath = "./output/privateKey.pem"
const FileForApply int = 0
const FileForRevoke int = 1
const CertificateFilePath string = "./output/cert.crt"

type ACMEClient struct {
	ContractAddress       common.Address
	AccountPrivateKeyByte []byte
	//ClientSession              DpkiAcme.DpkiAcmeSession
	ClientSession              DACME.DACMESession
	Cid                        common.Address
	TimeNotAfter               *big.Int
	TimeNotBefore              *big.Int
	Ids                        []string
	HttpServerType             int
	CertificatePublicKeyString string
	CertificatePrivateKey      *rsa.PrivateKey
	PreHash                    [32]byte
	PreEncryptedString         string
}
