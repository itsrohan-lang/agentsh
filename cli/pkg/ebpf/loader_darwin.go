//go:build darwin

package ebpf

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

const ESFDefaultSocket = "/var/run/agentsh_esf.sock"

// Interceptor manages the connection to the native macOS Endpoint Security (ESF) daemon.
type Interceptor struct {
	conn       net.Conn
	socketPath string
	blocked    map[uint16]bool
	dropCount  uint64
	mu         sync.Mutex
}

// ResolveContainerCgroup on macOS informs the caller that container cgroups reside inside the Linux VM.
func ResolveContainerCgroup(containerIDOrName string) (string, error) {
	return "", errors.New("cgroups are native to Linux; on macOS, Docker containers run within the hypervisor VM")
}

// AttachInterceptor connects to the native macOS EndpointSecurity helper daemon if running,
// or returns ErrUnsupported with actionable instructions.
func AttachInterceptor(target string, blockedPorts ...uint16) (*Interceptor, error) {
	socketPath := os.Getenv("AGENTSH_ESF_SOCKET")
	if socketPath == "" {
		socketPath = ESFDefaultSocket
	}

	// Check if the native ESF daemon is active
	conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("%w: macOS native EndpointSecurity daemon is not running on %s; build and launch experimental/mac/src/esf_interceptor as root or run with --allow-unenforced-network", ErrUnsupported, socketPath)
	}

	interceptor := &Interceptor{
		conn:       conn,
		socketPath: socketPath,
		blocked:    make(map[uint16]bool),
	}

	ports := blockedPorts
	if len(ports) == 0 {
		ports = DefaultBlockedPorts
	}

	if err := interceptor.UpdateBlockedPorts(ports); err != nil {
		interceptor.Detach()
		return nil, fmt.Errorf("configuring ESF interceptor: %w", err)
	}

	return interceptor, nil
}

// AttachContainerInterceptor on macOS aliases AttachInterceptor.
func AttachContainerInterceptor(containerIDOrName string, blockedPorts ...uint16) (*Interceptor, error) {
	return AttachInterceptor(containerIDOrName, blockedPorts...)
}

// UpdateBlockedPorts transmits policy configuration to the native ESF interceptor.
func (i *Interceptor) UpdateBlockedPorts(ports []uint16) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.conn == nil {
		return errors.New("not connected to ESF daemon")
	}

	msg := map[string]interface{}{
		"action":        "set_blocked_ports",
		"blocked_ports": ports,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	if _, err := i.conn.Write(payload); err != nil {
		return fmt.Errorf("sending policy to ESF daemon: %w", err)
	}

	i.blocked = make(map[uint16]bool)
	for _, p := range ports {
		i.blocked[p] = true
	}

	return nil
}

// GetDropCount queries the ESF daemon for the total number of blocked connections.
func (i *Interceptor) GetDropCount() (uint64, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.conn == nil {
		return i.dropCount, nil
	}

	msg := map[string]string{"action": "get_stats"}
	payload, _ := json.Marshal(msg)
	payload = append(payload, '\n')

	if _, err := i.conn.Write(payload); err != nil {
		return i.dropCount, nil
	}

	// Read response
	buf := make([]byte, 512)
	i.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	n, err := i.conn.Read(buf)
	if err == nil {
		var resp struct {
			DropCount uint64 `json:"drop_count"`
		}
		if json.Unmarshal(buf[:n], &resp) == nil {
			i.dropCount = resp.DropCount
		}
	}

	return i.dropCount, nil
}

// Detach disconnects from the ESF helper daemon.
func (i *Interceptor) Detach() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.conn != nil {
		i.conn.Close()
		i.conn = nil
	}
	return nil
}
