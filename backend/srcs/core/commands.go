package core

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

func runAndLog(moduleID string, cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("StdoutPipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("StderrPipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cmd.Start: %w", err)
	}

	var wg sync.WaitGroup
	// used to wait that each pipe actually finished to read before exiting runAndLog
	wg.Add(2)
	defer wg.Wait()

	scanAndLog := func(r io.Reader, level string) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			LogModule(moduleID, level, line, nil)
		}
		if err := scanner.Err(); err != nil {
			if !errors.Is(err, os.ErrClosed) && !errors.Is(err, io.ErrClosedPipe) {
				LogModule(moduleID, "ERROR", fmt.Sprintf("reading pipe: %v", err), err)
			}
		}
	}

	go scanAndLog(stdout, "INFO")
	go scanAndLog(stderr, "WARN")

	return cmd.Wait()
}
