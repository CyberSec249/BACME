package util

import (
	"acmeClient/DpkiAcme"
	"encoding/hex"
	"fmt"
	"github.com/FISCO-BCOS/go-sdk/client"
	"github.com/FISCO-BCOS/go-sdk/conf"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
)

// GetClientSession create a new blockchain client session for DPKI-ACME smart contract
func GetClientSession(contractAddress common.Address) (*conf.Config, DpkiAcme.DpkiAcmeSession, common.Address) {
	configs, err := conf.ParseConfigFile(BlockchainConfigFilename)
	config := &configs[0]
	acmeClient, err := client.Dial(config)
	if err != nil {
		log.Fatal(err)
	}
	key, err := crypto.ToECDSA(config.PrivateKey)
	if err != nil {
		log.Fatal(err)
	}
	account := crypto.PubkeyToAddress(key.PublicKey)
	log.Printf("The loaded contract is: %v", contractAddress.Hex())
	log.Printf("The account is: %v\n", account.Hex())

	acmeInstance, err := DpkiAcme.NewDpkiAcme(contractAddress, acmeClient)
	if err != nil {
		log.Fatal(err)
	}
	acmeClientSession := DpkiAcme.DpkiAcmeSession{
		Contract:     acmeInstance,
		CallOpts:     *acmeClient.GetCallOpts(),
		TransactOpts: *acmeClient.GetTransactOpts(),
	}
	return config, acmeClientSession, account
}

func DeployDpkiAcmeContract() {
	vrfCoreAddress := common.HexToAddress("0x58e3cab5c2d30558c5e01b697458b35d7fa06641")
	hashKey, _ := stringHashToBytes32("0x14d72138d742e0a53c0b00549b2bc7b169c3622eab9248760c4bde3b2850fedf")
	oracleAddress := common.HexToAddress("0xed9a4b19aff0cc6d4455de9759365ddd950d6977")
	configs, err := conf.ParseConfigFile(BlockchainConfigFilename)
	if err != nil {
		log.Fatal(err)
	}
	config := &configs[0]
	Client, err := client.Dial(config)
	address, r, _, err := DpkiAcme.DeployDpkiAcme(Client.GetTransactOpts(), Client, vrfCoreAddress, hashKey, oracleAddress)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("合约地址为：" + address.Hex())
	log.Println("合约部署Gas消耗为：" + r.GasPrice().String())
}

func stringHashToBytes32(hash string) ([32]byte, error) {
	var b [32]byte
	decoded, err := hex.DecodeString(hash)
	if err != nil {
		return b, err
	}
	if len(decoded) > 32 {
		return b, fmt.Errorf("hash length exceeds 32 bytes")
	}
	copy(b[:], decoded)
	return b, nil
}
