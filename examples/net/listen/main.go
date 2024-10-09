package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"syscall"
	"time"
)

func main() {
	lc := &net.ListenConfig{
		KeepAlive: time.Minute,
		KeepAliveConfig: net.KeepAliveConfig{
			Enable:   true,
			Idle:     15 * time.Second,
			Interval: 15 * time.Second,
			Count:    9,
		},
		Control: func(network, address string, c syscall.RawConn) error {
			log.Printf("Control: network: %s, addr: %s, conn: %v\n", network, address, c)
			return nil
		},
	}

	lis, err := lc.Listen(context.Background(), "tcp", ":8080")
	if err != nil {
		panic(err)
	}

	time.Sleep(30 * time.Second)
	fmt.Println(lis)
}
