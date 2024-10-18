package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	transport := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
		DisableKeepAlives:  false, // 启用 Keep-Alive
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	go startHeartbeat(client)

	// 主逻辑处理
	for {
		resp, err := client.Get("http://localhost:8080/")
		if err != nil {
			log.Printf("Failed to get response: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		resp.Body.Close()

		fmt.Println("Received response from server")
		time.Sleep(10 * time.Second)
	}
}

func startHeartbeat(client *http.Client) {
	ticker := time.NewTicker(10 * time.Second) // 心跳间隔
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 发送心跳请求
			req, err := http.NewRequest("GET", "http://localhost:8080/heartbeat", nil)
			if err != nil {
				log.Printf("Failed to create heartbeat request: %v", err)
				continue
			}
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Failed to send heartbeat: %v", err)
				continue
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Printf("Heartbeat response status: %v", resp.Status)
				continue
			}
			log.Println("Heartbeat successful")
		}
	}
}
