package main

import (
	"baseline2/util"
	"fmt"
	"os"
	"runtime"
	"time"
)

func main() {
	//util.DeployDCVContract()
	//_ = client
	//client := util.Init()
	//util.Baseline2ApplyACertificate(client)
	//util.Baseline2UpdateCertificate(client)
	//util.Baseline2RevokeCertificate(client)
	//if util.VerifyCertificate(client) {
	//	fmt.Println("Certificate verification passed!")
	//} else {
	//	fmt.Println("Certificate verification failed!")
	//}
	f1, _ := os.OpenFile("./output/V.time", os.O_APPEND|os.O_CREATE, 0666)
	//f2, _ := os.OpenFile("./output/U1.timeR", os.O_APPEND|os.O_CREATE, 0666)
	f3, _ := os.OpenFile("./output/V.mem", os.O_APPEND|os.O_CREATE, 0666)
	//f4, _ := os.OpenFile("./output/AR3.memR", os.O_APPEND|os.O_CREATE, 0666)
	defer f1.Close()
	//defer f2.Close()
	defer f3.Close()
	//defer f4.Close()
	var totalTime time.Duration
	var totalMem uint64 = 0

	//var totalTime2 time.Duration
	//var totalMem2 uint64 = 0
	//domain := flag.String("domain", "World", "a name to say hello to")
	//flag.Parse()
	round := 1
	for i := 0; i < round; i++ {
		runtime.GC()
		startMem := new(runtime.MemStats)
		runtime.ReadMemStats(startMem)
		client := util.Init()
		startTime := time.Now()
		// Operation 1
		//util.Baseline2ApplyACertificate(client)
		//util.Baseline2UpdateCertificate(client)
		//util.Baseline2RevokeCertificate(client)
		util.Baseline2VerifyCertificate(client)
		endMem := new(runtime.MemStats)
		runtime.ReadMemStats(endMem)
		endTime := time.Now()
		spentTime := endTime.Sub(startTime)
		totalTime += spentTime
		totalMem += endMem.TotalAlloc - startMem.TotalAlloc
		//time1 = append(time1, spentTime)
		//time.Sleep(time.Second * 10)

		//runtime.GC()
		//startMem2 := new(runtime.MemStats)
		//runtime.ReadMemStats(startMem2)
		//client = util.Init()
		//startTime2 := time.Now()
		//// Operation 2
		//util.RevokeACertificate(client)
		//endMem2 := new(runtime.MemStats)
		//runtime.ReadMemStats(endMem2)
		//endTime2 := time.Now()
		//spentTime2 := endTime2.Sub(startTime2)
		//totalTime2 += spentTime2
		//totalMem2 += endMem2.TotalAlloc - startMem2.TotalAlloc
	}
	_, err := f1.WriteString(fmt.Sprintf("%v\n", totalTime.Seconds()/1))
	if err != nil {
		return
	}
	//_, err = f2.WriteString(fmt.Sprintf("%v\n", totalTime2.Seconds()/1))
	//if err != nil {
	//	return
	//}
	_, err = f3.WriteString(fmt.Sprintf("%.6v\n", bToMb(totalMem/1)))
	if err != nil {
		return
	}
	//_, err = f4.WriteString(fmt.Sprintf("%.6v\n", bToMb(totalMem2/1)))
	//if err != nil {
	//	return
	//}
}

// bToMb converts bytes to megabytes
func bToMb(b uint64) float64 {
	return float64(b) / 1024 / 1024
}
