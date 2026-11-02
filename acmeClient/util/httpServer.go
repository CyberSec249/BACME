package util

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// CreateFileWithDirs 创建可能缺少的路径
func CreateFileWithDirs(filePath string, token string) error {
	// 确保文件路径的目录存在
	dir, _ := filepath.Split(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			log.Println(err)
			return
		}
	}(file)
	_, err = file.WriteString(token)
	if err != nil {
		return err
	}
	return nil
}

// WriteTokenToNginxWeb 部署token到Nginx上
func WriteTokenToNginxWeb(token string) {
	err := CreateFileWithDirs(ControlValidationFilePath, token)
	if err != nil {
		log.Println(err)
		return
	}
}

// WriteTokenToBuiltinWeb 部署token到Nginx上
func WriteTokenToBuiltinWeb(token string) {
	err := CreateFileWithDirs(ControlValidationBuiltinPath, token)
	if err != nil {
		log.Println(err)
		return
	}
}

// StartFileServer 启动文件服务器
func StartFileServer() {
	// 启动文件服务器
	http.Handle("/", http.FileServer(http.Dir(FileServerDirectory)))
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Println("启动服务器失败:", err)
	}
}
