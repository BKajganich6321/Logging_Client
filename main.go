/*
FILE:          main.go
PROJECT:       SENG2045 Assignment 2 - Logging Service Test Client
PROGRAMMER:    Brad Kajganich
FIRST VERSION: 2026-10-17
DESCRIPTION:   Command-line test client for the logging server:
                 - interactive: type lines, see the server's replies
                 - one-shot:    -msg "text" sends a single message and exits

USAGE:         logclient [-addr host:port] [-name client_name]
                         [-level LEVEL -msg "text"]
*/

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	os.Exit(run())
}

// run does the real work and returns the process exit code. It is separate
// from main so that deferred cleanup (closing the connection) always runs.
func run() int {
	addr := flag.String("addr", "127.0.0.1:13300", "server address as host:port")
	name := flag.String("name", "goclient", "client name sent in the HELLO handshake")
	level := flag.String("level", "INFO", "level to use with -msg")
	msg := flag.String("msg", "", "send this one message and exit (omit for interactive mode)")
	flag.Parse()

	session, err := Connect(*addr, *name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect failed:", err)
		return 1
	}
	defer session.Close()

	if *msg != "" {
		return oneShot(session, *level, *msg)
	}
	return interactive(session, *addr)
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
// and deliberately invalid input can be tried. Type "quit" or press Ctrl-D
// (Ctrl-Z then Enter on Windows) to leave.
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
