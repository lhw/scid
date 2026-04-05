// healthcheck is a minimal TCP probe binary for use in distroless containers.
// It dials the port the companion server listens on and exits 0 on success.
// Reads LISTEN_ADDR from the environment (default ":8080").
package main

import (
	"net"
	"os"
)

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		os.Exit(1)
	}

	c, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		os.Exit(1)
	}
	c.Close()
}
