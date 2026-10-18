/*
FILE:          modes.go
PROJECT:       SENG2045 Assignment 2 - Logging Service Test Client
PROGRAMMER:    Brad Kajganich
FIRST VERSION: 2026-10-17
DESCRIPTION:   The automated test modes of the client:
                 multi - several clients connected at the same time, each
                         sending a stream of messages at all five levels
                 abuse - one client floods the server to check that the
                         rate limiter engages and disconnects it
*/

package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// levels are the five log levels the server understands.
var levels = []string{"DEBUG", "INFO", "WARN", "ERROR", "CRITICAL"}

// clientResult is what one multi-mode client reports when it finishes.
type clientResult struct {
	name    string
	sent    int
	replies map[string]int
	err     error
}

// runMulti starts `clients` simultaneous clients named <baseName>-1,
// <baseName>-2 ... Each sends `count` messages, cycling through all levels,
// pausing `delay` between messages. It prints a per-client summary and
// returns 0 only if every client received nothing but OK and had no errors.
func runMulti(addr, baseName string, clients, count int, delay time.Duration) int {
	fmt.Printf("Starting %d simultaneous clients, %d messages each...\n", clients, count)

	results := make([]clientResult, clients)
	var wg sync.WaitGroup
	for i := 0; i < clients; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("%s-%d", baseName, i+1)
			results[i] = multiWorker(addr, name, count, delay)
		}(i) // each goroutine writes only its own slot, so no lock is needed
	}
	wg.Wait()

	exitCode := 0
	for _, r := range results {
		status := formatReplies(r.replies)
		if r.err != nil {
			status += "  ERROR: " + r.err.Error()
			exitCode = 1
		}
		if r.replies["OK"] != count {
			exitCode = 1
		}
		fmt.Printf("  %-16s sent %-4d %s\n", r.name, r.sent, status)
	}
	if exitCode == 0 {
		fmt.Println("PASS: every client got OK for every message")
	} else {
		fmt.Println("FAIL: at least one client did not get OK for every message")
	}
	return exitCode
}

// multiWorker is one client of runMulti.
func multiWorker(addr, name string, count int, delay time.Duration) clientResult {
	result := clientResult{name: name, replies: map[string]int{}}

	session, err := Connect(addr, name)
	if err != nil {
		result.err = err
		return result
	}
	defer session.Close()

	for j := 0; j < count; j++ {
		level := levels[j%len(levels)]
		line := fmt.Sprintf("%s %s message %d of %d", level, name, j+1, count)
		reply, err := session.Send(line)
		if err != nil {
			result.err = err
			return result
		}
		result.sent++
		result.replies[reply]++
		time.Sleep(delay)
	}
	return result
}

// runAbuse sends `count` messages as fast as possible from one client and
// reports how the server reacted. It returns 0 if the rate limiter engaged
// (at least one ERR rate_limited), and 1 if the flood was never limited.
func runAbuse(addr, name string, count int) int {
	session, err := Connect(addr, name)
	if err != nil {
		fmt.Println("connect failed:", err)
		return 1
	}
	defer session.Close()

	fmt.Printf("Flooding the server with %d messages as %q...\n", count, name)
	replies := map[string]int{}
	closedAfter := -1 // message number during which the server hung up

	for i := 1; i <= count; i++ {
		reply, err := session.Send(fmt.Sprintf("INFO flood message %d", i))
		if err != nil {
			closedAfter = i
			fmt.Printf("  server closed the connection while sending message %d (%v)\n", i, err)
			break
		}
		replies[reply]++
	}

	fmt.Println("  replies:", formatReplies(replies))
	if closedAfter > 0 {
		fmt.Printf("  disconnected after %d accepted/answered messages\n", closedAfter-1)
	} else {
		fmt.Println("  server did not disconnect the client")
	}

	if replies["ERR rate_limited"] == 0 {
		fmt.Println("FAIL: the rate limiter never engaged")
		return 1
	}
	fmt.Println("PASS: the rate limiter engaged")
	return 0
}

// formatReplies turns a reply tally into e.g. "OK=20  ERR rate_limited=5",
// with the keys sorted so output is stable from run to run.
func formatReplies(replies map[string]int) string {
	keys := make([]string, 0, len(replies))
	for key := range replies {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, replies[key]))
	}
	return strings.Join(parts, "  ")
}
