package util

import (
	"database/sql"
	"fmt"
	"log"
	_ "modernc.org/sqlite" // 导入纯 Go 的 SQLite 驱动
	"os"
)

type IDControlRight struct {
	ID        string `json:"id"`
	ValidTime string `json:"validTime"`
}

type CertInfo struct {
	CertHash  string `json:"certHash"`
	IDs       string `json:"IDs"`
	PKHash    string `json:"PKHash"`
	ValidTime string `json:"validTime"`
}

type RevokedCertInfo struct {
	CertHash   string `json:"certHash"`
	IDs        string `json:"IDs"`
	RevokeTime string `json:"revokeTime"`
}

type VerifiedCertInfo struct {
	CertHash       string `json:"certHash"`
	IDs            string `json:"IDs"`
	VerifiedTime   string `json:"verifiedTime"`
	VerifiedResult string `json:"verifiedResult"`
}

func InitDB() *sql.DB {
	// 数据库文件路径
	dbPath := "./acme.db"
	// 判断数据库文件是否存在
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Println("数据库不存在.创建一个新的...")
		// 创建数据库文件
		file, err := os.Create(dbPath)
		if err != nil {
			log.Fatal(err)
		}
		file.Close()
	}
	// 打开数据库连接
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	// 创建表
	createTable1(db)
	createTable2(db)
	createTable3(db)
	createTable4(db)
	return db
}

// 创建表
func createTable1(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS idControl (
		id TEXT PRIMARY KEY,
		validTime TEXT NOT NULL
	);`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("表1创建成功或已存在")
}

func createTable2(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS certInfo (
		certHash TEXT PRIMARY KEY,
		IDs TEXT NOT NULL,
		PKHash TEXT NOT NULL,
		validTime TEXT NOT NULL
	);`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("表2创建成功或已存在")
}

func createTable3(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS revokedCertInfo (
		certHash TEXT PRIMARY KEY,
		IDs TEXT NOT NULL,
		revokeTime TEXT NOT NULL
	);`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("表3创建成功或已存在")
}

func createTable4(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS verifiedCertInfo (
		certHash TEXT PRIMARY KEY,
		IDs TEXT NOT NULL,
		verifiedTime TEXT NOT NULL,
		verifiedResult TEXT NOT NULL
	);`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("表4创建成功或已存在")
}

// InsertIDControlRight 插入新控制权记录
func InsertIDControlRight(db *sql.DB, idControlRight IDControlRight) (int64, error) {
	query := "INSERT OR REPLACE INTO idControl (id, validTime) VALUES (?, ?)"
	result, err := db.Exec(query, idControlRight.ID, idControlRight.ValidTime)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetIDControlRight 查询所有控制权记录
func GetIDControlRight(db *sql.DB) ([]IDControlRight, error) {
	query := "SELECT id, validTime FROM idControl"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var idControlRights []IDControlRight
	for rows.Next() {
		var idControlRight IDControlRight
		err := rows.Scan(&idControlRight.ID, &idControlRight.ValidTime)
		if err != nil {
			return nil, err
		}
		idControlRights = append(idControlRights, idControlRight)
	}
	return idControlRights, nil
}

// DeleteIDControlRight 删除控制权记录
func DeleteIDControlRight(db *sql.DB, id string) error {
	query := "DELETE FROM idControl WHERE id = ?"
	_, err := db.Exec(query, id)
	return err
}

// GetIDControlRightCount 查询控制权数量
func GetIDControlRightCount(db *sql.DB) (int, error) {
	query := "SELECT COUNT(*) FROM idControl"
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// InsertCertInfo 插入新证书记录
func InsertCertInfo(db *sql.DB, certInfo CertInfo) (int64, error) {
	query := "INSERT OR REPLACE INTO certInfo (certHash, IDs, PKHash, validTime) VALUES (?, ?, ?, ?)"
	result, err := db.Exec(query, certInfo.CertHash, certInfo.IDs, certInfo.PKHash, certInfo.ValidTime)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetCertInfo 查询所有证书记录
func GetCertInfo(db *sql.DB) ([]CertInfo, error) {
	query := "SELECT certHash, IDs, PKHash, validTime FROM certInfo"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certInfos []CertInfo
	for rows.Next() {
		var certInfo CertInfo
		err := rows.Scan(&certInfo.CertHash, &certInfo.IDs, &certInfo.PKHash, &certInfo.ValidTime)
		if err != nil {
			return nil, err
		}
		certInfos = append(certInfos, certInfo)
	}
	return certInfos, nil
}

// DeleteCertInfo 删除证书记录
func DeleteCertInfo(db *sql.DB, certHash string) error {
	query := "DELETE FROM certInfo WHERE certHash = ?"
	_, err := db.Exec(query, certHash)
	return err
}

// GetCertInfoCount 查询控制权数量
func GetCertInfoCount(db *sql.DB) (int, error) {
	query := "SELECT COUNT(*) FROM certInfo"
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// InsertRevokedCertInfo 插入已紧急撤销证书记录
func InsertRevokedCertInfo(db *sql.DB, certInfo RevokedCertInfo) (int64, error) {
	query := "INSERT OR REPLACE INTO revokedCertInfo (certHash, IDs, revokeTime) VALUES (?, ?, ?)"
	result, err := db.Exec(query, certInfo.CertHash, certInfo.IDs, certInfo.RevokeTime)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetRevokedCertInfo 查询已紧急撤销证书记录
func GetRevokedCertInfo(db *sql.DB) ([]RevokedCertInfo, error) {
	query := "SELECT certHash, IDs, revokeTime FROM revokedCertInfo"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certInfos []RevokedCertInfo
	for rows.Next() {
		var certInfo RevokedCertInfo
		err := rows.Scan(&certInfo.CertHash, &certInfo.IDs, &certInfo.RevokeTime)
		if err != nil {
			return nil, err
		}
		certInfos = append(certInfos, certInfo)
	}
	return certInfos, nil
}

// GetRevokedCertInfoCount 查询已紧急撤销证书数量
func GetRevokedCertInfoCount(db *sql.DB) (int, error) {
	query := "SELECT COUNT(*) FROM revokedCertInfo"
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// InsertVerifiedCertInfo 插入验证过的证书历史记录
func InsertVerifiedCertInfo(db *sql.DB, certInfo VerifiedCertInfo) (int64, error) {
	query := "INSERT OR REPLACE INTO verifiedCertInfo (certHash, IDs, VerifiedTime, VerifiedResult) VALUES (?, ?, ?, ?)"
	result, err := db.Exec(query, certInfo.CertHash, certInfo.IDs, certInfo.VerifiedTime, certInfo.VerifiedResult)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetVerifiedCertInfo 查询验证过的证书历史记录
func GetVerifiedCertInfo(db *sql.DB) ([]VerifiedCertInfo, error) {
	query := "SELECT certHash, IDs, verifiedTime, verifiedResult FROM verifiedCertInfo"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certInfos []VerifiedCertInfo
	for rows.Next() {
		var certInfo VerifiedCertInfo
		err := rows.Scan(&certInfo.CertHash, &certInfo.IDs, &certInfo.VerifiedTime, &certInfo.VerifiedResult)
		if err != nil {
			return nil, err
		}
		certInfos = append(certInfos, certInfo)
	}
	return certInfos, nil
}

// GetVerifiedCertInfoCount 查询验证的证书次数
func GetVerifiedCertInfoCount(db *sql.DB) (int, error) {
	query := "SELECT COUNT(*) FROM verifiedCertInfo"
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
