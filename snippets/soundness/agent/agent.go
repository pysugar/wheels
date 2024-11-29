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
		brokerURL         *url.URL
		heartbeatInterval time.Duration
		heartbeatUrl      string
		collectorUrl      string
		statusFile        string
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
		brokerURL:         brokerURL,
		heartbeatInterval: o.heartbeatInterval,
		heartbeatUrl:      heartbeatUrl,
		collectorUrl:      collectorUrl,
		statusFile:        o.statusFile,
	}
}

func (o *agent) Start(ctx context.Context) error {
	watcher, er := fsnotify.NewWatcher()
	if er != nil {
		return er
	}
	defer watcher.Close()

	if err := watcher.Add(o.statusFile); err != nil {
		return err
	}

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
				o.loadAndSendJSON()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Println("file watch error:", err)
		case <-heartbeatTicker.C:
			o.sendHeartbeat(ctx)
		}
	}
}

func (o *agent) loadAndSendJSON() {
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

	req, err := http.NewRequest("POST", o.collectorUrl, bytes.NewBuffer(byteValue))
	if err != nil {
		log.Printf("Create request failure: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Send request failure: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Printf("Data send success, status: %v", resp.StatusCode)
}

func (o *agent) sendHeartbeat(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	log.Println("Send Heartbeat...")
	resp, err := http.Get(o.heartbeatUrl)
	if err != nil {
		log.Printf("Send Heartbeat Failure: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("Send Heartbeat Success: %v", resp.StatusCode)
}

func (o *agent) waitForCompleteWrite() {
	startTime := time.Now()
	for {
		if time.Since(startTime) > maxWaitTime {
			log.Println("等待文件写入超时")
			return
		}

		file, err := os.Open(o.statusFile)
		if err == nil {
			defer file.Close()
			if er := syscall.Flock(int(file.Fd()), syscall.LOCK_SH); er == nil {
				break
			} else {
				log.Printf("syscall.Flock error: %v", er)
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}
