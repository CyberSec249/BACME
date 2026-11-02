package util

import (
	"baseline1/DACME"
	"encoding/hex"
	"fmt"
	"github.com/FISCO-BCOS/go-sdk/client"
	"github.com/FISCO-BCOS/go-sdk/conf"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
)

// GetClientSession create a new blockchain client session for DPKI-ACME smart contract
func GetClientSession(contractAddress common.Address) (DACME.DACMESession, common.Address) {
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

	acmeInstance, err := DACME.NewDACME(contractAddress, acmeClient)
	if err != nil {
		log.Fatal(err)
	}
	acmeClientSession := DACME.DACMESession{
		Contract:     acmeInstance,
		CallOpts:     *acmeClient.GetCallOpts(),
		TransactOpts: *acmeClient.GetTransactOpts(),
	}
	return acmeClientSession, account
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
