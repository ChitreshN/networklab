package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Error starting TCP listener: %v", err)
	}
	log.Println("Proxy server listening on :8080")

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(clientConn)
	}
}

func handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	reader := bufio.NewReader(clientConn)

	firstLine, err := reader.ReadString('\n')
	if err != nil {
		log.Println("Error reading request:", err)
		return
	}
	log.Printf(">>> %s", strings.TrimSpace(firstLine))

	tokens := strings.Split(firstLine, " ")
	if len(tokens) < 3 {
		log.Println("Malformed request line:", firstLine)
		return
	}

	method := tokens[0]
	dest := tokens[1]

	if method == "CONNECT" {
		handleConnect(clientConn, dest)
	} else {
		handleHTTP(clientConn, method, dest, reader)
	}
}

func handleConnect(clientConn net.Conn, dest string) {
	if !strings.Contains(dest, ":") { dest += ":443" }

	serverConn, err := net.Dial("tcp", dest)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", dest, err)
		fmt.Fprintf(clientConn, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
		return
	}
	defer serverConn.Close()

	fmt.Fprintf(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")

	go io.Copy(serverConn, clientConn)
	io.Copy(clientConn, serverConn)
}

func handleHTTP(clientConn net.Conn, method, dest string, reader *bufio.Reader) {
	var host string
	if strings.HasPrefix(dest, "http://") {
		dest = strings.TrimPrefix(dest, "http://")
	}

	if idx := strings.Index(dest, "/"); idx != -1 {
		host = dest[:idx]
		dest = dest[idx:]
	} else {
		host = dest
		dest = "/"
	}

	if !strings.Contains(host, ":") {
		host += ":80" 
	}

	serverConn, err := net.Dial("tcp", host)
	if err != nil {
		log.Printf("Failed to connect to host %s: %v", host, err)
		fmt.Fprintf(clientConn, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
		return
	}
	defer serverConn.Close()

	fmt.Fprintf(serverConn, "%s %s HTTP/1.0\r\n", method, dest) 
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 { 
			break
		}
		if strings.HasPrefix(line, "Proxy-Connection:") {
			fmt.Fprintf(serverConn, "Connection: close\r\n")
		} else {
			fmt.Fprintf(serverConn, "%s\r\n", line)
		}
	}
	fmt.Fprintf(serverConn, "\r\n") 

	io.Copy(clientConn, serverConn)
}

