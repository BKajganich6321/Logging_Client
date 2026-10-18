/*
FILE:          main.go
PROJECT:       SENG2045 Assignment 2 - Logging Service Test Client
PROGRAMMER:    Brad Kajganich
FIRST VERSION: 2026-10-17
DESCRIPTION:   Command-line test client for the logging server. Modes:
                 manual (default) - interactive typing, or -msg for one message
                 multi            - N simultaneous clients (-clients, -count)
                 abuse            - flood the server to test rate limiting
               The automated test suite is added in a later step.

USAGE:         logclient [-addr host:port] [-name client_name]
                         [-mode manual|multi|abuse]
                         manual: [-level LEVEL -msg "text"]
                         multi:  [-clients N] [-count M] [-delay 100ms]
                         abuse:  [-count M]

*/

*/

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	os.Exit(run())
}

// run does the real work and returns the process exit code. It is separate
// from main so that deferred cleanup (closing connections) always runs.
func run() int {
	addr := flag.String("addr", "127.0.0.1:13300", "server address as host:port")
	name := flag.String("name", "goclient", "client name sent in the HELLO handshake")
	mode := flag.String("mode", "manual", "manual, multi or abuse")
	level := flag.String("level", "INFO", "manual mode: level to use with -msg")
	msg := flag.String("msg", "", "manual mode: send this one message and exit (omit for interactive)")
	clients := flag.Int("clients", 5, "multi mode: number of simultaneous clients")
	count := flag.Int("count", 0, "multi/abuse mode: messages per client (default 5 for multi, 100 for abuse)")
	delay := flag.Duration("delay", 100*time.Millisecond, "multi mode: pause between a client's messages")
	flag.Parse()

	switch *mode {
	case "manual":
		return runManual(*addr, *name, *level, *msg)
	case "multi":
		return runMulti(*addr, *name, *clients, defaultCount(*count, 5), *delay)
	case "abuse":
		return runAbuse(*addr, *name, defaultCount(*count, 100))
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q (use manual, multi or abuse)\n", *mode)
		return 2
	}
}

// defaultCount returns value, or fallback when value was left at 0.
func defaultCount(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

// runManual connects once, then either sends the single -msg or goes
// interactive.
func runManual(addr, name, level, msg string) int {
	session, err := Connect(addr, name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect failed:", err)
		return 1
	}
	defer session.Close()

	if msg != "" {
		return oneShot(session, level, msg)
	}
	return interactive(session, addr)
}

// oneShot sends a single "<LEVEL> <message>" line and prints the reply.
// The exit code is 0 only if the server answered OK, so scripts can check it.
func oneShot(session *Session, level, msg string) int {
	reply, err := session.Send(level + " " + msg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "send failed:", err)
		return 1
	}
	fmt.Println(reply)
	if reply != "OK" {
		return 1
	}
	return 0
}

// interactive sends every line typed on stdin exactly as typed, so both valid
// and deliberately invalid input can be tried. 
func interactive(session *Session, addr string) int {
	fmt.Printf("Connected to %s as %q.\n", addr, session.Name)
	fmt.Println(`Type lines like: INFO hello world   (or "quit" to exit)`)

	input := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !input.Scan() {
			fmt.Println()
			return 0
		}
		line := input.Text()
		if strings.TrimSpace(line) == "quit" {
			return 0
		}

		reply, err := session.Send(line)
		if err != nil {
			fmt.Fprintln(os.Stderr, "connection closed by server:", err)
			return 1
		}
		fmt.Println(reply)
	}
}
