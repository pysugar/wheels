package agent

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

type (
	Agent interface {
		Start(context.Context) error
	}

	agent struct {
		agentID           string
		brokerURL         *url.URL
		heartbeatInterval time.Duration
		heartbeatUrl      string
		collectorUrl      string
		statusFile        string
		customHeaders     map[string]string
		fileLock          sync.Mutex
	}
)

const (
	maxWaitTime = 1 * time.Minute
)

func NewAgent(brokerURL *url.URL, opts ...Option) Agent {
	o := evaluateOptions(opts)
	brokerURL.Path = o.heartbeatPath
	heartbeatUrl := brokerURL.String()
	brokerURL.Path = o.collectPath
	collectorUrl := brokerURL.String()

	return &agent{
		agentID:           o.agentID,
		brokerURL:         brokerURL,
		heartbeatInterval: o.heartbeatInterval,
		heartbeatUrl:      heartbeatUrl,
		collectorUrl:      collectorUrl,
		statusFile:        o.statusFile,
		customHeaders:     o.customHeaders,
	}
}

func (o *agent) Start(ctx context.Context) error {
	watcher, er := fsnotify.NewWatcher()
	if er != nil {
		return er
	}
	defer watcher.Close()

	go o.watchStatusFile(ctx, watcher)

	heartbeatTicker := time.NewTicker(o.heartbeatInterval)
	defer heartbeatTicker.Stop()

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			log.Println("file watch event:", event)
			if event.Has(fsnotify.Write) {
				log.Println("modified file:", event.Name)
				o.waitForCompleteWrite()
				o.loadAndSendJSON(ctx)
			} else if event.Has(fsnotify.Remove) {
				log.Println("removed file:", event.Name)
				go o.watchStatusFile(ctx, watcher)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Println("file watch error:", err)
		case <-heartbeatTicker.C:
			o.sendHeartbeat(ctx)
		case <-ctx.Done():
			log.Println("Context cancelled, shutting down")
			return nil
		}
	}
}

func (o *agent) watchStatusFile(ctx context.Context, watcher *fsnotify.Watcher) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping watchStatusFile")
			return
		case <-ticker.C:
			if _, err := os.Stat(o.statusFile); !os.IsNotExist(err) {
				if err := watcher.Add(o.statusFile); err != nil {
					log.Printf("Can't add file to watcher: %v", err)
				} else {
					log.Printf("Add file to watcher...")
				}
				return
			}
		}
	}
}

func (o *agent) loadAndSendJSON(ctx context.Context) {
	o.fileLock.Lock()
	defer o.fileLock.Unlock()

	file, err := os.Open(o.statusFile)
	if err != nil {
		log.Printf("Can't open file: %v", err)
		return
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Can't read file content: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", o.collectorUrl, bytes.NewBuffer(byteValue))
	if err != nil {
		log.Printf("Create request failure: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range o.customHeaders {
		req.Header.Set(k, v)
	}
	if o.agentID != "" {
		req.Header.Set("X-Resource-Id", o.agentID)
		query := req.URL.Query()
		query.Set("name", o.agentID)
		req.URL.RawQuery = query.Encode()
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Send request failure: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Printf("Sync data(%s) success, status: %v", req.URL.RequestURI(), resp.Status)
}

func (o *agent) sendHeartbeat(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	log.Println("Send Heartbeat...")
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.heartbeatUrl, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		log.Printf("Create heartbeat request failure: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range o.customHeaders {
		req.Header.Set(k, v)
	}
	if o.agentID != "" {
		req.Header.Set("X-Resource-Id", o.agentID)
		query := req.URL.Query()
		query.Set("name", o.agentID)
		req.URL.RawQuery = query.Encode()
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Send Heartbeat Failure: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("Send Heartbeat(%s) Success: %v", req.URL.RequestURI(), resp.StatusCode)
}

func (o *agent) waitForCompleteWrite() {
	startTime := time.Now()
	for {
		if time.Since(startTime) > maxWaitTime {
			log.Println("write file timeout")
			return
		}

		file, err := os.Open(o.statusFile)
		if err == nil {
			if er := syscall.Flock(int(file.Fd()), syscall.LOCK_SH); er == nil {
				file.Close()
				break
			} else {
				log.Printf("syscall.Flock error: %v", er)
			}
			file.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
}
