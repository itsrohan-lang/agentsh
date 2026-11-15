//go:build windows

package ebpf

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Microsoft/go-winio"
)

const WFPDefaultPipe = `\\.\pipe\AgentshWFP`

// Interceptor manages the connection to the Windows Filtering Platform (WFP) service.
type Interceptor struct {
	pipeConn  net.Conn
	pipePath  string
	blocked   map[uint16]bool
	dropCount uint64
	mu        sync.Mutex
}

// ResolveContainerCgroup on Windows notes that cgroups are Linux-specific.
func ResolveContainerCgroup(containerIDOrName string) (string, error) {
	return "", errors.New("cgroups are native to Linux; on Windows, Docker containers run inside WSL2 or Hyper-V")
}

// AttachInterceptor connects to the Windows Filtering Platform service if running,
// or returns ErrUnsupported.
func AttachInterceptor(target string, blockedPorts ...uint16) (*Interceptor, error) {
	pipePath := WFPDefaultPipe

	conn, err := winio.DialPipe(pipePath, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: Windows WFP kernel filter driver is not active (pipe %s unreachable); install and start experimental/windows/src/wfp_service or run with --allow-unenforced-network", ErrUnsupported, pipePath)
	}

	interceptor := &Interceptor{
		pipeConn: conn,
		pipePath: pipePath,
		blocked:  make(map[uint16]bool),
	}

	ports := blockedPorts
	if len(ports) == 0 {
		ports = DefaultBlockedPorts
	}

	if err := interceptor.UpdateBlockedPorts(ports); err != nil {
		interceptor.Detach()
		return nil, fmt.Errorf("configuring WFP interceptor: %w", err)
	}

	return interceptor, nil
}

// AttachContainerInterceptor on Windows aliases AttachInterceptor.
func AttachContainerInterceptor(containerIDOrName string, blockedPorts ...uint16) (*Interceptor, error) {
	return AttachInterceptor(containerIDOrName, blockedPorts...)
}

// UpdateBlockedPorts configures blocked ports via the WFP service pipe.
func (i *Interceptor) UpdateBlockedPorts(ports []uint16) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.pipeConn == nil {
		return errors.New("not connected to WFP service")
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

	if _, err := i.pipeConn.Write(payload); err != nil {
		return fmt.Errorf("sending policy to WFP service: %w", err)
	}

	i.blocked = make(map[uint16]bool)
	for _, p := range ports {
		i.blocked[p] = true
	}

	return nil
}

// GetDropCount queries the WFP service for kernel-level dropped connections.
func (i *Interceptor) GetDropCount() (uint64, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.pipeConn == nil {
		return i.dropCount, nil
	}

	msg := map[string]string{"action": "get_stats"}
	payload, _ := json.Marshal(msg)
	payload = append(payload, '\n')

	if _, err := i.pipeConn.Write(payload); err != nil {
		return i.dropCount, nil
	}

	buf := make([]byte, 512)
	i.pipeConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	n, err := i.pipeConn.Read(buf)
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

// Detach closes the pipe to the WFP service.
func (i *Interceptor) Detach() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.pipeConn != nil {
		i.pipeConn.Close()
		i.pipeConn = nil
	}
	return nil
}
