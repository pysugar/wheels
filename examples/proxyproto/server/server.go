package main

import (
	"fmt"
	"log"
	"net"

	"github.com/pires/go-proxyproto"
)

func main() {
	addr := "localhost:9876"
	list, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("couldn't listen to %q: %q\n", addr, err.Error())
	}

	// Wrap listener in a proxyproto listener
	proxyListener := &proxyproto.Listener{Listener: list}
	defer proxyListener.Close()

	// Wait for a connection and accept it
	conn, err := proxyListener.Accept()
	if err != nil {
		log.Fatalf("accept error: %v", err)
	}
	defer conn.Close()

	// Print connection details
	if conn.LocalAddr() == nil {
		log.Fatal("couldn't retrieve local address")
	}
	log.Printf("local address: %q", conn.LocalAddr().String())

	if conn.RemoteAddr() == nil {
		log.Fatal("couldn't retrieve remote address")
	}
	log.Printf("remote address: %q", conn.RemoteAddr().String())

	bs := make([]byte, 4096)
	n, err := conn.Read(bs)
	if err != nil {
		log.Fatalf("read error: %v", err)
	}
	fmt.Printf("%d: %s", n, string(bs[0:n]))
}
