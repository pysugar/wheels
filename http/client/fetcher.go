package client

//type connPool struct {
//	mu    sync.Mutex               // TODO: maybe switch to RWMutex
//	conns map[string][]*clientConn // key is host:port
//}

//
//import (
//	"context"
//	"github.com/pysugar/wheels/binproto/http2"
//	"github.com/pysugar/wheels/concurrent"
//	pb "google.golang.org/grpc/health/grpc_health_v1"
//	"log"
//	"net"
//	"net/url"
//	"sync/atomic"
//)
//
//const (
//	ClientPreface = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
//)
//
//type (
//	Fetcher interface {
//		Close()
//	}
//
//	fetcher struct {
//		userAgent  string
//		method     string
//		serializer *concurrent.CallbackSerializer
//		cancel     context.CancelFunc
//	}
//)
//
//var (
//	clientPreface = []byte(ClientPreface)
//)
//
//func NewFetcher() Fetcher {
//	ctx, cancel := context.WithCancel(context.Background())
//	return &fetcher{
//		serializer: concurrent.NewCallbackSerializer(ctx),
//		cancel:     cancel,
//	}
//}
//
//func (f *fetcher) Close() {
//	f.cancel()
//}
//
//func (f *fetcher) sendRequest(conn net.Conn) {
//
//}
//
//func (f *fetcher) CallHTTP2(parsedURL *url.URL) error {
//	conn, err := dialConn(parsedURL)
//	if err != nil {
//		return err
//	}
//	defer conn.Close()
//
//	clientPreface := []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")
//	log.Printf("< Send HTTP/2 Client Preface: %s\n", clientPreface)
//	n, err := conn.Write(clientPreface)
//	if err != nil {
//		log.Printf("Failed to send HTTP/2 Client Preface, err = %v, n = %d\n", err, n)
//		return err
//	}
//
//	// SETTINGS payload:
//	settings := []byte{
//		0x00, 0x03, 0x00, 0x00, 0x00, 0x64, // SETTINGS_MAX_CONCURRENT_STREAMS = 100
//		0x00, 0x04, 0x00, 0x00, 0x40, 0x00, // SETTINGS_INITIAL_WINDOW_SIZE = 16384
//	}
//	err = http2.WriteSettingsFrame(conn, 0, settings)
//	if err != nil {
//		log.Println("Failed to send HTTP/2 settings:", err)
//		return err
//	}
//	log.Printf("Send HTTP/2 Client Preface and Settings Done >\n")
//
//	doneCh := make(chan struct{})
//	go func() {
//		defer close(doneCh)
//		f.readLoop(conn)
//	}()
//
//	localStreamID := atomic.AddUint32(&streamID, 2) - 2
//	err = f.sendRequestHeadersHTTP2(conn, localStreamID, parsedURL)
//	if err != nil {
//		log.Println("Failed to send HTTP/2 request headers:", err)
//		return err
//	}
//
//	requestData := &pb.HealthCheckRequest{}
//	requestBody, err := http2.EncodeGrpcFrame(requestData)
//	if err != nil {
//		log.Println("Failed to BuildGrpcFrame:", err)
//		return err
//	}
//	err = f.sendRequestBodyHTTP2(conn, localStreamID, requestBody)
//	if err != nil {
//		log.Println("Failed to send HTTP/2 request body:", err)
//		return err
//	}
//
//	log.Printf("< Send HTTP/2 request done, url: %v\n", parsedURL)
//	<-doneCh
//	return nil
//}
