package http

import (
	"io"
	"net"
	"net/http"
	"time"

	C "clash/constant"
)
// HttpAdapter 是一个HTTP适配器，用于处理HTTP请求和响应。
type HttpAdapter struct {
    addr *C.Addr        // 保存HTTP服务的地址信息
    r    *http.Request  // 保存HTTP请求对象
    w    http.ResponseWriter  // 保存HTTP响应写入器
    done chan struct{}  // 用于标记适配器是否完成的通道
}

// Close 表示关闭HttpAdapter，通过向done通道发送结构体来实现。
func (h *HttpAdapter) Close() {
    h.done <- struct{}{}
}

// Addr 返回HttpAdapter的地址。
func (h *HttpAdapter) Addr() *C.Addr {
    return h.addr
}

// Connect 通过给定的代理建立一个HTTP连接。
func (h *HttpAdapter) Connect(proxy C.ProxyAdapter) {
    // 创建一个HTTP传输对象，用于发起HTTP请求
    req := http.Transport{
        Dial: func(network, addr string) (net.Conn, error) {
            return proxy.Conn(), nil
        },
        // 使用默认的HTTP传输设置
        // from http.DefaultTransport
        MaxIdleConns:          100,
        IdleConnTimeout:       90 * time.Second,
        ExpectContinueTimeout: 1 * time.Second,
    }
    // 使用HTTP传输对象发起一个请求
    resp, err := req.RoundTrip(h.r)
    if err != nil {
        return
    }
    defer resp.Body.Close()  // 确保响应体被关闭

    // 复制响应头到适配器的响应写入器
    header := h.w.Header()
    for k, vv := range resp.Header {
        for _, v := range vv {
            header.Add(k, v)
        }
    }
    // 设置响应状态码
    h.w.WriteHeader(resp.StatusCode)
    // 根据响应是否使用了分块传输编码，选择合适的写入器
    var writer io.Writer = h.w
    if len(resp.TransferEncoding) > 0 && resp.TransferEncoding[0] == "chunked" {
        writer = ChunkWriter{Writer: h.w}
    }
    // 复制响应体到适配器的响应写入器
    io.Copy(writer, resp.Body)
}

// ChunkWriter 是一个实现了io.Writer接口的类型，用于处理分块传输编码的写入。
type ChunkWriter struct {
    io.Writer
}

// Write 实现了io.Writer接口的写入方法，用于分块写入数据。
func (cw ChunkWriter) Write(b []byte) (int, error) {
    n, err := cw.Writer.Write(b)
    if err == nil {
        cw.Writer.(http.Flusher).Flush()
    }
    return n, err
}

// NewHttp 创建并返回一个新的HttpAdapter实例，以及一个用于标记适配器完成的通道。
func NewHttp(host string, w http.ResponseWriter, r *http.Request) (*HttpAdapter, chan struct{}) {
    done := make(chan struct{})
    return &HttpAdapter{
        addr: parseHttpAddr(host),
        r:    r,
        w:    w,
        done: done,
    }, done
}