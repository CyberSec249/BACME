package main

import (
	"deployContract/DpkiAcme"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/FISCO-BCOS/go-sdk/client"
	"github.com/FISCO-BCOS/go-sdk/conf"
	"github.com/ethereum/go-ethereum/common"
	"log"
)

var vrfCoreAddressString = flag.String("v", "", "VRFCore contract address")
var hashKeyString = flag.String("h", "", "public key hash of the oracle (hashKey)")
var oracleAddressString = flag.String("o", "", "OracleCore contract address")

func DeployDpkiAcmeContract() {
	flag.Parse()
	if *vrfCoreAddressString == "" || *hashKeyString == "" || *oracleAddressString == "" {
		fmt.Println("Please provide all required arguments!")
		return
	}
	vrfCoreAddress := common.HexToAddress(*vrfCoreAddressString)
	hashKey, _ := stringHashToBytes32(*hashKeyString)
	oracleAddress := common.HexToAddress(*oracleAddressString)
	configs, err := conf.ParseConfigFile("./conf/config.toml")
	if err != nil {
		log.Fatal(err)
	}
	config := &configs[0]
	Client, err := client.Dial(config)
	address, _, _, err := DpkiAcme.DeployDpkiAcme(Client.GetTransactOpts(), Client, vrfCoreAddress, hashKey, oracleAddress)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Contract address: " + address.Hex())
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

func main() {
	DeployDpkiAcmeContract()
}
