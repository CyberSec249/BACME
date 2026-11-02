package main

import (
	"acmeClient/DpkiAcme"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/FISCO-BCOS/go-sdk/client"
	"github.com/FISCO-BCOS/go-sdk/conf"
)

const (
	configFile = "./conf/config.toml" // 配置文件路径
	groupID    = "group0"             // 群组ID
	qps        = 10000                  // 每秒交易数 (QPS)
	totalTx    = 10000                // 总交易数
	poolSize   = 128                  // 线程池大小
)

var (
	sentTxCount  int32 // 已发送交易数
	successCount int32 // 成功交易数
	failureCount int32 // 失败交易数
	wg           sync.WaitGroup
)

func main() {
	// 加载配置文件
	configs, err := conf.ParseConfigFile(configFile)
	if err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	// 创建客户端池
	clientPool := make(chan *client.Client, poolSize)
	for i := 0; i < poolSize; i++ {
		c, err := client.Dial(&configs[0])
		if err != nil {
			log.Fatalf("Failed to connect to FISCO-BCOS: %v", err)
		}
		clientPool <- c
	}

	// 部署合约
	contractAddress := common.HexToAddress("0x9014be51ce391a7afc48ddca23ba9b70dcf675a7")
	fmt.Println("智能合约地址：" + contractAddress.String())

	// 启动压力测试
	fmt.Println("====== Starting pressure test ======")
	startTime := time.Now()

	// 使用限流器控制QPS
	limiter := time.Tick(time.Second / time.Duration(qps))

	// 创建任务通道
	taskChan := make(chan struct{}, totalTx)

	// 启动线程池
	for i := 0; i < poolSize; i++ {
		go worker(clientPool, taskChan, contractAddress)
	}

	// 提交任务
	for i := 0; i < totalTx; i++ {
		<-limiter // 限流
		taskChan <- struct{}{}
	}

	// 关闭任务通道
	close(taskChan)

	// 等待所有任务完成
	wg.Wait()

	for {
		if atomic.LoadInt32(&successCount)+atomic.LoadInt32(&failureCount) == totalTx {
			break
		}
	}

	// 计算并输出性能统计结果
	duration := time.Since(startTime)
	tps := float64(totalTx-failureCount) / duration.Seconds()

	fmt.Println("====== Pressure test completed ======")
	fmt.Printf("Total transactions: %d\n", totalTx)
	fmt.Printf("Failed rate: %v\n", float64(failureCount)/float64(totalTx))
	fmt.Printf("Time elapsed: %v\n", duration)
	fmt.Printf("Transactions per second (TPS): %.2f\n", tps)
}

// 工作线程
func worker(clientPool chan *client.Client, taskChan chan struct{}, contractAddress common.Address) {
	for range taskChan {
		wg.Add(1)
		defer wg.Done()

		c := <-clientPool // 从客户端池获取客户端
		sendTransaction(c, contractAddress)
		clientPool <- c // 将客户端放回客户端池
	}
}

// 发送交易
func sendTransaction(client *client.Client, contractAddress common.Address) {
	//auth := client.GetTransactOpts()
	instance, err := DpkiAcme.NewDpkiAcme(contractAddress, client)
	if err != nil {
		log.Printf("Failed to create contract instance: %v", err)
		atomic.AddInt32(&failureCount, 1)
		return
	}

	// 调用合约方法
	//_, err = instance.AsyncApplyCertificate(func(receipt *types.Receipt, err error) {
	//	if err != nil {
	//		log.Printf("Transaction failed: %v", err)
	//		atomic.AddInt32(&failureCount, 1)
	//	} else if receipt.GetStatus() == 0 {
	//		atomic.AddInt32(&successCount, 1)
	//	} else {
	//		atomic.AddInt32(&failureCount, 1)
	//	}
	//}, auth, "www.t0004.com", "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi", common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e"))
	//if err != nil {
	//	log.Printf("Failed to send transaction: %v", err)
	//	atomic.AddInt32(&failureCount, 1)
	//	return
	//}
	//================================================================
	_, _, _, err = instance.VerifyCertificate(client.GetCallOpts(), "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi")
	if err != nil {
		atomic.AddInt32(&failureCount, 1)
	} else {
		atomic.AddInt32(&successCount, 1)
	}

	// 更新已发送交易数
	atomic.AddInt32(&sentTxCount, 1)
	if sentTxCount%100 == 0 {
		fmt.Printf("Sent %d transactions\n", sentTxCount)
	}
}
