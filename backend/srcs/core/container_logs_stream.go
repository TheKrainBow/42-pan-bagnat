package core

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"backend/websocket"

	dockerTypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockerfilters "github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
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
	cli, err := getDockerClient()
	if err != nil {
		return
	}
	var sinceCursor string
	ref, ok := waitForComposeContainer(ctx, cli, module, containerName, time.Time{})
	if !ok {
		return
	}
	for {
		nextSince, err := followContainerLogs(ctx, cli, module.ID, containerName, ref.ID, sinceCursor)
		sinceCursor = nextSince
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			return
		}
		fmt.Printf("[Logs] container %s/%s log stream interrupted: %v\n", module.Slug, containerName, err)
		fmt.Printf("[Logs] waiting for %s/%s to restart after %s\n", module.Slug, containerName, ref.StartedAt.Format(time.RFC3339Nano))
		ref, ok = waitForComposeContainer(ctx, cli, module, containerName, ref.StartedAt)
		if !ok {
			return
		}
		if sinceCursor == "" && !ref.StartedAt.IsZero() {
			sinceCursor = ref.StartedAt.Format(time.RFC3339Nano)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
		}
	}
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

func followContainerLogs(ctx context.Context, cli *dockerclient.Client, moduleID, containerName, containerID, sinceCursor string) (string, error) {
	opts := dockercontainer.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
		Tail:       "40",
	}
	if sinceCursor != "" {
		opts.Since = sinceCursor
		opts.Tail = "0"
	}
	reader, err := cli.ContainerLogs(ctx, containerID, opts)
	if err != nil {
		fmt.Printf("[Logs] container %s/%s ContainerLogs error: %v\n", moduleID, containerName, err)
		return sinceCursor, err
	}
	defer reader.Close()

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		if _, err := stdcopy.StdCopy(pw, pw, reader); err != nil && !errors.Is(err, io.EOF) {
			pw.CloseWithError(err)
		}
	}()

	scanner := bufio.NewScanner(pr)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	nextSince := sinceCursor
	for {
		select {
		case <-ctx.Done():
			return nextSince, ctx.Err()
		default:
		}
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) && ctx.Err() == nil {
				return nextSince, err
			}
			if ctx.Err() != nil {
				return nextSince, ctx.Err()
			}
			return nextSince, errors.New("log stream ended")
		}
		line := scanner.Text()
		ts, msg := splitDockerLogLine(line)
		if parsed, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			nextSince = parsed.Format(time.RFC3339Nano)
		} else if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			nextSince = parsed.Format(time.RFC3339Nano)
		} else {
			nextSince = ts
		}
		websocket.SendContainerLogEvent(moduleID, containerName, ts, msg)
	}
}

type composeContainerRef struct {
	ID        string
	Name      string
	State     string
	StartedAt time.Time
}

func (ref composeContainerRef) Running() bool {
	return strings.EqualFold(strings.TrimSpace(ref.State), "running")
}

func waitForComposeContainer(ctx context.Context, cli *dockerclient.Client, module Module, service string, minStart time.Time) (composeContainerRef, bool) {
	if ref, ok := findComposeContainer(ctx, cli, module, service); ok && ref.Running() && (minStart.IsZero() || ref.StartedAt.After(minStart)) {
		return ref, true
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return composeContainerRef{}, false
		}
		select {
		case <-ctx.Done():
			return composeContainerRef{}, false
		case <-ticker.C:
			if ref, ok := findComposeContainer(ctx, cli, module, service); ok && ref.Running() && (minStart.IsZero() || ref.StartedAt.After(minStart)) {
				return ref, true
			}
		}
	}
}

func findComposeContainer(ctx context.Context, cli *dockerclient.Client, module Module, service string) (composeContainerRef, bool) {
	filter := dockerfilters.NewArgs()
	slug := strings.TrimSpace(module.Slug)
	if slug != "" {
		filter.Add("label", fmt.Sprintf("com.docker.compose.project=%s", slug))
	}
	service = strings.TrimSpace(service)
	if service != "" {
		filter.Add("label", fmt.Sprintf("com.docker.compose.service=%s", service))
	}
	containers, err := cli.ContainerList(ctx, dockercontainer.ListOptions{All: true, Filters: filter})
	if err != nil || len(containers) == 0 {
		return composeContainerRef{}, false
	}
	var chosen *dockercontainer.Summary
	isRunning := func(c *dockercontainer.Summary) bool {
		return c != nil && strings.EqualFold(strings.TrimSpace(c.State), "running")
	}
	for idx := range containers {
		c := &containers[idx]
		if chosen == nil {
			chosen = c
			continue
		}
		if isRunning(c) {
			if !isRunning(chosen) || c.Created > chosen.Created {
				chosen = c
			}
			continue
		}
		if !isRunning(chosen) && c.Created > chosen.Created {
			chosen = c
		}
	}
	if chosen == nil {
		return composeContainerRef{}, false
	}
	name := firstDockerName(chosen.Names)
	if name == "" {
		name = chosen.ID
	}
	details, err := cli.ContainerInspect(ctx, chosen.ID)
	if err != nil {
		return composeContainerRef{}, false
	}
	state := chosen.State
	if details.State != nil && details.State.Status != "" {
		state = details.State.Status
	}
	startedAt := parseDockerTimestamp(details)
	return composeContainerRef{ID: chosen.ID, Name: name, State: state, StartedAt: startedAt}, true
}


func parseDockerTimestamp(details dockerTypes.ContainerJSON) time.Time {
	if details.State == nil {
		return time.Time{}
	}
	start := strings.TrimSpace(details.State.StartedAt)
	if start == "" {
		return time.Time{}
	}
	if ts, err := time.Parse(time.RFC3339Nano, start); err == nil {
		return ts
	}
	if ts, err := time.Parse(time.RFC3339, start); err == nil {
		return ts
	}
	return time.Time{}
}
