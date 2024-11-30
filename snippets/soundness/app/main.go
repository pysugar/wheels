package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"soundness/agent"
)

func main() {
	rawURL, has := os.LookupEnv("BROKER_URL")
	if !has || rawURL == "" {
		panic("BROKER_URL environment variable not set")
	}
	brokerURL, err := url.Parse(rawURL)
	if err != nil {
		panic(fmt.Errorf("failed to parse BROKER_URL: %v", err))
	}

	agentID := os.Getenv("AGENT_NAME")
	if agentID == "" {
		panic("AGENT_NAME environment variable not set")
	}

	options := []agent.Option{
		agent.WithCustomHeaders(map[string]string{
			"X-Resource-Type": "znt-i",
			"X-Resource-Op":   "update",
		}),
		agent.WithAgentID(agentID),
	}

	if v, ok := os.LookupEnv("HEARTBEAT_INTERVAL"); ok {
		if interval, er := strconv.Atoi(v); er == nil {
			options = append(options, agent.WithHeartbeatInterval(time.Duration(interval)*time.Second))
		}
	}

	if v, ok := os.LookupEnv("STATUS_PATH"); ok && v != "" {
		options = append(options, agent.WithCollectURL(v))
		options = append(options, agent.WithHeartbeatPath(v))
	}

	srv := agent.NewAgent(brokerURL, options...)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if er := srv.Start(ctx); er != nil {
		panic(er)
	}
}
