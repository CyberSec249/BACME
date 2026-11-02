package util

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var srv = &http.Server{Addr: ":80"}

// StartServer 启动HTTP服务器的函数
func StartServer(token string, categories int) {
	if categories == 0 {
		http.HandleFunc("/well-known/pki-validation", func(w http.ResponseWriter, r *http.Request) {
			_, err := fmt.Fprintln(w, token)
			if err != nil {
				log.Fatal(err)
			}
		})
	}

	log.Println("Starting server on :80")
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("ListenAndServe(): %s", err)
	}
}

// ShutDownServer shutdown build-in http server
func ShutDownServer() {
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Println("http server exited properly")
}

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
			log.Fatal(err)
		}
	}(file)
	_, err = file.WriteString(token)
	if err != nil {
		return err
	}
	return nil
}

// WriteTokenToNginxWeb deploy the token to Nginx default web root path
func WriteTokenToNginxWeb(token string, categories int) {
	if categories == 0 {
		err := CreateFileWithDirs(NginxApplyCertificateFilePath, token)
		if err != nil {
			log.Fatal(err)
		}
	}
}
