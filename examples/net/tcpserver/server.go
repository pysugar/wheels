package main

import (
	"net"
	"time"
)

func main() {
	listener, err := net.Listen("tcp", ":9876")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	time.Sleep(10 * time.Second)
}
