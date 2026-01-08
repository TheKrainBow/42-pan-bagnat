package core

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"backend/websocket"
)

// Manager for per-container docker logs -f streamers, keyed by topic string
// topic format: "container:<moduleID>:<containerName>"

var (
	streamMu      sync.Mutex
	streamCancels = make(map[string]context.CancelFunc)
)

// WSOnSubscribe is wired from main.go to react when WS clients subscribe to topics.
// When the first subscriber for a container topic arrives, start streaming.
func WSOnSubscribe(topic string, count int) {
	if !strings.HasPrefix(topic, "container:") {
		return
	}
	if count != 1 {
		// already streaming (or multiple subs), nothing to do
		return
	}
	parts := strings.SplitN(strings.TrimPrefix(topic, "container:"), ":", 2)
	if len(parts) != 2 {
		return
	}
	moduleID := parts[0]
	containerName := parts[1]

	// Look up module to resolve slug and verify existence
	module, err := GetModule(moduleID)
	if err != nil || module.ID == "" {
		return
	}
	ensureContainerLogStreamer(topic, module, containerName)
}

// WSOnUnsubscribe stops streaming when the last subscriber leaves.
func WSOnUnsubscribe(topic string, count int) {
	if !strings.HasPrefix(topic, "container:") {
		return
	}
	if count != 0 {
		return
	}
	// last subscriber left → cancel streamer
	streamMu.Lock()
	cancel, ok := streamCancels[topic]
	if ok {
		cancel()
		delete(streamCancels, topic)
	}
	streamMu.Unlock()
}

func ensureContainerLogStreamer(topic string, module Module, containerName string) {
	streamMu.Lock()
	if _, exists := streamCancels[topic]; exists {
		streamMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	streamCancels[topic] = cancel
	streamMu.Unlock()

	go streamContainerLogs(ctx, module, containerName)
}

func streamContainerLogs(ctx context.Context, module Module, containerName string) {
	// Full container name matches other docker helpers
	fullName := fmt.Sprintf("%s-%s-1", module.Slug, containerName)
	// Use timestamps for proper [Date] [Time]
	cmd := exec.CommandContext(ctx, "docker", "logs", "--timestamps", "--follow", fullName)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	cmd.Stderr = cmd.Stdout // merge

	if err := cmd.Start(); err != nil {
		return
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			return
		default:
		}
		line := scanner.Text()
		// Expect format: "<ts> <message>" with ts RFC3339-like
		ts, msg := splitDockerLogLine(line)
		websocket.SendContainerLogEvent(module.ID, containerName, ts, msg)
	}
	// exit or error
	_ = cmd.Wait()
}

func splitDockerLogLine(line string) (string, string) {
	// docker --timestamps prefixes with RFC3339Nano timestamp followed by space
	idx := strings.IndexByte(line, ' ')
	if idx <= 0 {
		return time.Now().UTC().Format(time.RFC3339), line
	}
	ts := strings.TrimSpace(line[:idx])
	msg := strings.TrimSpace(line[idx+1:])
	return ts, msg
}
