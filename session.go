/*
FILE:          session.go
PROJECT:       SENG2045 Assignment 2 - Logging Service Test Client
PROGRAMMER:    Brad Kajganich
FIRST VERSION: 2026-10-17
DESCRIPTION:   A Session is one connection to the logging server. It sends 
				one line at a time, returning the server's one-line reply. 
				Every mode of the client (manual, multi-client, abuse, 
				automated tests) is built on this type.
*/

package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// ioTimeout limits how long we wait for the server to connect or reply.
const ioTimeout = 5 * time.Second

// Session is a connected, handshaken client.
type Session struct {
	conn   net.Conn
	reader *bufio.Reader
	Name   string
}

// Connect opens a TCP connection to addr and performs the HELLO handshake.
// It returns an error if the server is unreachable or rejects the handshake.
func Connect(addr, name string) (*Session, error) {
	conn, err := net.DialTimeout("tcp", addr, ioTimeout)
	if err != nil {
		return nil, err
	}

	s := &Session{conn: conn, reader: bufio.NewReader(conn), Name: name}
	reply, err := s.Send("HELLO " + name)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if reply != "OK" {
		conn.Close()
		return nil, fmt.Errorf("handshake rejected by server: %s", reply)
	}
	return s, nil
}

// Send writes one line (a newline is added) and returns the server's reply
// without its trailing newline. The line is sent exactly as given, so tests
// can deliberately send malformed input.
func (s *Session) Send(line string) (string, error) {
	if err := s.conn.SetDeadline(time.Now().Add(ioTimeout)); err != nil {
		return "", err
	}
	if _, err := s.conn.Write([]byte(line + "\n")); err != nil {
		return "", err
	}
	reply, err := s.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(reply, "\r\n"), nil
}

// Close ends the connection.
func (s *Session) Close() error {
	return s.conn.Close()
}
