package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	grpchealthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	var (
		serverAddr = flag.String("addr", "localhost:8443", "The server address in the format of host:port")
		certFile   = flag.String("cert", "../cert/server.crt", "TLS cert file")
	)
	flag.Parse()

	cert, err := ioutil.ReadFile(*certFile)
	if err != nil {
		log.Fatalf("Failed to read cert file: %v", err)
	}
	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(cert); !ok {
		log.Fatalf("Failed to append certs")
	}

	creds := credentials.NewTLS(&tls.Config{
		RootCAs: pool,
	})

	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()

	c := grpchealthv1.NewHealthClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.Check(ctx, &grpchealthv1.HealthCheckRequest{Service: ""})
	if err != nil {
		log.Fatalf("Could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.String())
}
