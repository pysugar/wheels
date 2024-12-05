package subcmds

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/pysugar/wheels/cmd/base"
	"github.com/pysugar/wheels/http/client"
	"github.com/pysugar/wheels/http/extensions"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

var (
	traceIdGen uint32

	fetchCmd = &cobra.Command{
		Use:   `fetch https://www.google.com`,
		Short: "fetch http2 response from url",
		Long: `
fetch http2 response from url

fetch http2 response from url: netool fetch https://www.google.com
call grpc service: netool fetch --grpc https://localhost:8443/grpc.health.v1.Health/Check
call grpc via context path: netool fetch --grpc http://localhost:8080/grpc/grpc.health.v1.Health/Check
call grpc service: netool fetch --grpc https://localhost:8443/grpc.health.v1.Health/Check --proto-path=health.proto -d'{"service": ""}'
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				log.Printf("you must specify the url")
				return
			}

			targetURL, err := url.Parse(args[0])
			if err != nil {
				log.Printf("invalid url %s\n", args[0])
				return
			}

			isGRPC, _ := cmd.Flags().GetBool("grpc")
			if isGRPC {
				err := gRPCCall(cmd, targetURL)
				if err != nil {
					log.Fatal(err)
				}
				time.Sleep(time.Second)
				return
			}

			isWS, _ := cmd.Flags().GetBool("websocket")
			isGorilla, _ := cmd.Flags().GetBool("gorilla")
			if isWS || isGorilla {
				err := wsCall(cmd, targetURL, isGorilla)
				if err != nil {
					log.Fatal(err)
				}
				time.Sleep(time.Second)
				return
			}

			isVerbose, _ := cmd.Flags().GetBool("verbose")
			isUpgrade, _ := cmd.Flags().GetBool("upgrade")
			ctx, cancel := newContext(isVerbose, isUpgrade)
			defer cancel()

			isHTTP1, _ := cmd.Flags().GetBool("http1")
			isHTTP2, _ := cmd.Flags().GetBool("http2")
			if isHTTP1 {
				ctx = client.WithProtocol(ctx, client.HTTP1)
			} else if isHTTP2 {
				ctx = client.WithProtocol(ctx, client.HTTP2)
			}
			method, _ := cmd.Flags().GetString("method")

			var body io.Reader
			var contentLength int64 = 0
			if strings.EqualFold(method, http.MethodPost) || strings.EqualFold(method, http.MethodPut) || strings.EqualFold(method, http.MethodPatch) {
				data, _ := cmd.Flags().GetString("data")
				if data != "" {
					body = strings.NewReader(data)
					contentLength = int64(len(data))
				}
			}

			req, err := http.NewRequestWithContext(ctx, method, targetURL.String(), body)
			if err != nil {
				fmt.Printf("failed to create request: %v\n", err)
				return
			}
			req.ContentLength = contentLength

			res, er := client.NewFetcher().Do(ctx, req)
			if er != nil {
				fmt.Printf("Call %v %s error: %v\n", client.ProtocolFromContext(ctx), targetURL, er)
				return
			}

			if isVerbose {
				fmt.Println("\n+++++++++++++++++++++++++++")
			}
			fmt.Printf("%s %s\r\n", res.Status, res.Proto)
			for k, v := range res.Header {
				fmt.Printf("%s: %s\r\n", k, strings.Join(v, ","))
			}
			fmt.Printf("\r\n")
			resBody, _ := io.ReadAll(res.Body)
			fmt.Printf("%s", resBody)
		},
	}
)

func init() {
	fetchCmd.Flags().StringP("user-agent", "A", "", "User Agent")
	fetchCmd.Flags().StringP("method", "M", "GET", "HTTP Method")
	fetchCmd.Flags().StringP("data", "d", "{}", "request data")
	fetchCmd.Flags().BoolP("http1", "", false, "Is HTTP2 Request Or Not")
	fetchCmd.Flags().BoolP("http2", "", false, "Is HTTP2 Request Or Not")
	fetchCmd.Flags().BoolP("websocket", "W", false, "Is WebSocket Request Or Not")
	fetchCmd.Flags().BoolP("gorilla", "g", false, "Is Gorilla WebSocket Request Or Not")
	fetchCmd.Flags().BoolP("grpc", "G", false, "Is GRPC Request Or Not")
	fetchCmd.Flags().BoolP("verbose", "V", false, "Verbose mode")
	fetchCmd.Flags().BoolP("upgrade", "U", false, "try http upgrade")
	fetchCmd.Flags().StringP("proto-path", "P", "", "Proto Path")
	base.AddSubCommands(fetchCmd)
}

func wsCall(cmd *cobra.Command, targetURL *url.URL, isGorilla bool) error {
	isVerbose, _ := cmd.Flags().GetBool("verbose")
	ctx, cancel := newContext(isVerbose, true)
	defer cancel()
	ctx = client.WithProtocol(ctx, client.WebSocket)
	if isGorilla {
		ctx = client.WithGorilla(ctx)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		fmt.Printf("failed to create request: %v\n", err)
		return err
	}
	err = client.NewFetcher().WS(ctx, req)
	if err != nil {
		fmt.Printf("failed to fetch websocket: %v\n", err)
		return err
	}
	return nil
}

func gRPCCall(cmd *cobra.Command, targetURL *url.URL) error {
	service, method, err := extractServiceAndMethod(targetURL)
	if err != nil {
		log.Printf("invalid service or method %v\n", err)
		return err
	}

	requestJson, _ := cmd.Flags().GetString("data")
	protoPath, _ := cmd.Flags().GetString("proto-path")
	isVerbose, _ := cmd.Flags().GetBool("verbose")
	reqMessage, resMessage, err := getReqResMessages(protoPath, service, method, isVerbose)
	if err != nil {
		log.Printf("invalid proto file: %s, err: %v\n", protoPath, err)
		return err
	}

	err = protojson.Unmarshal([]byte(requestJson), reqMessage)
	if err != nil {
		fmt.Printf("failed to parse JSON to Protobuf: %v", err)
		return err
	}

	isUpgrade, _ := cmd.Flags().GetBool("upgrade")
	ctx, cancel := newContext(isVerbose, isUpgrade)
	defer cancel()

	ctx = client.WithProtocol(ctx, client.HTTP2)
	if er := client.NewFetcher().CallGRPC(ctx, targetURL, reqMessage, resMessage); er != nil {
		log.Printf("Call grpc %s error: %v\n", targetURL, er)
		return err
	}

	responseJson, err := protojson.Marshal(resMessage)
	if err != nil {
		fmt.Printf("failed to serialize response to JSON: %v", err)
		return err
	}
	fmt.Printf("%s\n", responseJson)
	return nil
}

func newContext(isVerbose, isUpgrade bool) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	if isVerbose {
		ctx = client.WithVerbose(ctx)
		traceId := atomic.AddUint32(&traceIdGen, 1)
		ctx = httptrace.WithClientTrace(ctx, extensions.NewDebugClientTrace(fmt.Sprintf("trace-req-%03d", traceId)))
	}
	if isUpgrade {
		ctx = client.WithUpgrade(ctx)
	}
	return ctx, cancel
}

func extractServiceAndMethod(parsedURL *url.URL) (string, string, error) {
	path := strings.Trim(parsedURL.Path, "/")

	segments := strings.Split(path, "/")

	if len(segments) < 2 {
		return "", "", fmt.Errorf("missing servie or method")
	}

	service := segments[len(segments)-2]
	method := segments[len(segments)-1]

	return service, method, nil
}

func getReqResMessages(protoPath, serviceName, methodName string, verbose bool) (proto.Message, proto.Message, error) {
	parser := protoparse.Parser{
		ImportPaths: []string{filepath.Dir(protoPath)},
	}

	fileDescriptors, err := parser.ParseFiles(filepath.Base(protoPath))
	if err != nil {
		return nil, nil, fmt.Errorf("resolve proto file failure: %v", err)
	}

	if verbose {
		fmt.Printf("dir: %+v\n", filepath.Dir(protoPath))
		fmt.Printf("base: %+v\n", filepath.Base(protoPath))
		for _, fd := range fileDescriptors {
			fmt.Printf("fd: %v\n", fd)
		}
	}

	if len(fileDescriptors) == 0 {
		return nil, nil, fmt.Errorf("file descriptor not found")
	}

	descriptor := fileDescriptors[0]
	srvDesc := descriptor.FindService(serviceName)
	if srvDesc == nil {
		showServices(descriptor.UnwrapFile())
		return nil, nil, fmt.Errorf("service not found: %s", serviceName)
	}

	method := srvDesc.FindMethodByName(methodName)
	if method == nil {
		fmt.Println(serviceInfo(srvDesc.UnwrapService()))
		return nil, nil, fmt.Errorf("method not found: %s in service: %s", methodName, serviceName)
	}

	inputDesc := method.GetInputType()
	outputDesc := method.GetOutputType()

	if inputDesc == nil || outputDesc == nil {
		return nil, nil, fmt.Errorf("input or output is empty")
	}

	reqMessage := dynamicpb.NewMessage(inputDesc.UnwrapMessage())
	resMessage := dynamicpb.NewMessage(outputDesc.UnwrapMessage())

	return reqMessage, resMessage, nil
}

func showServices(fd protoreflect.FileDescriptor) {
	for i := 0; i < fd.Services().Len(); i++ {
		fmt.Println(serviceInfo(fd.Services().Get(i)))
	}
}

func serviceInfo(srvDesc protoreflect.ServiceDescriptor) string {
	var out strings.Builder
	out.WriteString("service: ")
	out.WriteString(string(srvDesc.Name()))
	out.WriteString("(")
	out.WriteString(string(srvDesc.FullName()))
	out.WriteString(")")
	out.WriteString("\n")
	for k := 0; k < srvDesc.Methods().Len(); k++ {
		mth := srvDesc.Methods().Get(k)
		out.WriteString("\t")
		out.WriteString(string(mth.Name()))
		out.WriteString("(")
		out.WriteString(string(mth.FullName()))
		out.WriteString(")\n")
	}
	return out.String()
}
