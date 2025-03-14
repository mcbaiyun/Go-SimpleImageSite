package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	// 创建files目录
	if err := os.MkdirAll("files", 0755); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/yunimage", uploadPage)
	http.HandleFunc("/yunupload-api", uploadHandler)
	http.HandleFunc("/", imageHandler)

	fmt.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// 上传页面
func uploadPage(w http.ResponseWriter, r *http.Request) {
	// 使用模板引擎渲染HTML
	tmpl := template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
    <title>简易图床</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
</head>
<body>
    <div class="container mt-5">
        <h1 class="text-center">上传图片</h1>
        <form action="/yunupload-api" method="POST" enctype="multipart/form-data" class="mt-4">
            <div class="mb-3">
                <input type="file" name="file" class="form-control" accept="image/*">
            </div>
            <button type="submit" class="btn btn-primary">上传</button>
        </form>
    </div>
</body>
</html>
`))
	tmpl.Execute(w, nil)
}

// 上传处理
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "请选择图片文件", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := path.Ext(handler.Filename)
	if ext == "" || !isImage(ext) {
		http.Error(w, "仅支持图片格式", http.StatusBadRequest)
		return
	}

	// 生成时间戳文件名（精确到毫秒）
	ts := fmt.Sprintf("%d", time.Now().UnixMilli())
	filename := fmt.Sprintf("%s%s", ts, ext)
	dstPath := "files/" + filename

	// 保存文件
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "保存失败", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "保存失败", http.StatusInternalServerError)
		return
	}

	// 重定向到图片查看页面
	http.Redirect(w, r, fmt.Sprintf("/%s", filename), http.StatusSeeOther)
}

// 图片查看处理
func imageHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/"):]
	if filename == "" {
		http.NotFound(w, r)
		return
	}

	filePath := filepath.Join("files", filename)
	file, err := os.Open(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	http.ServeContent(w, r, filename, time.Now(), file)
}

// 检查是否为图片格式
func isImage(ext string) bool {
	ext = strings.ToLower(ext)
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp"
}
