package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const (
	port      = "8080"
	uploadDir = "D:\\data\\share" // 这里是你想要浏览和下载文件的目录
)

func main() {
	// 确保上传目录存在
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err = os.MkdirAll(uploadDir, os.ModePerm)
		if err != nil {
			log.Fatalf("无法创建上传目录: %v", err)
		}
	}

	http.HandleFunc("/", fileHandler)

	log.Printf("文件服务器正在运行，访问地址: http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("无法启动服务器: %v", err)
	}
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.ServeFile(w, r, filepath.Join(uploadDir, r.URL.Path))
	default:
		http.Error(w, "仅支持 GET 方法", http.StatusMethodNotAllowed)
	}
}
