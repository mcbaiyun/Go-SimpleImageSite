package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	// 扩展后的图片扩展名列表
	allowedExts = ".jpg|.jpeg|.png|.gif|.webp|.tiff|.tif|.bmp|.ico|.svg|.heic|.heif|.jfif|.pjpeg|.pjpg|.avif|.svgz|.ico|.cur|.xbm|.webp|.psd|.ai|.eps"
)

func main() {
	http.HandleFunc("/", handleRequest)
	fmt.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	currentDir, err := os.Getwd()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	path := r.URL.Path
	cleanedPath := filepath.Clean(path)

	// 确保路径在当前目录内
	fullPath := filepath.Join(currentDir, cleanedPath)
	if !strings.HasPrefix(fullPath, currentDir) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 禁止访问子目录
	if strings.Count(cleanedPath, "/") > 1 {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 处理根路径请求
	if cleanedPath == "/" {
		handle404(w, currentDir)
		return
	}

	// 禁止直接访问404.html
	if filepath.Base(cleanedPath) == "404.html" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(fullPath); err == nil {
		info, _ := os.Stat(fullPath)
		if info.IsDir() {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// 检查文件扩展名是否为图片类型
		ext := strings.ToLower(filepath.Ext(cleanedPath))
		if !isAllowedExtension(ext) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		http.ServeFile(w, r, fullPath)
		return
	}

	// 文件不存在
	handle404(w, currentDir)
}

func handle404(w http.ResponseWriter, currentDir string) {
	notFoundPath := filepath.Join(currentDir, "404.html")
	notFoundFile, err := os.Open(notFoundPath)
	if err != nil {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}
	defer notFoundFile.Close()

	data, err := io.ReadAll(notFoundFile)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNotFound)
	w.Write(data)
}

// 判断扩展名是否为允许的图片类型
func isAllowedExtension(ext string) bool {
	extList := strings.Split(allowedExts, "|")
	for _, e := range extList {
		if ext == e {
			return true
		}
	}
	return false
}
