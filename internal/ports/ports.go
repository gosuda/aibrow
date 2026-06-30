// Package ports implements a simple TCP port availability scanner.
// It is used by the AIBrow server and Swagger UI server to pick the next
// free port when the default port is already in use.
package ports

import (
	"fmt"
	"net"
)

// FindAvailable returns the first port starting at start that can be listened on.
// It tries up to maxAttempts ports to avoid an infinite loop.
func FindAvailable(start, maxAttempts int) (int, error) {
	for i := 0; i < maxAttempts; i++ {
		port := start + i
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			_ = ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("ports: no available port found between %d and %d", start, start+maxAttempts-1)
}
