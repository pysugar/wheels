package subcmds

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/pysugar/wheels/cmd/base"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	_ "google.golang.org/protobuf/types/known/anypb"
	_ "google.golang.org/protobuf/types/known/durationpb"
	_ "google.golang.org/protobuf/types/known/timestamppb"
	_ "google.golang.org/protobuf/types/known/wrapperspb"
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
			if len(args) < 2 {
				log.Printf("you must specify the url and operation")
				return
			}

			plaintextMode, _ := cmd.Flags().GetBool("plaintext")
			insecureMode, _ := cmd.Flags().GetBool("insecure")

			cred := insecure.NewCredentials()
			if !plaintextMode && insecureMode {
				cred = credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
			}

			target := args[0]
			op := args[1]
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if strings.EqualFold(op, "list") {
				if len(args) > 2 {
					serviceName := args[2]
					if err := listDescriptors(ctx, target, serviceName, grpc.WithTransportCredentials(cred)); err != nil {
						log.Printf("List descriptors error: %v\n", err)
					}
				} else {
					if err := listServices(ctx, target, grpc.WithTransportCredentials(cred)); err != nil {
						log.Printf("List services error: %v\n", err)
					}
				}
			}
		},
	}
)

func init() {
	grpcCmd.Flags().BoolP("plaintext", "p", false, "Use plain-text HTTP/2 when connecting to server (no TLS)")
	grpcCmd.Flags().BoolP("insecure", "i", false, "Skip server certificate and domain verification (skip TLS)")
	base.AddSubCommands(grpcCmd)
}

func listDescriptors(ctx context.Context, target, serviceName string, opts ...grpc.DialOption) error {
	clientStream, err := newReflectionClient(ctx, target, opts...)
	if err != nil {
		return err
	}

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		for {
			resp, er := clientStream.Recv()
			if er != nil {
				log.Printf("Failed to receive service reflection response: %v", er)
				return
			}

			if errResp := resp.GetErrorResponse(); errResp != nil {
				log.Printf("Failed to receive service reflection response: %v", errResp)
				continue
			}

			fileDescriptorResp := resp.GetFileDescriptorResponse()
			if fileDescriptorResp == nil {
				continue
			}

			for i, fdBytes := range fileDescriptorResp.FileDescriptorProto {
				fdProto := &descriptorpb.FileDescriptorProto{}
				if e := proto.Unmarshal(fdBytes, fdProto); e != nil {
					log.Printf("[%d] failed to unmarshal FileDescriptorProto: %v\n", i, e)
					continue
				}

				//protoregistry.GlobalTypes.RangeMessages(func(mt protoreflect.MessageType) bool {
				//	log.Printf("GlobalType: %v\n", mt.Descriptor().FullName())
				//	return true
				//})

				for _, dep := range fdProto.GetDependency() {
					log.Printf("Dependency: %s\n", dep)
				}

				fileDesc, e := protodesc.NewFile(fdProto, protoregistry.GlobalFiles)
				if e != nil {
					log.Printf("[%d] failed to create FileDescriptor from %v: %v\n", i, fdProto, e)
					return
				}

				for j := 0; j < fileDesc.Services().Len(); j++ {
					srv := fileDesc.Services().Get(j)
					log.Printf("[%d] %s\n", i, srv.FullName())
					for k := 0; k < srv.Methods().Len(); k++ {
						mth := srv.Methods().Get(k)
						log.Printf("[%d]\t %s\n", i, mth.FullName())
						log.Printf("[%d]\t\t %v\n", i, mth.Input().FullName())
						log.Printf("[%d]\t\t %v\n", i, mth.Output().FullName())
						log.Printf("[%d]\t\t stream client: %v\n", i, mth.IsStreamingClient())
						log.Printf("[%d]\t\t stream server: %v\n", i, mth.IsStreamingServer())
					}
				}
			}
			return
		}
	}()

	if er := clientStream.Send(&reflectionpb.ServerReflectionRequest{
		MessageRequest: &reflectionpb.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: serviceName,
		},
	}); er != nil {
		return fmt.Errorf("failed to send reflection request for service: %v", er)
	}

	<-doneCh
	return nil
}

func listServices(ctx context.Context, target string, opts ...grpc.DialOption) error {
	clientStream, err := newReflectionClient(ctx, target, opts...)
	if err != nil {
		return err
	}

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		for {
			resp, er := clientStream.Recv()
			if er != nil {
				log.Printf("Failed to receive service reflection response: %v", er)
				return
			}

			if errResp := resp.GetErrorResponse(); errResp != nil {
				log.Printf("Failed to receive service reflection response: %v", errResp)
				continue
			}

			listServicesResp := resp.GetListServicesResponse()
			if listServicesResp == nil {
				continue
			}

			for _, srv := range listServicesResp.GetService() {
				log.Printf("Discovered service: %v\n", srv.GetName())
			}
			return
		}
	}()

	if er := clientStream.Send(&reflectionpb.ServerReflectionRequest{
		MessageRequest: &reflectionpb.ServerReflectionRequest_ListServices{
			ListServices: "*",
		},
	}); er != nil {
		return fmt.Errorf("failed to send reflection request for service: %v", err)
	}

	<-doneCh
	return nil
}

func findMethodDescriptor(ctx context.Context, target, method string, opts ...grpc.DialOption) (protoreflect.MethodDescriptor, error) {
	clientStream, err := newReflectionClient(ctx, target, opts...)
	if err != nil {
		return nil, err
	}

	doneCh := make(chan struct{})
	<-doneCh
	return nil, nil
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

func newReflectionClient(ctx context.Context, target string, opts ...grpc.DialOption) (grpc.BidiStreamingClient[reflectionpb.ServerReflectionRequest, reflectionpb.ServerReflectionResponse], error) {
	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	reflectionClient := reflectionpb.NewServerReflectionClient(conn)
	clientStream, err := reflectionClient.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}
	return clientStream, nil
}
