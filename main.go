package main

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/skip2/go-qrcode"
)

// 版本号 - 可以在编译时通过 ldflags 注入
// 例如: go build -ldflags="-X main.Version=1.0.1" -o go-fileserver main.go
var Version = "1.0.1"

const (
	// 要共享的目录
	shareDir = "D:/Download"
	// 服务器地址 - 监听所有接口
	serverAddr = "localhost:8080"
)

// 文件信息结构
type FileInfo struct {
	Name    string
	Path    string
	Size    string
	IsDir   bool
	ModTime string // 文件修改时间
}

// IP地址信息
type IPAddress struct {
	IP          string
	DisplayName string
}

// MIME类型映射 - 将映射表提取为全局变量
var mimeTypes = map[string]string{
	".html": "text/html",
	".htm":  "text/html",
	".txt":  "text/plain",
	".css":  "text/css",
	".js":   "application/javascript",
	".json": "application/json",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
	".pdf":  "application/pdf",
	".mp3":  "audio/mpeg",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".zip":  "application/zip",
	".rar":  "application/x-rar-compressed",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
}

// HTML模板
var htmlTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>文件服务器</title>
    <link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600&display=swap">
    <style>
        :root {
            --primary: #4f46e5;
            --primary-light: #6366f1;
            --secondary: #0ea5e9;
            --text: #1e293b;
            --text-light: #64748b;
            --background: #f8fafc;
            --surface: #ffffff;
            --border: #e2e8f0;
            --hover: #f1f5f9;
            --folder: #f59e0b;
            --file: #3b82f6;
            --success: #10b981;
            --error: #ef4444;
        }
        
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background-color: var(--background);
            color: var(--text);
            line-height: 1.6;
            padding: 0;
            margin: 0;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 1rem;
        }
        
        header {
            background-color: var(--surface);
            border-bottom: 1px solid var(--border);
            padding: 1rem 0;
            margin-bottom: 1rem;
            box-shadow: 0 1px 3px rgba(0,0,0,0.05);
        }
        
        .header-content {
            display: flex;
            align-items: center;
            justify-content: space-between;
        }
        
        h1 {
            color: var(--primary);
            font-weight: 600;
            font-size: 1.5rem;
            margin: 0;
        }
        
        .qr-container {
            position: relative;
        }
        
        .qr-code {
            width: 32px;
            height: 32px;
            background-color: var(--primary-light);
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-size: 18px;
            cursor: pointer;
        }
        
        .qr-popup {
            display: none;
            position: absolute;
            right: 0;
            top: 100%;
            margin-top: 10px;
            background: var(--surface);
            padding: 1rem;
            border-radius: 8px;
            box-shadow: 0 10px 25px rgba(0,0,0,0.1);
            z-index: 1000;
            text-align: center;
            width: 240px;
        }
        
        .qr-popup.show {
            display: block;
        }
        
        .qr-popup img {
            max-width: 180px;
            height: auto;
            margin-bottom: 10px;
            border: 1px solid var(--border);
            padding: 5px;
            border-radius: 4px;
        }
        
        .qr-popup p {
            font-size: 0.75rem;
            color: var(--text-light);
            overflow-wrap: break-word;
            word-wrap: break-word;
            margin-bottom: 10px;
        }
        
        .qr-close {
            position: absolute;
            top: 5px;
            right: 5px;
            width: 20px;
            height: 20px;
            display: flex;
            align-items: center;
            justify-content: center;
            cursor: pointer;
            color: var(--text-light);
        }
        
        .qr-close:hover {
            color: var(--primary);
        }
        
        .ip-selector {
            margin-top: 10px;
            width: 100%;
        }
        
        .ip-selector select {
            width: 100%;
            padding: 5px;
            border-radius: 4px;
            border: 1px solid var(--border);
            background-color: var(--surface);
            font-size: 0.75rem;
            color: var(--text);
        }
        
        .current-ip {
            margin-top: 5px;
            font-size: 0.75rem;
            color: var(--text-light);
        }
        
        .file-browser {
            background-color: var(--surface);
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 1px 3px rgba(0,0,0,0.05);
            border: 1px solid var(--border);
        }
        
        .back {
            padding: 0.75rem 1rem;
            border-bottom: 1px solid var(--border);
        }
        
        .back a {
            color: var(--text-light);
            text-decoration: none;
            display: flex;
            align-items: center;
            gap: 0.5rem;
            font-size: 0.9rem;
        }
        
        .back a:hover {
            color: var(--primary);
        }
        
        .back a svg {
            width: 16px;
            height: 16px;
        }
        
        table {
            width: 100%;
            border-collapse: collapse;
        }
        
        th {
            text-align: left;
            padding: 0.75rem 1rem;
            font-weight: 500;
            color: var(--text-light);
            font-size: 0.85rem;
            text-transform: uppercase;
            letter-spacing: 0.025em;
            background-color: var(--surface);
            border-bottom: 1px solid var(--border);
        }
        
        td {
            padding: 0.75rem 1rem;
            border-bottom: 1px solid var(--border);
        }
        
        tr:last-child td {
            border-bottom: none;
        }
        
        tr:hover {
            background-color: var(--hover);
        }
        
        a {
            color: var(--primary);
            text-decoration: none;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }
        
        a:hover {
            text-decoration: underline;
        }
        
        .folder {
            color: var(--folder);
            font-weight: 500;
        }
        
        .file {
            color: var(--file);
        }
        
        .size, .time {
            text-align: center;
            font-family: monospace;
            color: var(--text-light);
            font-size: 0.9rem;
            white-space: nowrap;
        }
        
        .icon {
            display: inline-flex;
            align-items: center;
            justify-content: center;
        }
        
        footer {
            text-align: center;
            color: var(--text-light);
            font-size: 0.8rem;
            margin-top: 2rem;
            padding: 1rem 0;
        }
        
        /* 上传按钮样式 */
        .upload-btn {
            background-color: var(--success);
            color: white;
            border: none;
            border-radius: 8px;
            padding: 0.5rem 1rem;
            font-size: 0.9rem;
            font-weight: 500;
            cursor: pointer;
            display: flex;
            align-items: center;
            gap: 0.5rem;
            transition: background-color 0.2s;
        }
        
        .upload-btn:hover {
            background-color: #0d9668;
        }
        
        .upload-container {
            margin-bottom: 1rem;
            border-radius: 8px;
            border: 1px dashed var(--border);
            padding: 1rem;
            background-color: var(--surface);
            transition: all 0.2s;
        }
        
        .upload-container.drag-active {
            border-color: var(--primary);
            background-color: var(--hover);
        }
        
        .upload-form {
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
        }
        
        .file-input-container {
            position: relative;
            margin-bottom: 1rem;
            text-align: center;
        }
        
        .file-input {
            opacity: 0;
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            cursor: pointer;
        }
        
        .file-input-label {
            border: 1px dashed var(--border);
            padding: 2rem 1rem;
            border-radius: 8px;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            color: var(--text-light);
            cursor: pointer;
            transition: all 0.2s;
        }
        
        .file-input-label:hover {
            border-color: var(--primary);
            background-color: var(--hover);
        }
        
        .file-list {
            margin-top: 1rem;
            font-size: 0.9rem;
            color: var(--text);
        }
        
        .file-list-item {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            margin-bottom: 0.5rem;
        }
        
        .upload-progress {
            height: 4px;
            width: 100%;
            background-color: var(--border);
            border-radius: 2px;
            overflow: hidden;
            margin-top: 1rem;
        }
        
        .upload-progress-bar {
            height: 100%;
            background-color: var(--primary);
            width: 0%;
            transition: width 0.3s;
        }
        
        .upload-status {
            margin-top: 1rem;
            font-size: 0.9rem;
        }
        
        .upload-success {
            color: var(--success);
        }
        
        .upload-error {
            color: var(--error);
        }
        
        .upload-actions {
            display: flex;
            justify-content: flex-end;
            margin-top: 1rem;
        }
        
        .header-buttons {
            display: flex;
            gap: 0.75rem;
            align-items: center;
        }
        
        @media (max-width: 640px) {
            .container {
                padding: 0.5rem;
            }
            
            th, td {
                padding: 0.5rem;
            }
            
            h1 {
                font-size: 1.25rem;
            }
            
            .time {
                display: none;
            }
            
            .upload-form {
                flex-direction: column;
            }
            
            .file-input-label {
                padding: 1rem;
            }
        }
    </style>
    <script>
        document.addEventListener('DOMContentLoaded', function() {
            // QR码交互
            const qrButton = document.querySelector('.qr-code');
            const qrPopup = document.querySelector('.qr-popup');
            const qrClose = document.querySelector('.qr-close');
            
            qrButton.addEventListener('click', function() {
                qrPopup.classList.add('show');
            });
            
            qrClose.addEventListener('click', function() {
                qrPopup.classList.remove('show');
            });
            
            // 点击外部关闭
            document.addEventListener('click', function(e) {
                if (!qrPopup.contains(e.target) && !qrButton.contains(e.target)) {
                    qrPopup.classList.remove('show');
                }
            });

            // 页面加载时刷新二维码
            refreshQRCode();
            
            // 文件上传相关脚本
            setupFileUpload();
        });
        
        // 刷新当前二维码
        function refreshQRCode() {
            const selectElement = document.querySelector('.ip-selector select');
            if (selectElement) {
                updateQRCode(selectElement);
            }
        }
        
        // 更新QR码和路径 - 只影响二维码，不影响页面链接
        function updateQRCode(selectElement) {
            const selectedIP = selectElement.value;
            const currentPath = '{{.CurrentPath}}';
            const serverPort = '{{.ServerPort}}';
            const baseURL = 'http://' + selectedIP + serverPort;
            const fullURL = baseURL + (currentPath.startsWith('/') ? currentPath : '/' + currentPath);
            
            console.log('Updating QR code for URL:', fullURL);
            
            // 使用Ajax请求本地生成的二维码
            fetch('/generate-qrcode?data=' + encodeURIComponent(fullURL))
                .then(response => {
                    if (!response.ok) {
                        throw new Error('二维码生成请求失败: ' + response.status);
                    }
                    return response.text();
                })
                .then(dataUrl => {
                    // 更新二维码图片
                    const qrImg = document.getElementById('qrCodeImg');
                    if (qrImg) {
                        qrImg.src = dataUrl;
                        console.log('二维码已更新');
                    }
                    
                    // 更新显示的URL
                    const urlElement = document.getElementById('currentUrl');
                    if (urlElement) {
                        urlElement.innerText = fullURL;
                    }
                })
                .catch(error => {
                    console.error('获取二维码失败:', error);
                });
                
            // 不再更新页面URL和页面链接
        }
        
        // 设置文件上传功能
        function setupFileUpload() {
            const uploadForm = document.getElementById('uploadForm');
            const fileInput = document.getElementById('fileInput');
            const fileList = document.getElementById('fileList');
            const uploadContainer = document.querySelector('.upload-container');
            const uploadProgress = document.querySelector('.upload-progress-bar');
            const uploadStatus = document.querySelector('.upload-status');
            const uploadToggle = document.getElementById('uploadToggle');
            const uploadSection = document.getElementById('uploadSection');
            
            if (!uploadForm || !fileInput) return;
            
            // 显示/隐藏上传表单
            if (uploadToggle && uploadSection) {
                uploadSection.style.display = 'none';
                uploadToggle.addEventListener('click', function(e) {
                    e.preventDefault();
                    if (uploadSection.style.display === 'none') {
                        uploadSection.style.display = 'block';
                        uploadToggle.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg> 取消上传';
                    } else {
                        uploadSection.style.display = 'none';
                        uploadToggle.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="17 8 12 3 7 8"></polyline><line x1="12" y1="3" x2="12" y2="15"></line></svg> 上传文件';
                    }
                });
            }
            
            // 文件选择后更新界面
            fileInput.addEventListener('change', function() {
                updateFileList();
            });
            
            // 拖放文件处理
            if (uploadContainer) {
                uploadContainer.addEventListener('dragover', function(e) {
                    e.preventDefault();
                    uploadContainer.classList.add('drag-active');
                });
                
                uploadContainer.addEventListener('dragleave', function() {
                    uploadContainer.classList.remove('drag-active');
                });
                
                uploadContainer.addEventListener('drop', function(e) {
                    e.preventDefault();
                    uploadContainer.classList.remove('drag-active');
                    // Don't directly assign to fileInput.files, handle files separately
                    const droppedFiles = e.dataTransfer.files;
                    if (droppedFiles.length > 0) {
                        // Update the visual file list based on dropped files
                        updateFileListDisplay(droppedFiles);
                        // Store dropped files for form submission
                        uploadForm.droppedFiles = droppedFiles; 
                    }
                });
            }
            
            // 更新文件列表显示
            function updateFileList() {
                 // Use stored dropped files if available, otherwise use fileInput
                const filesToDisplay = uploadForm.droppedFiles || fileInput.files;
                updateFileListDisplay(filesToDisplay);
            }

            // Helper function to update the file list UI
            function updateFileListDisplay(files) {
                fileList.innerHTML = '';
                if (files && files.length > 0) {
                    for (let i = 0; i < files.length; i++) {
                        const file = files[i];
                        const fileSize = formatFileSize(file.size);
                        const listItem = document.createElement('div');
                        listItem.className = 'file-list-item';
                        listItem.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path><polyline points="14 2 14 8 20 8"></polyline></svg>' +
                                          file.name + ' <span style="color:var(--text-light);margin-left:auto;">' + fileSize + '</span>';
                        fileList.appendChild(listItem);
                    }
                } else {
                    fileList.innerHTML = '<div style="color:var(--text-light);text-align:center;">未选择文件</div>';
                }
            }
            
            // 格式化文件大小
            function formatFileSize(bytes) {
                if (bytes === 0) return '0 B';
                const units = ['B', 'KB', 'MB', 'GB', 'TB'];
                const i = Math.floor(Math.log(bytes) / Math.log(1024));
                return (bytes / Math.pow(1024, i)).toFixed(2) + ' ' + units[i];
            }
            
            // 表单提交处理
            uploadForm.addEventListener('submit', function(e) {
                e.preventDefault();
                
                const filesToUpload = uploadForm.droppedFiles || fileInput.files;

                if (!filesToUpload || filesToUpload.length === 0) {
                    uploadStatus.innerHTML = '<div class="upload-error">请选择或拖放要上传的文件</div>';
                    return;
                }
                
                const formData = new FormData(); // Create new FormData
                for (let i = 0; i < filesToUpload.length; i++) {
                    formData.append('file', filesToUpload[i]); // Append files one by one
                }
                

                // 清除之前的状态
                uploadStatus.innerHTML = '';
                uploadProgress.style.width = '0%';
                
                // 当前路径作为目标目录
                const currentPath = '{{.CurrentPath}}';
                
                // 设置表单action属性，确保即使JavaScript失效，表单也能提交到正确URL
                // Correctly form the upload URL based on CurrentPath
                let uploadURL = '/upload/';
                if (currentPath && currentPath !== "/" && currentPath !== ".") {
                    uploadURL += strings.TrimPrefix(currentPath, "/");
                }
                uploadForm.action = uploadURL;
                
                // 创建上传请求
                const xhr = new XMLHttpRequest();
                xhr.open('POST', uploadURL, true); // Use the constructed URL
                
                // 进度处理
                xhr.upload.addEventListener('progress', function(e) {
                    if (e.lengthComputable) {
                        const percentComplete = (e.loaded / e.total) * 100;
                        uploadProgress.style.width = percentComplete + '%';
                    }
                });
                
                // 成功处理
                xhr.addEventListener('load', function() {
                    if (xhr.status === 200) {
                        uploadStatus.innerHTML = '<div class="upload-success">文件上传成功！</div>';
                        // 成功后刷新页面显示新上传的文件
                        setTimeout(function() {
                            window.location.reload();
                        }, 1500);
                    } else {
                        uploadStatus.innerHTML = '<div class="upload-error">上传失败: ' + xhr.statusText + '</div>';
                    }
                });
                
                // 错误处理
                xhr.addEventListener('error', function() {
                    uploadStatus.innerHTML = '<div class="upload-error">上传出错，请检查网络连接</div>';
                });
                
                // 发送数据
                xhr.send(formData);
            });
            
            // Initial call to display status if no files selected
            updateFileListDisplay(null);
        }
    </script>
</head>
<body>
    <header>
        <div class="container header-content">
            <h1>文件服务器</h1>
            <div class="header-buttons">
                <button id="uploadToggle" class="upload-btn">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="17 8 12 3 7 8"></polyline><line x1="12" y1="3" x2="12" y2="15"></line></svg>
                    上传文件
                </button>
                <div class="qr-container">
                    <div class="qr-code">
                        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="5" height="5" rx="1"></rect><rect x="16" y="3" width="5" height="5" rx="1"></rect><rect x="3" y="16" width="5" height="5" rx="1"></rect><path d="M21 16h-3a2 2 0 0 0-2 2v3"></path><path d="M21 21v.01"></path><path d="M12 7v3a2 2 0 0 1-2 2H7"></path><path d="M3 12h.01"></path><path d="M12 3h.01"></path><path d="M12 16v.01"></path><path d="M16 12h1"></path><path d="M21 12v.01"></path><path d="M12 21v-1"></path></svg>
                    </div>
                    <div class="qr-popup">
                        <div class="qr-close">✕</div>
                        <img id="qrCodeImg" src="{{.QRCodeURL}}" alt="扫描二维码访问该页面">
                        <p id="currentUrl">{{.CurrentURL}}</p>
                        <div class="ip-selector">
                            <select onchange="updateQRCode(this)">
                                {{range .IPAddresses}}
                                <option value="{{.IP}}" {{if eq $.SelectedIP .IP}}selected{{end}}>{{.DisplayName}}</option>
                                {{end}}
                            </select>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </header>
    
    <div class="container">
        <!-- 文件上传区域 -->
        <div id="uploadSection" class="upload-container">
            <form id="uploadForm" class="upload-form" enctype="multipart/form-data" method="post" action="">
                <div class="file-input-container">
                    <label for="fileInput" class="file-input-label">
                        <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="17 8 12 3 7 8"></polyline><line x1="12" y1="3" x2="12" y2="15"></line></svg>
                        <div style="margin-top:0.5rem;">点击选择文件或拖放文件到此处</div>
                    </label>
                    <input type="file" id="fileInput" name="file" class="file-input" multiple>
                </div>
                <div class="file-list" id="fileList">
                    <div style="color:var(--text-light);text-align:center;">未选择文件</div>
                </div>
                <div class="upload-progress">
                    <div class="upload-progress-bar"></div>
                </div>
                <div class="upload-status"></div>
                <div class="upload-actions">
                    <button type="submit" class="upload-btn">
                        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="17 8 12 3 7 8"></polyline><line x1="12" y1="3" x2="12" y2="15"></line></svg>
                        开始上传
                    </button>
                </div>
            </form>
        </div>
        
        <div class="file-browser">
            {{if and .ShowBackButton (ne .CurrentPath "/")}}
            <div class="back">
                <a href="{{.ParentPath}}">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"></path></svg>
                    返回上级目录
                </a>
            </div>
            {{end}}
            
            <table>
                <thead>
                    <tr>
                        <th>名称</th>
                        <th style="width:180px;text-align:center">修改时间</th>
                        <th style="width:100px;text-align:center">大小</th>
                    </tr>
                </thead>
                <tbody>
                    {{if eq (len .Files) 0}}
                    <tr>
                        <td colspan="3" style="text-align:center;color:var(--text-light)">此目录为空</td>
                    </tr>
                    {{end}}
                    {{range .Files}}
                    <tr>
                        <td>
                            {{if .IsDir}}
                            <a href="{{.Path}}" class="folder">
                                <span class="icon">
                                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"></path><path d="M2 10h20"></path></svg>
                                </span>
                                {{.Name}}
                            </a>
                            {{else}}
                            <a href="{{.Path}}" class="file">
                                <span class="icon">
                                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path><polyline points="14 2 14 8 20 8"></polyline></svg>
                                </span>
                                {{.Name}}
                            </a>
                            {{end}}
                        </td>
                        <td class="time">{{.ModTime}}</td>
                        <td class="size">{{.Size}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
    </div>
    
    <footer class="container">
        <p>文件服务器 · 内网分享工具</p>
    </footer>
</body>
</html>`

// 将字节大小转换为人类可读的格式
func humanizeSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// 获取本机所有IP地址
func getAllIPs() []IPAddress {
	var ips []IPAddress

	// 添加localhost
	ips = append(ips, IPAddress{
		IP:          "localhost",
		DisplayName: "本地 (localhost)",
	})

	// 获取所有网卡接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	var ipList []IPAddress

	// 遍历所有网卡接口
	for _, iface := range interfaces {
		// 跳过禁用的接口
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// 获取接口地址
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// 遍历接口地址
		for _, addr := range addrs {
			var ip net.IP

			// 检查IP地址类型
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			default:
				continue
			}

			// 只处理IPv4地址
			if ip.To4() == nil {
				continue
			}

			ipStr := ip.String()
			// 跳过回环地址
			if ipStr == "127.0.0.1" {
				continue
			}

			// 创建IP地址信息
			ipInfo := IPAddress{
				IP:          ipStr,
				DisplayName: fmt.Sprintf("%s (%s)", ipStr, iface.Name),
			}

			ipList = append(ipList, ipInfo)
		}
	}

	// 对IP地址进行排序，优先显示192.168开头的地址
	sort.Slice(ipList, func(i, j int) bool {
		// 192.168开头的IP优先
		if strings.HasPrefix(ipList[i].IP, "192.168") && !strings.HasPrefix(ipList[j].IP, "192.168") {
			return true
		}
		// 然后是其他192开头的IP
		if strings.HasPrefix(ipList[i].IP, "192") && !strings.HasPrefix(ipList[j].IP, "192") {
			return true
		}
		// 再是10开头的IP
		if strings.HasPrefix(ipList[i].IP, "10.") && !strings.HasPrefix(ipList[j].IP, "10.") {
			return true
		}
		// 最后按字母顺序排序
		return ipList[i].IP < ipList[j].IP
	})

	// 添加所有IP
	ips = append(ips, ipList...)

	return ips
}

// 创建二维码URL - 本地生成Base64编码的二维码图片
func generateQRCodeURL(content string) string {
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		log.Printf("生成二维码错误: %v", err)
		return ""
	}

	// 设置二维码参数
	qr.DisableBorder = false

	// 生成PNG图片数据
	png, err := qr.PNG(150)
	if err != nil {
		log.Printf("生成二维码PNG错误: %v", err)
		return ""
	}

	// 转换为base64编码
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}

// 获取文件的MIME类型
func getMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeType, ok := mimeTypes[ext]
	if ok {
		return mimeType
	}
	return "application/octet-stream"
}

// 配置服务器并启动
func main() {
	// 显示版本信息
	log.Printf("文件服务器 v%s 正在启动...", Version)

	// 初始化服务器
	config := initConfig()

	// 设置路由处理器
	setupRoutes(config)

	// 启动服务器
	log.Printf("服务器已启动: http://%s", serverAddr)
	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		log.Fatal("服务器启动失败: ", err)
	}
}

// 服务器配置结构
type ServerConfig struct {
	absShareDir string      // 共享目录的绝对路径
	allIPs      []IPAddress // 所有可用IP地址
	defaultIP   string      // 默认IP地址
}

// 初始化服务器配置
func initConfig() *ServerConfig {
	// 获取共享目录的绝对路径
	absShareDir, err := filepath.Abs(shareDir)
	if err != nil {
		log.Fatalf("无法获取共享目录的绝对路径: %v", err)
	}

	// 检查共享目录是否存在
	if _, err := os.Stat(absShareDir); os.IsNotExist(err) {
		log.Fatalf("共享目录 %s 不存在", absShareDir)
	}

	// 获取所有可用IP地址
	allIPs := getAllIPs()

	// 默认选择第一个非localhost的IP（通常是优先级最高的IP）
	var defaultIP string
	for _, ip := range allIPs {
		if ip.IP != "localhost" {
			defaultIP = ip.IP
			break
		}
	}

	// 如果没有找到合适的IP，使用localhost
	if defaultIP == "" {
		defaultIP = "localhost"
	}

	return &ServerConfig{
		absShareDir: absShareDir,
		allIPs:      allIPs,
		defaultIP:   defaultIP,
	}
}

// 设置HTTP路由处理器
func setupRoutes(config *ServerConfig) {
	// 处理二维码生成请求
	http.HandleFunc("/generate-qrcode", handleQRCodeGeneration)

	// 处理文件上传请求 - 只保留带斜杠的路由
	http.HandleFunc("/upload/", func(w http.ResponseWriter, r *http.Request) {
		handleFileUpload(w, r, config.absShareDir)
	})

	// 处理文件下载请求
	http.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		handleFileDownload(w, r, config.absShareDir)
	})

	// 处理主页和目录浏览请求
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleDirectoryBrowsing(w, r, config)
	})
}

// 处理二维码生成请求
func handleQRCodeGeneration(w http.ResponseWriter, r *http.Request) {
	data := r.URL.Query().Get("data")
	if data == "" {
		http.Error(w, "缺少二维码数据", http.StatusBadRequest)
		return
	}

	qrCodeURL := generateQRCodeURL(data)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, qrCodeURL)
}

// 处理文件下载请求
func handleFileDownload(w http.ResponseWriter, r *http.Request, absShareDir string) {
	// 从URL路径获取相对路径，去除/download/前缀
	urlRelativePath := strings.TrimPrefix(r.URL.Path, "/download/") // Corrected prefix
	// urlRelativePath = strings.TrimPrefix(urlRelativePath, "/") // This is likely not needed now

	// 如果为空，返回错误
	if urlRelativePath == "" {
		http.Error(w, "无效的下载路径", http.StatusBadRequest)
		return
	}

	// 验证路径，获取清理后的相对URL路径和绝对本地路径
	_, fullPath, err := validateRequestPath(urlRelativePath, absShareDir)
	if err != nil {
		log.Printf("路径验证失败: %v (原始路径: %s)", err, urlRelativePath)
		http.Error(w, "禁止访问或路径无效", http.StatusForbidden)
		return
	}

	// 获取文件信息
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "文件不存在", http.StatusNotFound)
		} else {
			log.Printf("获取文件信息错误: %v (路径: %s)", err, fullPath)
			http.Error(w, "服务器内部错误", http.StatusInternalServerError)
		}
		return
	}

	// 确保是文件而非目录
	if fileInfo.IsDir() {
		http.Error(w, "不能下载目录", http.StatusBadRequest)
		return
	}

	// 提供文件下载
	serveFile(w, fullPath, fileInfo)
}

// 处理目录浏览请求
func handleDirectoryBrowsing(w http.ResponseWriter, r *http.Request, config *ServerConfig) {
	// 获取查询参数中的IP（仅用于二维码生成）
	selectedIP := r.URL.Query().Get("ip")

	// 如果没有选择IP，使用默认IP
	if selectedIP == "" {
		selectedIP = config.defaultIP
	}

	// 检查IP是否在可用列表中
	validIP := false
	for _, ip := range config.allIPs {
		if ip.IP == selectedIP {
			validIP = true
			break
		}
	}

	// 如果IP无效，使用默认IP
	if !validIP {
		selectedIP = config.defaultIP
	}

	// 构建服务器URL基础（用于二维码）
	serverURLBase := fmt.Sprintf("http://%s%s", selectedIP, serverAddr)

	// 处理文件浏览请求，IP参数仅用于二维码功能
	handleFileServer(w, r, config.absShareDir, serverURLBase, selectedIP, config.allIPs)
}

// 处理文件服务器请求 - 专注于目录浏览
func handleFileServer(w http.ResponseWriter, r *http.Request, absShareDir, serverURLBase, selectedIP string, allIPs []IPAddress) {
	// 获取、验证和清理请求路径
	// validateRequestPath 返回清理后的URL相对路径和绝对本地路径
	urlRelativePath, fullPath, err := validateRequestPath(r.URL.Path, absShareDir)
	if err != nil {
		log.Printf("路径验证失败: %v (原始路径: %s)", err, r.URL.Path)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// 获取文件信息
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "文件或目录不存在", http.StatusNotFound)
		} else {
			log.Printf("获取文件信息错误: %v (路径: %s)", err, fullPath)
			http.Error(w, "服务器内部错误", http.StatusInternalServerError)
		}
		return
	}

	// 如果是目录，显示目录内容 - 使用清理后的URL相对路径
	if fileInfo.IsDir() {
		// 确保传递给listDirectory的路径以/开头
		listDirectory(w, fullPath, "/"+urlRelativePath, serverURLBase, selectedIP, allIPs)
		return
	}

	// 对于文件请求，重定向到专门的下载路由（不包含IP参数）
	// 使用清理后的URL相对路径构建下载链接
	http.Redirect(w, r, "/download/"+urlRelativePath, http.StatusFound)
}

// 验证请求路径，返回处理后的请求路径、完整文件系统路径和可能的错误
func validateRequestPath(urlPath, absShareDir string) (string, string, error) {
	// 解码URL路径
	decodedPath, err := url.PathUnescape(urlPath)
	if err != nil {
		return "", "", fmt.Errorf("无效的URL路径编码: %v", err)
	}

	// 清理URL路径，确保使用正斜杠
	cleanedURLPath := path.Clean(decodedPath)
	cleanedURLPath = strings.TrimPrefix(cleanedURLPath, "/")

	// 将清理后的URL路径转换为本地文件系统路径
	localPath := filepath.Join(absShareDir, filepath.FromSlash(cleanedURLPath))

	// 再次清理本地路径以处理".."等
	cleanedLocalPath := filepath.Clean(localPath)

	// 安全检查：确保最终路径在共享目录内
	relPath, err := filepath.Rel(absShareDir, cleanedLocalPath)
	if err != nil {
		// 如果无法计算相对路径，可能是因为路径形式问题或确实在外部
		return "", "", fmt.Errorf("路径安全检查错误: %v", err)
	}
	if strings.HasPrefix(relPath, "..") || relPath == ".." {
		return "", "", fmt.Errorf("禁止访问父目录")
	}

	// 返回清理后的URL相对路径和绝对本地路径
	return cleanedURLPath, cleanedLocalPath, nil
}

// 提供文件下载
func serveFile(w http.ResponseWriter, fullPath string, fileInfo os.FileInfo) {
	file, err := os.Open(fullPath)
	if err != nil {
		http.Error(w, "无法打开文件", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// 获取文件名（处理非ASCII字符）
	fileName := filepath.Base(fullPath)

	// 设置内容类型
	contentType := getMimeType(fileName)
	w.Header().Set("Content-Type", contentType)

	// 设置文件下载头
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
		fileName, url.PathEscape(fileName)))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// 发送文件内容
	io.Copy(w, file)
}

// 检查是否为根目录
func isRootDirectory(requestPath string) bool {
	// 清理路径并检查是否为根 ("/" or ".")
	cleanPath := path.Clean(strings.ReplaceAll(requestPath, "\\", "/"))
	return cleanPath == "/" || cleanPath == "." || cleanPath == ""
}

// 列出目录内容
func listDirectory(w http.ResponseWriter, fullPath, requestPath, serverURLBase, selectedIP string, allIPs []IPAddress) {
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		http.Error(w, "无法读取目录", http.StatusInternalServerError)
		return
	}

	// 构建文件列表
	files := buildFileList(entries, requestPath)

	// 准备父目录路径（不包含IP参数）
	parentPath := prepareParentPath(requestPath)

	// 生成二维码URL
	qrCodeURL, currentPageURL := generateQRData(requestPath, serverURLBase, selectedIP)

	// 确定是否显示返回上一级目录按钮
	hideBackButton := isRootDirectory(requestPath)

	// 执行模板
	renderTemplate(w, files, requestPath, parentPath, qrCodeURL, currentPageURL, serverURLBase, selectedIP, allIPs, serverAddr, hideBackButton)
}

// 构建文件列表
func buildFileList(entries []os.DirEntry, requestPath string) []FileInfo {
	files := make([]FileInfo, 0, len(entries))

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 使用path包进行URL路径拼接，确保使用正斜杠
		var relativePath string
		// 清理请求路径，统一使用正斜杠
		cleanRequestPath := path.Clean(strings.ReplaceAll(requestPath, "\\", "/"))
		// 确保requestPath是相对路径（不以/开头）
		cleanRequestPath = strings.TrimPrefix(cleanRequestPath, "/")

		if cleanRequestPath == "" || cleanRequestPath == "." {
			relativePath = entry.Name()
		} else {
			relativePath = path.Join(cleanRequestPath, entry.Name())
		}

		// 确保relativePath不以/开头，因为我们要手动添加前缀
		relativePath = strings.TrimPrefix(relativePath, "/")

		// 生成浏览/下载URL（不包含IP参数）
		var displayPath string

		if entry.IsDir() {
			// 目录链接 - 始终以/开头
			displayPath = "/" + relativePath
			if !strings.HasSuffix(displayPath, "/") {
				displayPath += "/"
			}
		} else {
			// 文件链接 - 始终以/download/开头，并确保路径不重复加/
			displayPath = "/download/" + relativePath
		}

		files = append(files, FileInfo{
			Name:    entry.Name(),
			Path:    displayPath,
			Size:    humanizeSize(info.Size()),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			IsDir:   entry.IsDir(),
		})
	}

	return files
}

// 准备父目录路径 (for URL)
func prepareParentPath(requestPath string) string {
	// 清理路径并确保使用正斜杠
	cleanPath := path.Clean(strings.ReplaceAll(requestPath, "\\", "/"))
	parentPath := path.Dir(cleanPath)

	// 确保父路径以/开头并以/结尾 (unless it's the root)
	if parentPath == "." || parentPath == "" {
		parentPath = "/"
	} else {
		if !strings.HasPrefix(parentPath, "/") {
			parentPath = "/" + parentPath
		}
		if !strings.HasSuffix(parentPath, "/") {
			parentPath += "/"
		}
	}
	return parentPath
}

// 生成二维码相关数据
func generateQRData(requestPath, serverURLBase, selectedIP string) (string, string) {
	// requestPath 已经是清理过的URL相对路径
	currentPath := requestPath

	// 构建URL
	currentPageURL := serverURLBase
	// 确保URL以/结尾（如果非根目录）
	if !strings.HasSuffix(currentPageURL, "/") {
		currentPageURL += "/"
	}
	// 拼接相对路径，去除可能的前导/
	currentPageURL += strings.TrimPrefix(currentPath, "/")

	// 添加IP参数
	if selectedIP != "" && selectedIP != "localhost" { // Only add IP if it's not localhost
		if !strings.Contains(currentPageURL, "?") {
			currentPageURL += "?ip=" + selectedIP
		} else {
			currentPageURL += "&ip=" + selectedIP // Use & if params already exist
		}
	}

	// 生成二维码
	return generateQRCodeURL(currentPageURL), currentPageURL
}

// 渲染HTML模板
func renderTemplate(w http.ResponseWriter, files []FileInfo, currentPath, parentPath, qrCodeURL, currentURL, serverURLBase, selectedIP string, allIPs []IPAddress, serverPort string, hideBackButton bool) {
	// 解析模板
	t, err := template.New("directory").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, "模板解析错误", http.StatusInternalServerError)
		return
	}

	// 准备模板数据
	data := struct {
		Files          []FileInfo
		CurrentPath    string
		ParentPath     string
		QRCodeURL      string
		CurrentURL     string
		ServerURLBase  string
		IPAddresses    []IPAddress
		SelectedIP     string
		ServerPort     string
		ShowBackButton bool
	}{
		Files:          files,
		CurrentPath:    currentPath,
		ParentPath:     parentPath,
		QRCodeURL:      qrCodeURL,
		CurrentURL:     currentURL,
		ServerURLBase:  serverURLBase,
		IPAddresses:    allIPs,
		SelectedIP:     selectedIP,
		ServerPort:     serverPort,
		ShowBackButton: !hideBackButton,
	}

	// 执行模板
	if err := t.Execute(w, data); err != nil {
		http.Error(w, "模板执行错误", http.StatusInternalServerError)
	}
}

// 处理文件上传请求
func handleFileUpload(w http.ResponseWriter, r *http.Request, baseShareDir string) {
	// 提取上传的目标相对路径 (URL Path থেকে /upload/ বাদ দিয়ে)
	urlTargetPath := strings.TrimPrefix(r.URL.Path, "/upload/")

	// 验证目标路径是否有效，并获取完整的本地目标目录路径
	_, targetDirFullPath, err := validateRequestPath(urlTargetPath, baseShareDir)
	if err != nil {
		log.Printf("上传路径验证失败: %v (原始路径: %s)", err, urlTargetPath)
		http.Error(w, "无效的上传目标路径", http.StatusBadRequest)
		return
	}

	// 确保目标路径是一个目录
	fileInfo, err := os.Stat(targetDirFullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "上传目标目录不存在", http.StatusNotFound)
		} else {
			log.Printf("获取上传目录信息错误: %v (路径: %s)", err, targetDirFullPath)
			http.Error(w, "服务器内部错误", http.StatusInternalServerError)
		}
		return
	}
	if !fileInfo.IsDir() {
		http.Error(w, "上传目标必须是一个目录", http.StatusBadRequest)
		return
	}

	// 如果是GET请求，重定向到对应的目录浏览页面
	if r.Method == http.MethodGet {
		// 重定向到浏览路径，而不是上传路径
		redirectPath := "/" + urlTargetPath
		if !strings.HasSuffix(redirectPath, "/") {
			redirectPath += "/"
		}
		http.Redirect(w, r, redirectPath, http.StatusSeeOther)
		return
	}

	// 处理POST请求
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST上传请求", http.StatusMethodNotAllowed)
		return
	}

	// 限制上传大小为1GB
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*1024)

	// 解析多部分表单（文件上传）
	err = r.ParseMultipartForm(32 << 20) // 32MB内存缓冲
	if err != nil {
		http.Error(w, "解析上传表单失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 获取上传的文件
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		http.Error(w, "没有找到上传的文件", http.StatusBadRequest)
		return
	}

	// 存储上传状态
	var uploadStatus struct {
		Success []string
		Failed  []string
	}

	// 处理每个文件
	for _, fileHeader := range files {
		// 打开上传的文件
		file, err := fileHeader.Open()
		if err != nil {
			uploadStatus.Failed = append(uploadStatus.Failed, fileHeader.Filename+": "+err.Error())
			continue
		}
		// 使用 defer确保文件关闭
		func() {
			defer file.Close()

			// 创建目标文件 - 保存到验证过的目标目录
			destPath := filepath.Join(targetDirFullPath, fileHeader.Filename)
			dest, err := os.Create(destPath)
			if err != nil {
				uploadStatus.Failed = append(uploadStatus.Failed, fileHeader.Filename+": "+err.Error())
				return // return from inner func
			}
			// 使用 defer确保目标文件关闭
			defer dest.Close()

			// 复制文件内容
			_, err = io.Copy(dest, file)
			if err != nil {
				uploadStatus.Failed = append(uploadStatus.Failed, fileHeader.Filename+": "+err.Error())
				// 尝试删除可能部分写入的文件
				os.Remove(destPath)
				return // return from inner func
			}

			// 标记为成功
			uploadStatus.Success = append(uploadStatus.Success, fileHeader.Filename)
			log.Printf("文件上传成功: %s -> %s", fileHeader.Filename, destPath)
		}() // Call the inner func immediately
	}

	// 返回成功消息
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "上传完成。成功: %d, 失败: %d", len(uploadStatus.Success), len(uploadStatus.Failed))
}
