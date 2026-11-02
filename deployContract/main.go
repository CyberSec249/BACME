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

var vrfCoreAddressString = flag.String("v", "", "VRFCore合约地址")
var hashKeyString = flag.String("h", "", "预言机的公钥哈希（hashKey）")
var oracleAddressString = flag.String("o", "", "OracleCore合约地址")

func DeployDpkiAcmeContract() {
	flag.Parse()
	if *vrfCoreAddressString == "" || *hashKeyString == "" || *oracleAddressString == "" {
		fmt.Println("请输入所有必要参数！")
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
	log.Println("合约地址为：" + address.Hex())
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
