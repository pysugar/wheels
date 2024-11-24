package main

import (
	"fmt"
	"google.golang.org/protobuf/encoding/protojson"
	"log"

	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
	"path/filepath"
)

func main() {
	protoPath := "/opt/webfs/lifework/projects/inspirations/grpc-proto/grpc/health/v1/health.proto"
	serviceName := "grpc.health.v1.Health"
	methodName := "Check"

	req, res, err := GetReqResMessages(protoPath, serviceName, methodName)
	if err != nil {
		panic(err)
	}

	err = protojson.Unmarshal([]byte(`{"service": "service1"}`), req)
	if err != nil {
		panic(fmt.Errorf("failed to parse JSON to Protobuf: %v", err))
	}
	err = protojson.Unmarshal([]byte(`{"status": "SERVING"}`), res)
	if err != nil {
		panic(fmt.Errorf("failed to parse JSON to Protobuf: %v", err))
	}
	fmt.Printf("req: %+v, res: %+v\n", req, res)

	fmt.Println(req.ProtoReflect().Descriptor().FullName())
	fmt.Println(res.ProtoReflect().Descriptor().FullName())
}

func GetReqResMessages(protoPath, serviceName, methodName string) (proto.Message, proto.Message, error) {
	parser := protoparse.Parser{
		ImportPaths: []string{filepath.Dir(protoPath)},
	}

	files, err := parser.ParseFiles(filepath.Base(protoPath))
	if err != nil {
		return nil, nil, fmt.Errorf("解析 proto 文件失败: %v", err)
	}

	log.Printf("dir: %+v", filepath.Dir(protoPath))
	log.Printf("base: %+v", filepath.Base(protoPath))
	for _, fd := range files {
		log.Printf("fd: %+v\n", fd)
	}

	if len(files) == 0 {
		return nil, nil, fmt.Errorf("未找到解析后的文件")
	}

	file := files[0]

	service := file.FindService(serviceName)
	if service == nil {
		return nil, nil, fmt.Errorf("未找到服务: %s", serviceName)
	}

	method := service.FindMethodByName(methodName)
	if method == nil {
		return nil, nil, fmt.Errorf("未找到方法: %s 在服务: %s 中", methodName, serviceName)
	}

	inputDesc := method.GetInputType()
	outputDesc := method.GetOutputType()

	if inputDesc == nil || outputDesc == nil {
		return nil, nil, fmt.Errorf("方法的输入或输出描述符为空")
	}

	reqMessage := dynamicpb.NewMessage(inputDesc.UnwrapMessage())
	resMessage := dynamicpb.NewMessage(outputDesc.UnwrapMessage())

	return reqMessage, resMessage, nil
}
