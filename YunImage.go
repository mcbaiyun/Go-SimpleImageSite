package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pquerna/otp/totp" // 引入TOTP库
)

const (
	// 扩展后的图片扩展名列表
	allowedExts = ".jpg|.jpeg|.png|.gif|.webp|.tiff|.tif|.bmp|.ico|.svg|.heic|.heif|.jfif|.pjpeg|.pjpg|.avif|.svgz|.ico|.cur|.xbm|.webp|.psd|.ai|.eps"
	totpKeyFile = "totp.key"
)

func main() {
	http.HandleFunc("/", handleRequest)
	http.HandleFunc("/setup-totp", setupTOTP)
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

	// 检查totp.key文件是否存在
	if !fileExists(filepath.Join(currentDir, totpKeyFile)) {
		http.Redirect(w, r, "/setup-totp", http.StatusSeeOther)
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

	// 处理根路径请求
	if cleanedPath == "\\" {
		if r.Method == "GET" {
			// 显示上传页面
			displayUploadPage(w)
		} else if r.Method == "POST" {
			// 处理文件上传
			imgDir := filepath.Join(currentDir, "IMG")
			os.MkdirAll(imgDir, os.ModePerm) // 确保IMG目录存在

			// 获取TOTP密码
			totpCode := r.FormValue("totp")
			if totpCode == "" {
				http.Error(w, "TOTP code is required", http.StatusBadRequest)
				return
			}

			// 读取TOTP密钥
			key, err := readTOTPKey(filepath.Join(currentDir, totpKeyFile))
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// 验证TOTP密码
			valid := totp.Validate(totpCode, key)
			if !valid {
				http.Error(w, "Invalid TOTP code", http.StatusUnauthorized)
				return
			}

			handleFileUpload(w, r, imgDir)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// 禁止访问子目录
	if strings.Count(cleanedPath, "/") > 1 {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 禁止直接访问404.html
	if filepath.Base(cleanedPath) == "404.html" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 检查文件是否存在
	imgDir := filepath.Join(currentDir, "IMG")
	fullPath = filepath.Join(imgDir, cleanedPath)
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

// 显示上传页面
func displayUploadPage(w http.ResponseWriter) {
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Upload Image</title>
    <link href="https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/4.5.2/css/bootstrap.min.css" rel="stylesheet">
    <style>
        body {
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background-color: #f8f9fa;
        }
        .upload-container {
            text-align: center;
            background-color: #fff;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
            width: 90%;
            max-width: 500px;
        }
        .custom-file-input {
            cursor: pointer;
        }
        .custom-file-label {
            cursor: pointer;
        }
        .custom-file-label::after {
            content: "Browse";
            background-color: #007bff;
            color: white;
            border-radius: 0 0.25rem 0.25rem 0;
        }
        .custom-file-label:hover::after {
            background-color: #0056b3;
        }
        /* 媒体查询，适应手机端 */
        @media (max-width: 768px) {
            .upload-container {
                padding: 15px;
            }
            .custom-file-label::after {
                font-size: 14px;
            }
        }
    </style>
</head>
<body>
    <div class="upload-container">
        <h2>Upload Image</h2>
        <form id="uploadForm" enctype="multipart/form-data" method="post">
            <div class="custom-file mb-3">
                <input type="file" class="custom-file-input" id="imageFile" name="imageFile" accept=".jpg,.jpeg,.png,.gif,.webp,.tiff,.tif,.bmp,.ico,.svg,.heic,.heif,.jfif,.pjpeg,.pjpg,.avif,.svgz,.ico,.cur,.xbm,.webp,.psd,.ai,.eps">
                <label class="custom-file-label" for="imageFile">Choose file</label>
            </div>
            <div class="form-group">
                <img id="preview" src="#" alt="Preview" style="display:none;max-width:100%;">
            </div>
            <div class="form-group">
                <input type="text" class="form-control" id="totp" name="totp" placeholder="Enter TOTP code">
            </div>
            <button type="submit" class="btn btn-primary">Upload</button>
        </form>
    </div>
    <script>
        document.getElementById('imageFile').addEventListener('change', function(event) {
            var file = event.target.files[0];
            if (file) {
                var reader = new FileReader();
                reader.onload = function(e) {
                    document.getElementById('preview').src = e.target.result;
                    document.getElementById('preview').style.display = 'block';
                };
                reader.readAsDataURL(file);
            }
        });
    </script>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// 处理文件上传
func handleFileUpload(w http.ResponseWriter, r *http.Request, currentDir string) {
	err := r.ParseMultipartForm(10 << 20) // 10 MB max memory
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("imageFile")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(handler.Filename))
	if !isAllowedExtension(ext) {
		http.Error(w, "Invalid file type", http.StatusBadRequest)
		return
	}

	// 计算文件哈希
	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		http.Error(w, "Unable to calculate file hash", http.StatusInternalServerError)
		return
	}
	fileHash := hex.EncodeToString(hash.Sum(nil))

	newFilename := fmt.Sprintf("%s%s", fileHash, ext)
	newPath := filepath.Join(currentDir, newFilename)

	// 重新打开文件，因为计算哈希后文件指针已经到了文件末尾
	file, handler, err = r.FormFile("imageFile")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	dst, err := os.Create(newPath)
	if err != nil {
		http.Error(w, "Unable to create the file for writing", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/"+newFilename, http.StatusSeeOther)
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

// 检查文件是否存在
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// 读取TOTP密钥
func readTOTPKey(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// 设置TOTP页面
func setupTOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		totpKey := r.FormValue("totpKey")
		totpCode := r.FormValue("totpCode")

		if totpKey == "" || totpCode == "" {
			http.Error(w, "TOTP key and code are required", http.StatusBadRequest)
			return
		}

		// 验证TOTP代码
		valid := totp.Validate(totpCode, totpKey)
		if !valid {
			http.Error(w, "Invalid TOTP code", http.StatusUnauthorized)
			return
		}

		// 保存TOTP密钥
		err := os.WriteFile(totpKeyFile, []byte(totpKey), 0644)
		if err != nil {
			http.Error(w, "Unable to save TOTP key", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// 生成随机TOTP密钥
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Go-SimpleImageSite",
		AccountName: "user",
	})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Setup TOTP</title>
    <link href="https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/4.5.2/css/bootstrap.min.css" rel="stylesheet">
    <script type="text/javascript" src="https://cdnjs.cloudflare.com/ajax/libs/qrcodejs/1.0.0/qrcode.min.js"></script>
    <style>
        body {
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background-color: #f8f9fa;
        }
        .setup-container {
            text-align: center;
            background-color: #fff;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
            width: 90%;
            max-width: 500px;
        }
        .qr-code>img {
			display: inline!important;
            margin: 0 auto; /* 添加此行以使二维码居中 */
            margin-bottom: 20px;
        }
        .totp-key {
            margin-bottom: 20px;
            font-weight: bold;
        }
    </style>
</head>
<body>
    <div class="setup-container">
        <h2>Setup TOTP</h2>
        <div class="qr-code" id="qrcode"></div>
        <div class="totp-key">TOTP Key: ` + key.Secret() + `</div>
        <form id="setupForm" method="post">
            <input type="hidden" id="totpKey" name="totpKey" value="` + key.Secret() + `">
            <div class="form-group">
                <input type="text" class="form-control" id="totpCode" name="totpCode" placeholder="Enter TOTP code">
            </div>
            <button type="submit" class="btn btn-primary">Submit</button>
        </form>
    </div>
    <script>
        new QRCode(document.getElementById("qrcode"), "` + key.URL() + `");
    </script>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
