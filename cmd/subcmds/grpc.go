package subcmds

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"log"
	"net/url"
	"strings"

	"github.com/pysugar/wheels/cmd/base"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/dynamicpb"
)

type grpcReflectClient struct {
	reflectionClient reflectionpb.ServerReflectionClient
}

func (c *grpcReflectClient) findMethodDesc(serviceName, methodName string) (protoreflect.MethodDescriptor, error) {
	ctx := context.Background()
	stream, err := c.reflectionClient.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reflection stream: %v", err)
	}

	if er := stream.Send(&reflectionpb.ServerReflectionRequest{
		MessageRequest: &reflectionpb.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: serviceName,
		},
	}); er != nil {
		return nil, fmt.Errorf("failed to send reflection request for service: %v", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive service reflection response: %v", err)
	}

	fileDescriptorResponse, ok := resp.MessageResponse.(*reflectionpb.ServerReflectionResponse_FileDescriptorResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type for file descriptor")
	}

	for _, fdBytes := range fileDescriptorResponse.FileDescriptorResponse.FileDescriptorProto {
		fdProto := &descriptorpb.FileDescriptorProto{}
		if er := proto.Unmarshal(fdBytes, fdProto); er != nil {
			return nil, fmt.Errorf("failed to unmarshal FileDescriptorProto: %v", er)
		}

		fileDesc, err := protodesc.NewFile(fdProto, protoregistry.GlobalFiles)
		if err != nil {
			return nil, fmt.Errorf("failed to create FileDescriptor: %v", err)
		}

		serviceDesc := fileDesc.Services().ByName(protoreflect.Name(serviceName))
		if serviceDesc == nil {
			continue
		}

		methodDesc := serviceDesc.Methods().ByName(protoreflect.Name(methodName))
		if methodDesc == nil {
			return nil, fmt.Errorf("method %s not found in service %s", methodName, serviceName)
		}

		return methodDesc, nil
	}

	return nil, fmt.Errorf("service %s not found", serviceName)
}

var (
	grpcCmd = &cobra.Command{
		Use:   `grpc -d '{}' 127.0.0.1:50051 grpc.health.v1.Health/Check`,
		Short: "call grpc service",
		Long: `
call grpc service

Send an empty request: netool grpc grpc.server.com:443 my.custom.server.Service/Method
Send a request with a header and a body: netool grpc -H "Authorization: Bearer $token" -d '{"foo": "bar"}' grpc.server.com:443 my.custom.server.Service/Method
List all services exposed by a server: netool grpc grpc.server.com:443 list
List all methods in a particular service: netool grpc grpc.server.com:443 list my.custom.server.Service
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				log.Printf("you must specify the url")
				return
			}

			isGRPC, _ := cmd.Flags().GetBool("grpc")
			method, _ := cmd.Flags().GetString("method")

			targetURL, err := url.Parse(args[0])
			if err != nil {
				log.Printf("invalid url %s\n", args[0])
				return
			}

			fetcher := &fetcher{
				grpc:   isGRPC,
				method: method,
			}

			if err := fetcher.callHTTP2(targetURL); err != nil {
				log.Printf("Call HTTP/2 request %s failure: %v\n", targetURL, err)
				return
			}
		},
	}
)

func init() {
	base.AddSubCommands(grpcCmd)
}

func makeGenericGrpcCall(target, method, jsonData string) error {
	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	reflectionClient := reflectionpb.NewServerReflectionClient(conn)
	client := &grpcReflectClient{
		reflectionClient: reflectionClient,
	}

	serviceName, methodName, err := parseMethod(method)
	if err != nil {
		return err
	}

	methodDesc, err := client.findMethodDesc(serviceName, methodName)
	if err != nil {
		return err
	}

	inputDesc := methodDesc.Input()
	reqMessage := dynamicpb.NewMessage(inputDesc)
	err = protojson.Unmarshal([]byte(jsonData), reqMessage)
	if err != nil {
		return fmt.Errorf("failed to parse JSON to Protobuf: %v", err)
	}

	outputDesc := methodDesc.Output()
	resMessage := dynamicpb.NewMessage(outputDesc)

	ctx := context.Background()
	rpcMethod := fmt.Sprintf("/%s/%s", serviceName, methodName)
	err = conn.Invoke(ctx, rpcMethod, reqMessage, resMessage)
	if err != nil {
		return fmt.Errorf("gRPC call failed: %v", err)
	}

	responseJson, err := protojson.Marshal(resMessage)
	if err != nil {
		return fmt.Errorf("failed to serialize response to JSON: %v", err)
	}

	log.Printf("Response: %s", responseJson)
	return nil
}

func parseMethod(fullMethodName string) (string, string, error) {
	parts := strings.Split(fullMethodName, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid method format: %s", fullMethodName)
	}
	serviceName := parts[0]
	methodName := parts[1]
	return serviceName, methodName, nil
}
