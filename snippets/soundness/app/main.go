package main

import (
	"context"
	"net/url"
	"soundness/agent"
	"time"
)

func main() {
	brokerURL, _ := url.Parse("http://localhost:5000")
	srv := agent.NewAgent(
		brokerURL,
		agent.WithHeartbeatInterval(5*time.Second),
		agent.WithCollectURL("/openapi"),
		agent.WithCustomHeaders(map[string]string{
			"X-Resource-Type": "znt-i",
			"X-Resource-Op":   "update",
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := srv.Start(ctx); err != nil {
		panic(err)
	}
}
