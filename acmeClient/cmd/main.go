package main

import (
	"acmeClient/util"
	"crypto/rsa"
	"encoding/pem"
	"fmt"
	"github.com/A1andNS/newCrypto/ecdsa"
	"github.com/A1andNS/newCrypto/x509"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ConfigInfo struct {
	BlockchainNode         string `json:"blockchainNode"`
	BlockchainGroup        int    `json:"blockchainGroup"`
	BlockchainID           int64  `json:"blockchainID"`
	BlockchainWallet       string `json:"blockchainWallet"`
	SmartContractAddress   string `json:"smartContractAddress"`
	ControlValidationTimes int    `json:"controlValidationTimes"`
	CertNum                int    `json:"certNum"`
	RevokedCertNum         int    `json:"revokedCertNum"`
	VerifiedCertNum        int    `json:"verifiedCertNum"`
}

type ControlValidationReq struct {
	IDs string `json:"ids"`
}

type ApplyCertificateReq struct {
	IDs string `json:"ids"`
}

type UpdateCertificateReq struct {
	IDs      string `json:"ids"`
	CertHash string `json:"certHash"`
}

type RevokeCertificateReq struct {
	CertHash string `json:"certHash"`
}

type CertInfo struct {
	Version   string `json:"version"`
	Serial    string `json:"serial"`
	SignAlg   string `json:"signAlg"`
	Subject   string `json:"subject"`
	Issuer    string `json:"issuer"`
	NotBefore string `json:"notBefore"`
	NotAfter  string `json:"notAfter"`
	PK        string `json:"pk"`
	San       string `json:"san"`
	CertType  string `json:"certType"`
	UsageType string `json:"usageType"`
	Status    string `json:"status"`
}

var client *util.ACMEClient

// 启动服务器
func main() {
	client = util.Init(true)
	e := echo.New()

	// 启用 CORS 中间件
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},                                                                // 允许所有域名访问
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}, // 允许的 HTTP 方法
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},       // 允许的请求头
	}))

	// 定义路由
	e.GET("/api/configInfo", getConfigInfo)                                    // 获取配置信息
	e.GET("/api/idControlRightInfo", getControlValidationInfo)                 // 获取控制权信息
	e.GET("/api/getCertInfo", getCertInfo)                                     // 获取证书信息
	e.GET("/api/getRevokedCertInfo", getRevokedCertInfo)                       // 获取已紧急撤销证书信息
	e.GET("/api/getVerifiedCertInfo", getVerifiedCertInfo)                     // 获取验证证书历史记录
	e.POST("/api/idControlRight", startControlValidation)                      // 开始控制权验证
	e.POST("/api/applyCertificate", startApplyCertificate)                     // 开始证书申请
	e.POST("/api/updateCertificate", startUpdateCertificate)                   // 开始证书更新
	e.POST("/api/revokeCertificate", startRevokeCertificate)                   // 开始证书撤销
	e.POST("/api/revokeCertificateEmergency", startRevokeCertificateEmergency) // 开始证书撤销
	e.POST("/api/verify", verifyCertificate)                                   // 开始证书撤销certInfo

	// 启动服务器
	e.Logger.Fatal(e.Start(":8080"))
}

// 获取配置信息
func getConfigInfo(c echo.Context) error {
	controlValidationTimes, err := util.GetIDControlRightCount(client.DB)
	if err != nil {
		return err
	}
	certNum, err := util.GetCertInfoCount(client.DB)
	if err != nil {
		return err
	}
	revokedCertNum, err := util.GetRevokedCertInfoCount(client.DB)
	if err != nil {
		return err
	}
	verifiedNum, err2 := util.GetVerifiedCertInfoCount(client.DB)
	if err2 != nil {
		return err2
	}
	configInfo := ConfigInfo{
		BlockchainNode:         client.Config.NodeURL,
		BlockchainGroup:        client.Config.GroupID,
		BlockchainID:           client.Config.ChainID,
		BlockchainWallet:       client.Account.String(),
		SmartContractAddress:   client.ContractAddress.String(),
		ControlValidationTimes: controlValidationTimes,
		CertNum:                certNum,
		RevokedCertNum:         revokedCertNum,
		VerifiedCertNum:        verifiedNum,
	}
	// 返回 JSON 响应
	return c.JSON(http.StatusOK, configInfo)
}

// 获取控制权信息
func getControlValidationInfo(c echo.Context) error {
	var err error
	var controlValidationInfo []util.IDControlRight
	controlValidationInfo, err = util.GetIDControlRight(client.DB)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, controlValidationInfo)
}

// 获取证书信息
func getCertInfo(c echo.Context) error {
	var err error
	var certInfo []util.CertInfo
	certInfo, err = util.GetCertInfo(client.DB)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, certInfo)
}

// 获取已紧急撤销的证书信息
func getRevokedCertInfo(c echo.Context) error {
	var err error
	var revokedCertInfo []util.RevokedCertInfo
	revokedCertInfo, err = util.GetRevokedCertInfo(client.DB)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return c.JSON(http.StatusOK, revokedCertInfo)
}

// getVerifiedCertInfo 获取证书验证历史记录
func getVerifiedCertInfo(c echo.Context) error {
	var err error
	var revokedCertInfo []util.VerifiedCertInfo
	revokedCertInfo, err = util.GetVerifiedCertInfo(client.DB)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return c.JSON(http.StatusOK, revokedCertInfo)
}

// 开始控制权验证
func startControlValidation(c echo.Context) error {
	var req ControlValidationReq
	validTime := time.Now().AddDate(0, 0, 30).Format("2006-01-02 15:04:05")
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "无效请求体",
		})
	}
	// 打印接收到的数据
	fmt.Printf("接收的控制权验证请求: %+v\n", req)
	ok := util.ControlValidation(client, req.IDs)
	// 控制权验证成功就记录到本地数据库中
	if ok {
		idList := strings.Split(req.IDs, ",")
		for i := 0; i < len(idList); i++ {
			idControlRight := util.IDControlRight{ID: idList[i], ValidTime: validTime}
			_, err := util.InsertIDControlRight(client.DB, idControlRight)
			if err != nil {
				return err
			}
		}
		return c.JSON(http.StatusOK, map[string]string{
			"message": "控制权验证成功",
		})
	} else {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "控制权验证失败",
		})
	}
}

// 证书申请
func startApplyCertificate(c echo.Context) error {
	var req ApplyCertificateReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "无效的请求体",
		})
	}
	// 打印接收到的数据
	fmt.Printf("接收到的证书申请请求: %+v\n", req)
	ok, validTime := util.ApplyCertificate(client, req.IDs)
	// 申请成功在本地数据库记录
	if ok {
		certInfo := util.CertInfo{CertHash: client.IPFSCid, IDs: req.IDs, PKHash: client.CertificatePublicKeyHash.String(), ValidTime: validTime}
		_, err := util.InsertCertInfo(client.DB, certInfo)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"message": err.Error(),
			})
		}
		return c.JSON(http.StatusOK, map[string]string{
			"message": "证书申请成功",
		})
	} else {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "证书申请失败",
		})
	}
}

// 证书更新路由处理
func startUpdateCertificate(c echo.Context) error {
	var req UpdateCertificateReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "无效请求体",
		})
	}
	// 打印接收到的数据
	fmt.Printf("接收到证书更新请求: %+v\n", req)
	ok, validTime := util.UpdateCertificate(client, req.CertHash, req.IDs)
	// 更新成功在本地数据库记录
	if ok {
		certInfo := util.CertInfo{CertHash: client.IPFSCid, IDs: req.IDs, PKHash: client.CertificatePublicKeyHash.String(), ValidTime: validTime}
		// 删除旧的证书
		err := util.DeleteCertInfo(client.DB, req.CertHash)
		if err != nil {
			fmt.Println(err)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"message": err.Error(),
			})
		}
		// 插入新记录
		_, err = util.InsertCertInfo(client.DB, certInfo)
		if err != nil {
			fmt.Println(err)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"message": err.Error(),
			})
		}
		return c.JSON(http.StatusOK, map[string]string{
			"message": "证书更新成功",
		})
	} else {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "证书更新失败",
		})
	}
}

// 证书撤销路由处理
func startRevokeCertificate(c echo.Context) error {
	var req RevokeCertificateReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "无效请求体",
		})
	}
	// 打印接收到的数据
	fmt.Printf("接收到的证书撤销请求: %+v\n", req)
	ok := util.RevokeCertificate(client, req.CertHash)
	// 撤销成功在本地数据库删除
	if ok {
		// 删除旧的证书
		err := util.DeleteCertInfo(client.DB, req.CertHash)
		if err != nil {
			fmt.Println(err)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"message": err.Error(),
			})
		}
		return c.JSON(http.StatusOK, map[string]string{
			"message": "证书撤销成功",
		})
	} else {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "证书撤销失败",
		})
	}
}

// 基于控制权或基于私钥验证撤销证书
func startRevokeCertificateEmergency(c echo.Context) error {
	revokeType := c.QueryParam("type")
	fmt.Println("受到请求", revokeType)
	//fmt.Println(c.Request())
	if revokeType == "0" {
		// 获取文件1
		file1, err := c.FormFile("file1")
		if err != nil {
			fmt.Println(err)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"message": "待撤销证书文件未上传",
			})
		}

		// 获取文件2
		file2, err := c.FormFile("file2")
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"message": "待撤销证书私钥未上传",
			})
		}

		// 保存文件1
		file1Path := filepath.Join("uploads", file1.Filename)
		if err := saveFile(file1, file1Path); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"message": fmt.Sprintf("文件1保存失败: %v", err),
			})
		}

		// 保存文件2
		file2Path := filepath.Join("uploads", file2.Filename)
		if err := saveFile(file2, file2Path); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"message": fmt.Sprintf("文件2保存失败: %v", err),
			})
		}
		ids, certHash := util.ParserOldCertificate(file1Path)
		// 读取私钥和旧证书，计算证书CID和生成私钥签名证明
		oldCertPrivateKey := util.ReadOldPrivateKey(file2Path)
		ok := util.RevokeCertificateWithPrivateKey(client, certHash, oldCertPrivateKey)
		revokeTime := time.Now().Format("2006-01-02 15:04:05")
		if ok {
			_ = os.Remove(file1Path)
			_ = os.Remove(file2Path)
			revokedCertInfo := util.RevokedCertInfo{CertHash: certHash, IDs: ids, RevokeTime: revokeTime}
			_, err = util.InsertRevokedCertInfo(client.DB, revokedCertInfo)
			if err != nil {
				return err
			}
			return c.JSON(http.StatusOK, map[string]string{
				"message": "基于私钥验证的证书撤销成功",
			})
		} else {
			_ = os.Remove(file1Path)
			_ = os.Remove(file2Path)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"message": "基于私钥验证的证书撤销失败",
			})
		}
	} else if revokeType == "1" {
		// 获取文件1
		file1, err := c.FormFile("file1")
		if err != nil {
			fmt.Println(err)
			return c.JSON(http.StatusBadRequest, map[string]string{
				"message": "待撤销证书文件未上传",
			})
		}
		// 保存文件1
		file1Path := filepath.Join("uploads", file1.Filename)
		if err := saveFile(file1, file1Path); err != nil {
			fmt.Println(err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"message": fmt.Sprintf("文件1保存失败: %v", err),
			})
		}
		ids, certHash := util.ParserOldCertificate(file1Path)
		ok := util.RevokeCertificateWithDV(client, certHash)
		revokeTime := time.Now().Format("2006-01-02 15:04:05")
		if ok {
			_ = os.Remove(file1Path)
			revokedCertInfo := util.RevokedCertInfo{CertHash: certHash, IDs: ids, RevokeTime: revokeTime}
			_, err = util.InsertRevokedCertInfo(client.DB, revokedCertInfo)
			if err != nil {
				fmt.Println(err)
				return err
			}
			return c.JSON(http.StatusOK, map[string]string{
				"message": "基于控制权验证的证书撤销成功",
			})
		} else {
			_ = os.Remove(file1Path)
			return c.JSON(http.StatusOK, map[string]string{
				"message": "基于控制权验证的证书撤销失败",
			})
		}
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "请求参数非法",
		})
	}
}

// verifyCertificate 验证证书
func verifyCertificate(c echo.Context) error {
	// 获取文件
	file, err := c.FormFile("file")
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "待撤销证书文件未上传",
		})
	}
	// 保存文件
	filePath := filepath.Join("uploads", file.Filename)
	if err := saveFile(file, filePath); err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": fmt.Sprintf("文件1保存失败: %v", err),
		})
	}
	ok, certificate, tip := util.VerifyCertificate(client, filePath)
	var publicKeyBytes []byte
	if certificate.PublicKeyAlgorithm == 1 {
		publicKeyBytes, err = x509.MarshalPKIXPublicKey(certificate.PublicKey.(*rsa.PublicKey))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"message": fmt.Sprintf("公钥解析失败: %v", err),
			})
		}
	} else if certificate.PublicKeyAlgorithm == 3 {
		publicKeyBytes, err = x509.MarshalPKIXPublicKey(certificate.PublicKey.(*ecdsa.PublicKey))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"message": fmt.Sprintf("公钥解析失败: %v", err),
			})
		}
	}

	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	pemString := string(pem.EncodeToMemory(pemBlock))
	ids := strings.Join(certificate.DNSNames, ",")
	for i := 0; i < len(certificate.IPAddresses); i++ {
		ids += "," + certificate.IPAddresses[i].String()
	}
	if ids == "" {
		ids = certificate.Subject.CommonName
	}
	var cerType string
	if util.IsDPKICertificate(certificate) {
		cerType = "dpki-acme"
	} else {
		cerType = "CA签名证书"
	}
	certInfo := CertInfo{
		Version:   strconv.Itoa(certificate.Version),
		Serial:    certificate.SerialNumber.String(),
		SignAlg:   certificate.SignatureAlgorithm.String(),
		Subject:   certificate.Subject.String(),
		Issuer:    certificate.Issuer.String(),
		NotBefore: certificate.NotBefore.Format("2006-01-02 15:04:05"),
		NotAfter:  certificate.NotAfter.Format("2006-01-02 15:04:05"),
		PK:        pemString,
		San:       ids,
		CertType:  cerType,
		Status:    "",
	}
	CertCID := util.CalculateIPFSCid(filePath)

	if ok {
		_ = os.Remove(filePath)
		verifiedCertInfo := util.VerifiedCertInfo{
			CertHash:       CertCID,
			IDs:            ids,
			VerifiedTime:   time.Now().Format("2006-01-02 15:04:05"),
			VerifiedResult: "有效",
		}
		_, err = util.InsertVerifiedCertInfo(client.DB, verifiedCertInfo)
		if err != nil {
			return err
		}
		certInfo.Status = "有效"
		return c.JSON(http.StatusOK, certInfo)
	} else {
		_ = os.Remove(filePath)
		verifiedCertInfo := util.VerifiedCertInfo{
			CertHash:       CertCID,
			IDs:            ids,
			VerifiedTime:   time.Now().Format("2006-01-02 15:04:05"),
			VerifiedResult: tip,
		}
		_, err = util.InsertVerifiedCertInfo(client.DB, verifiedCertInfo)
		if err != nil {
			return err
		}
		certInfo.Status = tip
		return c.JSON(http.StatusOK, certInfo)
	}
}

// 保存文件到指定路径
func saveFile(file *multipart.FileHeader, dst string) error {
	// 打开上传的文件
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// 创建目标文件
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	// 将上传的文件内容复制到目标文件
	if _, err := io.Copy(out, src); err != nil {
		return err
	}

	return nil
}
