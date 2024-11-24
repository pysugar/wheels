package subcmds

import (
	"context"
	"fmt"
	"github.com/pysugar/wheels/http/extensions"
	"io"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/pysugar/wheels/cmd/base"
	"github.com/pysugar/wheels/http/client"
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

			isGRPC, _ := cmd.Flags().GetBool("grpc")
			isHTTP2, _ := cmd.Flags().GetBool("http2")
			method, _ := cmd.Flags().GetString("method")

			targetURL, err := url.Parse(args[0])
			if err != nil {
				log.Printf("invalid url %s\n", args[0])
				return
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			isVerbose, _ := cmd.Flags().GetBool("verbose")
			if isVerbose {
				ctx = client.WithVerbose(ctx)
				ctx = httptrace.WithClientTrace(ctx, extensions.NewDebugClientTrace(fmt.Sprintf("req-%03d",
					atomic.AddUint32(&traceIdGen, 1))))
			}

			fetcher := client.NewFetcher()
			if isGRPC {
				service, method, err := extractServiceAndMethod(targetURL)
				if err != nil {
					log.Printf("invalid service or method %v\n", err)
					return
				}

				requestJson, _ := cmd.Flags().GetString("data")
				protoPath, _ := cmd.Flags().GetString("proto-path")
				reqMessage, resMessage, err := getReqResMessages(protoPath, service, method, isVerbose)
				if err != nil {
					log.Printf("invalid proto file: %s, err: %v\n", protoPath, err)
					return
				}
				err = protojson.Unmarshal([]byte(requestJson), reqMessage)
				if err != nil {
					fmt.Printf("failed to parse JSON to Protobuf: %v", err)
					return
				}

				ctx = client.WithProtocol(ctx, client.HTTP2)
				if er := fetcher.CallGRPC(ctx, targetURL, reqMessage, resMessage); er != nil {
					log.Printf("Call grpc %s error: %v\n", targetURL, er)
					return
				}

				responseJson, err := protojson.Marshal(resMessage)
				if err != nil {
					fmt.Printf("failed to serialize response to JSON: %v", err)
					return
				}
				log.Printf("%s", responseJson)
				return
			}

			if isHTTP2 {
				ctx = client.WithProtocol(ctx, client.HTTP2)
			}

			data, _ := cmd.Flags().GetString("data")
			var body io.Reader
			if data != "" {
				body = strings.NewReader(data)
			}

			req, err := http.NewRequestWithContext(ctx, method, targetURL.String(), body)
			res, er := fetcher.Do(ctx, req)
			if er != nil {
				log.Printf("Call %v %s error: %v\n", client.ProtocolFromContext(ctx), targetURL, err)
				return
			}
			fmt.Printf("http status: %s\n", res.Status)
			resBody, _ := io.ReadAll(res.Body)
			fmt.Printf("%s", resBody)
		},
	}
)

func init() {
	fetchCmd.Flags().StringP("user-agent", "A", "", "User Agent")
	fetchCmd.Flags().StringP("method", "M", "GET", "HTTP Method")
	fetchCmd.Flags().StringP("data", "d", "{}", "request data")
	fetchCmd.Flags().BoolP("grpc", "G", false, "Is GRPC Request Or Not")
	fetchCmd.Flags().BoolP("http2", "H", false, "Is HTTP2 Request Or Not")
	fetchCmd.Flags().BoolP("verbose", "V", false, "Verbose mode")
	fetchCmd.Flags().StringP("proto-path", "P", "", "Proto Path")
	base.AddSubCommands(fetchCmd)
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
