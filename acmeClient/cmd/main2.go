package main

import (
	"acmeClient/util"
	"fmt"
	"time"
)

func main() {
	validTime := time.Now().AddDate(0, 0, 30).Format("2006-01-02 15:04:05")
	fmt.Println(validTime)
	util.DeployDpkiAcmeContract()
	client := util.Init(true)
	util.ControlValidation(client, "www.t0004.com")
	util.ApplyCertificate(client, "www.t0004.com")
	ids, oldCertHash := util.ParserOldCertificate("./output/cert.crt")
	util.UpdateCertificate(client, oldCertHash, ids)
	util.RevokeCertificate(client, "./output/cert.crt")
	util.RevokeCertificateWithPrivateKey(client, oldCertHash, util.ReadOldPrivateKey("./output/privateKey.pem"))
	util.RevokeCertificateWithDV(client, "./output/cert.crt")
	ok, _, _ := util.VerifyCertificate(client, "./output/cert.crt")
	if ok {
		fmt.Println("证书验证通过！")
	} else {
		fmt.Println("证书验证不通过！")
	}
	//f1, _ := os.OpenFile("./Mutil/V.time", os.O_APPEND|os.O_CREATE, 0666)
	////f2, _ := os.OpenFile("./output/U1.timeR", os.O_APPEND|os.O_CREATE, 0666)
	//f3, _ := os.OpenFile("./Mutil/V.mem", os.O_APPEND|os.O_CREATE, 0666)
	////f4, _ := os.OpenFile("./output/AR3.memR", os.O_APPEND|os.O_CREATE, 0666)
	//defer f1.Close()
	////defer f2.Close()
	//defer f3.Close()
	////defer f4.Close()
	//var totalTime time.Duration
	//var totalMem uint64 = 0
	//
	////var totalTime2 time.Duration
	////var totalMem2 uint64 = 0
	//round := 1
	//for i := 0; i < round; i++ {
	//	runtime.GC()
	//	startMem := new(runtime.MemStats)
	//	runtime.ReadMemStats(startMem)
	//	client := util.Init()
	//	startTime := time.Now()
	//	//操作1
	//	//util.ApplyACertificate(client)
	//	//util.UpdateACertificate(client)
	//	//util.RevokeACertificate(client)
	//	util.VerifyCertificate(client)
	//	endMem := new(runtime.MemStats)
	//	runtime.ReadMemStats(endMem)
	//	endTime := time.Now()
	//	spentTime := endTime.Sub(startTime)
	//	totalTime += spentTime
	//	totalMem += endMem.TotalAlloc - startMem.TotalAlloc
	//	//time1 = append(time1, spentTime)
	//	//time.Sleep(time.Second * 10)
	//
	//	//runtime.GC()
	//	//startMem2 := new(runtime.MemStats)
	//	//runtime.ReadMemStats(startMem2)
	//	//client = util.Init()
	//	//startTime2 := time.Now()
	//	//// 操作2
	//	//util.RevokeACertificate(client)
	//	//endMem2 := new(runtime.MemStats)
	//	//runtime.ReadMemStats(endMem2)
	//	//endTime2 := time.Now()
	//	//spentTime2 := endTime2.Sub(startTime2)
	//	//totalTime2 += spentTime2
	//	//totalMem2 += endMem2.TotalAlloc - startMem2.TotalAlloc
	//}
	//_, err := f1.WriteString(fmt.Sprintf("%v\n", totalTime.Seconds()/1))
	//if err != nil {
	//	return
	//}
	////_, err = f2.WriteString(fmt.Sprintf("%v\n", totalTime2.Seconds()/1))
	////if err != nil {
	////	return
	////}
	//_, err = f3.WriteString(fmt.Sprintf("%.6v\n", bToMb(totalMem/1)))
	//if err != nil {
	//	return
	//}
	////_, err = f4.WriteString(fmt.Sprintf("%.6v\n", bToMb(totalMem2/1)))
	////if err != nil {
	////	return
	//////}
}

// bToMb converts bytes to megabytes
//func bToMb(b uint64) float64 {
//	return float64(b) / 1024 / 1024
//}
