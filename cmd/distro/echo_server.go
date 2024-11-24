package distro

import (
	"context"
	"log"

	pb "github.com/pysugar/wheels/grpc/proto"
	"github.com/pysugar/wheels/grpc/server"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type serverImp struct {
	pb.UnimplementedEchoServiceServer
}

func (s *serverImp) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	log.Printf("Received message from client: %s", req.Message)
	return &pb.EchoResponse{Message: req.Message}, nil
}

var echoServiceCmd = &cobra.Command{
	Use:   `echoservice -p 8080`,
	Short: "Start a gRPC echo service",
	Long: `
Start a gRPC echo service.

Start a gRPC echo service: netool echoservice --port=8080
`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		//verbose, _ := cmd.Flags().GetBool("verbose")
		err := server.StartGrpcServer(port, "echoservice", func(s *grpc.Server) {
			pb.RegisterEchoServiceServer(s, &serverImp{})
		})
		if err != nil {
			log.Fatal(err.Error())
		}
	},
}

func init() {
	echoServiceCmd.Flags().IntP("port", "p", 8080, "http proxy	 port")
	echoServiceCmd.Flags().BoolP("verbose", "V", false, "Verbose mode")
}
