package util

import (
	"acmeClient/DpkiAcme"
	"database/sql"
	"github.com/A1andNS/newCrypto/ecdsa"
	"github.com/FISCO-BCOS/go-sdk/conf"
	"github.com/ethereum/go-ethereum/common"
	ipfsApi "github.com/ipfs/go-ipfs-api"
)

const ContractAddressFilename string = "./conf/contractAddress"
const BlockchainConfigFilename string = "./conf/config.toml"
const ControlValidationFilePath string = "/var/www/html/acme/id-validation"
const ControlValidationBuiltinPath string = "./validation/acme/id-validation"
const CertificatePublicKeyFilePath = "./output/publicKey.pem"
const CertificatePrivateKeyFilePath = "./output/privateKey.pem"
const OutputPath string = "./output"
const FileServerDirectory = "./validation"
const port = "80"

type ACMEClient struct {
	ContractAddress          common.Address
	AccountPrivateKeyByte    []byte
	ClientSession            DpkiAcme.DpkiAcmeSession
	IPFSCid                  string
	Account                  common.Address
	Ids                      string
	HttpServerType           int
	CertificatePrivateKey    *ecdsa.PrivateKey
	CertificatePublicKeyHash common.Address
	IPFSClient               *ipfsApi.Shell
	Config                   *conf.Config
	DB                       *sql.DB
}
