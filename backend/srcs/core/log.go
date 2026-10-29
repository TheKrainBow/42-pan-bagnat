package core

import (
	"backend/database"
	"backend/websocket"
	"fmt"
	"sync"
	"time"
)

var (
	logTapsMu sync.Mutex
	logTaps   = map[string]map[chan string]struct{}{}
)

// tapModuleLogs streams the module's log lines ("[LEVEL] message") until the
// returned stop function is called. Lines are dropped if the reader lags.
func tapModuleLogs(moduleID string) (<-chan string, func()) {
	ch := make(chan string, 256)
	logTapsMu.Lock()
	if logTaps[moduleID] == nil {
		logTaps[moduleID] = map[chan string]struct{}{}
	}
	logTaps[moduleID][ch] = struct{}{}
	logTapsMu.Unlock()
	return ch, func() {
		logTapsMu.Lock()
		delete(logTaps[moduleID], ch)
		if len(logTaps[moduleID]) == 0 {
			delete(logTaps, moduleID)
		}
		logTapsMu.Unlock()
	}
}

func notifyLogTaps(moduleID, level, message string) {
	logTapsMu.Lock()
	defer logTapsMu.Unlock()
	for ch := range logTaps[moduleID] {
		select {
		case ch <- "[" + level + "] " + message:
		default:
		}
	}
}

func LogModule(moduleID, level, message string, meta map[string]any, err error) error {
	// Build the JSON meta payload
	if meta == nil {
		meta = make(map[string]any, 1)
	}

	if err != nil {
		meta["error"] = err.Error()
	}

	log, _ := database.InsertModuleLog(database.ModuleLog{
		ModuleID: moduleID,
		Level:    level,
		Message:  message,
		Meta:     meta,
	})

	ts := time.Now().Format(time.RFC3339)
	if level == "ERROR" {
		if err != nil {
			fmt.Printf("%s [%s] [module:%s] %s – error: %v\n",
				ts, level, moduleID, message, err,
			)
		} else {
			fmt.Printf("%s [%s] [module:%s] %s\n",
				ts, level, moduleID, message,
			)
		}
	}

	websocket.SendLogEvent(log)
	notifyLogTaps(moduleID, level, message)

	if err != nil {
		return fmt.Errorf("%s: %w", message, err)
	}
	return nil
}
