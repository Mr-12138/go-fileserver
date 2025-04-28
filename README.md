# 文件服务器

一个简单易用的文件共享服务器，支持在局域网内快速共享和传输文件。

## 功能特点

- 📂 浏览目录和文件
- 📤 文件上传（支持拖放）
- 📥 文件下载
- 📱 生成二维码，方便移动设备访问
- 💻 自动检测和显示所有网络接口

## 截图

![文件服务器截图](.github/screenshots/screenshot.png)

## 使用方法

### 从二进制文件运行

1. 从 [Releases](https://github.com/用户名/文件服务器/releases) 页面下载最新版本
2. 解压下载的压缩包
3. 根据需要修改默认配置（编辑源码中的`shareDir`和`serverAddr`）
4. 运行可执行文件
5. 打开浏览器访问 `http://localhost:8080` 或本机局域网IP地址

### 从源码运行

```bash
# 克隆仓库
git clone https://github.com/用户名/文件服务器.git
cd 文件服务器

# 直接运行
go run main.go

# 或者构建后运行
go build -o go-fileserver main.go
./go-fileserver  # 在Linux/macOS上
go-fileserver.exe  # 在Windows上
```

## 配置

目前配置在源码的常量中设置：

```go
const (
    // 要共享的目录 - 修改为你想要共享的目录
    shareDir = "D:/Download"
    // 服务器地址 - 修改监听地址和端口
    serverAddr = "localhost:8080"
)
```

## 构建

### 本地构建

```bash
# 普通构建
go build -o go-fileserver main.go

# 带版本号构建
go build -ldflags="-X main.Version=1.0.1" -o go-fileserver main.go

# 优化构建（减小体积，去除调试信息）
go build -ldflags="-s -w -X main.Version=1.0.1" -o go-fileserver main.go
```

### 跨平台构建

```bash
# Windows 64位
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.Version=1.0.1" -o go-fileserver.exe main.go

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.Version=1.0.1" -o go-fileserver main.go

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.Version=1.0.1" -o go-fileserver main.go

# Linux 64位
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=1.0.1" -o go-fileserver main.go
```

## 发布

项目使用GitHub Actions自动构建和发布。当推送一个标签（格式为`v*`，如`v1.0.1`）时，将自动触发构建并创建Release。

```bash
# 本地创建并推送标签
git tag v1.0.1
git push origin v1.0.1
```

## 开源协议

MIT 