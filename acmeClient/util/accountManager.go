package util

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"log"
	"os"
	"strings"
)

// CheckFileIsExist check the account file is exist or not
func CheckFileIsExist(filename string) bool {
	info, err := os.Lstat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
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
