//go:build linux

package ebpf

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

// Interceptor holds the loaded eBPF collection and active cgroup v2 link.
type Interceptor struct {
	coll       *ebpf.Collection
	link       link.Link
	cgroupPath string
	blocked    map[uint16]bool
	mu         sync.Mutex
}

// ResolveContainerCgroup locates the cgroup v2 path for a given Docker container ID or name.
func ResolveContainerCgroup(containerIDOrName string) (string, error) {
	if containerIDOrName == "" {
		return "", errors.New("container ID or name is required")
	}

	// 1. If an explicit absolute path was passed, verify it directly
	if strings.HasPrefix(containerIDOrName, "/sys/fs/cgroup") {
		if fi, err := os.Stat(containerIDOrName); err == nil && fi.IsDir() {
			return containerIDOrName, nil
		}
	}

	// 2. Query Docker for container PID
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Pid}}", containerIDOrName).Output()
	if err == nil {
		pidStr := strings.TrimSpace(string(out))
		if pidStr != "" && pidStr != "0" {
			// Read /proc/<pid>/cgroup for the cgroup v2 path
			cgroupFile := fmt.Sprintf("/proc/%s/cgroup", pidStr)
			if f, err := os.Open(cgroupFile); err == nil {
				defer f.Close()
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := scanner.Text()
					// cgroup v2 format: "0::<path>"
					if strings.HasPrefix(line, "0::") {
						rel := strings.TrimPrefix(line, "0::")
						candidate := filepath.Join("/sys/fs/cgroup", rel)
						if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
							return candidate, nil
						}
					}
				}
			}
		}
	}

	// 3. Check standard systemd / Docker cgroup v2 locations
	candidates := []string{
		fmt.Sprintf("/sys/fs/cgroup/system.slice/docker-%s.scope", containerIDOrName),
		fmt.Sprintf("/sys/fs/cgroup/docker/%s", containerIDOrName),
		fmt.Sprintf("/sys/fs/cgroup/%s", containerIDOrName),
	}

	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			return c, nil
		}
	}

	return "", fmt.Errorf("could not resolve cgroup v2 path for container %q; ensure cgroup v2 is enabled", containerIDOrName)
}

// AttachInterceptor loads the eBPF egress program and attaches it to the specified cgroup path.
func AttachInterceptor(cgroupPath string, blockedPorts ...uint16) (*Interceptor, error) {
	if cgroupPath == "" {
		return nil, errors.New("cgroup path must not be empty")
	}

	// Safety check: protect the root host cgroup from accidental attach
	cleanPath := filepath.Clean(cgroupPath)
	if cleanPath == "/sys/fs/cgroup" {
		return nil, errors.New("cannot attach egress filter to root /sys/fs/cgroup (would filter entire host); specify a container cgroup")
	}

	spec := NewEgressFilterCollectionSpec()
	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return nil, fmt.Errorf("loading eBPF collection: %w", err)
	}

	prog, ok := coll.Programs["block_outbound_db"]
	if !ok {
		coll.Close()
		return nil, errors.New("missing block_outbound_db program in collection")
	}

	// Attach program to cgroup v2 inet egress hook
	l, err := link.AttachCgroup(link.CgroupOptions{
		Path:    cgroupPath,
		Attach:  ebpf.AttachCGroupInetEgress,
		Program: prog,
	})
	if err != nil {
		coll.Close()
		return nil, fmt.Errorf("attaching eBPF program to cgroup %s: %w", cgroupPath, err)
	}

	interceptor := &Interceptor{
		coll:       coll,
		link:       l,
		cgroupPath: cgroupPath,
		blocked:    make(map[uint16]bool),
	}

	ports := blockedPorts
	if len(ports) == 0 {
		ports = DefaultBlockedPorts
	}

	if err := interceptor.UpdateBlockedPorts(ports); err != nil {
		interceptor.Detach()
		return nil, fmt.Errorf("populating blocked ports: %w", err)
	}

	return interceptor, nil
}

// AttachContainerInterceptor resolves the container's cgroup path and attaches the filter.
func AttachContainerInterceptor(containerIDOrName string, blockedPorts ...uint16) (*Interceptor, error) {
	cgPath, err := ResolveContainerCgroup(containerIDOrName)
	if err != nil {
		return nil, err
	}
	return AttachInterceptor(cgPath, blockedPorts...)
}

// UpdateBlockedPorts populates the eBPF blocked_ports map with the specified ports.
func (i *Interceptor) UpdateBlockedPorts(ports []uint16) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	bMap := i.coll.Maps["blocked_ports"]
	if bMap == nil {
		return errors.New("blocked_ports map not found")
	}

	// Reset tracked ports
	i.blocked = make(map[uint16]bool)

	var flag uint8 = 1
	for _, port := range ports {
		key := PortToKey(port)
		if err := bMap.Put(key, flag); err != nil {
			return fmt.Errorf("adding port %d to eBPF map: %w", port, err)
		}
		i.blocked[port] = true
	}

	return nil
}

// GetDropCount reads the kernel drop counter map for telemetry reporting.
func (i *Interceptor) GetDropCount() (uint64, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	cMap := i.coll.Maps["drop_counter"]
	if cMap == nil {
		return 0, errors.New("drop_counter map not found")
	}

	var key uint32 = 0
	var count uint64
	if err := cMap.Lookup(key, &count); err != nil {
		return 0, fmt.Errorf("reading drop_counter: %w", err)
	}

	return count, nil
}

// Detach releases the kernel hooks and frees eBPF resources.
func (i *Interceptor) Detach() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	var errs []string
	if i.link != nil {
		if err := i.link.Close(); err != nil {
			errs = append(errs, err.Error())
		}
		i.link = nil
	}

	if i.coll != nil {
		i.coll.Close()
		i.coll = nil
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
