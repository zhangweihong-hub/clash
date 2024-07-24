package main

import (
	"os"
	"os/signal"
	"syscall"

	C "clash/constant"
	"clash/hub"
	"clash/proxy/http"
	"clash/proxy/socks"
	"clash/tunnel"

	log "github.com/sirupsen/logrus"
)

// 主函数，程序的入口点
func main() {
    // 加载配置文件
    cfg, err := C.GetConfig()
    if err != nil {
        log.Fatalf("Read config error: %s", err.Error())
    }

    // 初始化默认的HTTP和SOCKS端口
    port, socksPort := C.DefalutHTTPPort, C.DefalutSOCKSPort
    // 获取配置文件的"General"部分
    section := cfg.Section("General")
    // 尝试从配置中读取HTTP端口
    if key, err := section.GetKey("port"); err == nil {
        port = key.Value()
    }
    // 尝试从配置中读取SOCKS端口
    if key, err := section.GetKey("socks-port"); err == nil {
        socksPort = key.Value()
    }

    // 更新隧道配置
    err = tunnel.GetInstance().UpdateConfig()
    if err != nil {
        log.Fatalf("Parse config error: %s", err.Error())
    }

    // 启动HTTP代理服务
    go http.NewHttpProxy(port)
    // 启动SOCKS代理服务
    go socks.NewSocksProxy(socksPort)

    // 如果配置中指定了外部控制器，启动Hub服务
    // Hub
    if key, err := section.GetKey("external-controller"); err == nil {
        go hub.NewHub(key.Value())
    }

    // 监听中断信号，优雅地关闭程序
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
}
