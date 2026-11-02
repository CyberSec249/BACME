package util

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
	"os"
	"strings"
)

// GenerateAccount This function used to generate a new account for FISCO-BCOS blockchain.
func GenerateAccount() {
	// generate a ECDSA private key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	// save private key to file
	err = crypto.SaveECDSA(PrivateKeyFilename, privateKey)
	if err != nil {
		log.Fatal(err)
	}
	// Info print area
	log.Println("Private key saved to RootDirectory/conf/account/account")
	//fmt.Println("private key: ", hexutil.Encode(privateKeyBytes)[2:]) // privateKey in hex without "0x"
}

// CheckFileIsExist check the account file is exist or not
func CheckFileIsExist(filename string) bool {
	info, err := os.Lstat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// GetPrivateKey get the ECDSA private key from local file
func GetPrivateKey() ([]byte, common.Address) {
	// if the local file is not exist, generate account firstly
	if !CheckFileIsExist(PrivateKeyFilename) {
		GenerateAccount()
	}
	privateKey, err := crypto.LoadECDSA(PrivateKeyFilename)
	if err != nil {
		log.Fatal(err)
	}
	// generate public key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	//publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	//log.Println("Account publick key: ", hexutil.Encode(publicKeyBytes)[4:]) // publicKey in hex without "0x"
	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	log.Println("Account address: ", address) // account address
	return crypto.FromECDSA(privateKey), crypto.PubkeyToAddress(*publicKeyECDSA)
}

// GetContractAddress get the contract address from address config file
func GetContractAddress() common.Address {
	// check the contract address file is exist or not
	if !CheckFileIsExist(ContractAddressFilename) {
		var input string
		fmt.Println("Please enter the contract address with prefix 0x")
		fmt.Print("Contract Address: ")
		_, err := fmt.Scanln(&input)
		if err != nil {
			log.Fatal(err)
		}
		return common.HexToAddress(strings.TrimPrefix(input, "0x"))
	} else {
		f, err := os.ReadFile(ContractAddressFilename)
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Println(strings.TrimPrefix(string(f), "0x"))
		return common.HexToAddress(strings.TrimPrefix(string(f), "0x"))
	}
}
