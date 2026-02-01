package main

import (
	"context"
	"io"
	"log"
	"net"
	"os"

	"tailscale.com/tsnet"
)

func main() {
	authKey := os.Getenv("TS_AUTHKEY")
	if authKey == "" {
		log.Println("Warning: TS_AUTHKEY environment variable not set. Assuming pre-authenticated or interactive flow.")
	}

	// Create a new tsnet server.
	// We use "grey-link-poc-linux" as the hostname.
	// State will be stored in "grey-link-poc-linux-state" directory in current CWD by default if not configured.
	srv := &tsnet.Server{
		Hostname: "grey-link-poc-linux",
		AuthKey:  authKey,
	}
	defer srv.Close()

	// Listen on a TCP port on the tailnet
	listener, err := srv.Listen("tcp", ":9999")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Get the LocalClient to retrieve status
	lc, err := srv.LocalClient()
	if err != nil {
		log.Printf("Warning: Failed to get local client: %v", err)
	} else {
		status, err := lc.StatusWithoutPeers(context.Background())
		if err != nil {
			log.Printf("Warning: Failed to get status: %v", err)
		} else {
			log.Printf("Listening on %s:9999", status.TailscaleIPs)
		}
	}
	
	log.Println("PoC Linux Server started. Waiting for connections...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(c net.Conn) {
	defer c.Close()
	log.Printf("New connection from %s", c.RemoteAddr())
	
	// Echo back everything
	buf := make([]byte, 1024)
	for {
		n, err := c.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error: %v", err)
			}
			return
		}
		
		log.Printf("Received %d bytes: %s", n, string(buf[:n]))
		
		_, err = c.Write(buf[:n])
		if err != nil {
			log.Printf("Write error: %v", err)
			return
		}
	}
}
