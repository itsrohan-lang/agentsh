package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/stuntdouble/cli/pkg/dlp"
)

const (
	DummyAnthropicKey = "sd-dummy-anthropic-key-000"
	DummyOpenAIKey    = "sd-dummy-openai-key-000"
)

// ZeroTrustProxy intercepts outgoing API calls from sandboxed agents, replacing
// dummy credentials with real host keys on egress so agents never hold real secrets.
type ZeroTrustProxy struct {
	server        *http.Server
	listener      net.Listener
	listenAddr    string
	realAnthropic string
	realOpenAI    string
	dlpScanner    *dlp.Scanner
	StrictDLP     bool
	mu            sync.RWMutex
}

// NewZeroTrustProxy initializes a local API key substitution proxy
func NewZeroTrustProxy() *ZeroTrustProxy {
	return &ZeroTrustProxy{
		realAnthropic: os.Getenv("ANTHROPIC_API_KEY"),
		realOpenAI:    os.Getenv("OPENAI_API_KEY"),
		dlpScanner:    dlp.NewScanner(),
	}
}

// SetStrictDLP enables or disables strict DLP request blocking
func (p *ZeroTrustProxy) SetStrictDLP(strict bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.StrictDLP = strict
}

// Start binds to an available interface port and starts serving requests
func (p *ZeroTrustProxy) Start(ctx context.Context) (string, error) {
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		// Fallback to loopback if 0.0.0.0 is restricted
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return "", fmt.Errorf("failed to bind proxy listener: %w", err)
		}
	}

	p.listener = listener
	p.listenAddr = listener.Addr().String()

	p.server = &http.Server{
		Handler: http.HandlerFunc(p.handleProxy),
	}

	go func() {
		if err := p.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("[ZeroTrustProxy] Server error: %v", err)
		}
	}()

	return p.listenAddr, nil
}

// Stop gracefully shuts down the proxy server
func (p *ZeroTrustProxy) Stop() error {
	if p.server != nil {
		return p.server.Shutdown(context.Background())
	}
	return nil
}

// GetListenAddr returns the assigned IP:Port of the proxy
func (p *ZeroTrustProxy) GetListenAddr() string {
	return p.listenAddr
}

// GetPort returns the assigned numeric port of the proxy listener
func (p *ZeroTrustProxy) GetPort() int {
	if p.listener != nil {
		if tcpAddr, ok := p.listener.Addr().(*net.TCPAddr); ok {
			return tcpAddr.Port
		}
	}
	return 0
}

// handleProxy performs dummy key substitution and forwards API requests
func (p *ZeroTrustProxy) handleProxy(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	strictDLP := p.StrictDLP
	p.mu.RUnlock()

	// 1. Perform Strict DLP Request Inspection
	if strictDLP && p.dlpScanner != nil {
		path := r.URL.Path
		findings := p.dlpScanner.Scan(path)
		if len(findings) > 0 {
			log.Printf("🚨 [agentsh DLP] Blocked egress request containing sensitive data: %s", path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "Blocked by agentsh DLP Engine: Sensitive credential or PII leak detected in egress request"}`))
			return
		}
	}

	// 2. Resolve destination URL (handles proxy absolute URL and BaseURL reverse proxy)
	targetURL := p.resolveTargetURL(r)

	// Prepare forwarded request
	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	outReq.Header = r.Header.Clone()

	// 3. Perform zero-trust secret substitution
	p.substituteSecret(outReq.Header, "x-api-key", DummyAnthropicKey, p.realAnthropic)
	p.substituteSecret(outReq.Header, "Authorization", DummyAnthropicKey, p.realAnthropic)
	p.substituteSecret(outReq.Header, "Authorization", DummyOpenAIKey, p.realOpenAI)

	// Remove hop-by-hop headers
	outReq.Header.Del("Proxy-Connection")

	// 4. Execute HTTP request
	client := &http.Client{}
	resp, err := client.Do(outReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Proxy egress error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 5. Copy response back to caller
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *ZeroTrustProxy) resolveTargetURL(r *http.Request) string {
	rawURL := r.URL.String()
	// If the request already has an absolute HTTP/HTTPS URL (standard proxy forward)
	if r.URL.Scheme == "http" || r.URL.Scheme == "https" {
		return rawURL
	}

	// If target host is in Host header and not our local proxy listener
	host := r.Host
	if host != "" && !strings.Contains(host, "127.0.0.1") && !strings.Contains(host, "localhost") && !strings.Contains(host, "host.docker.internal") {
		return fmt.Sprintf("https://%s%s", host, r.URL.RequestURI())
	}

	// BaseURL mode (e.g. ANTHROPIC_BASE_URL or OPENAI_BASE_URL set to this proxy)
	path := r.URL.Path
	if r.Header.Get("x-api-key") != "" || strings.Contains(path, "messages") || strings.Contains(path, "anthropic") {
		parsed, _ := url.Parse("https://api.anthropic.com")
		parsed.Path = path
		parsed.RawQuery = r.URL.RawQuery
		return parsed.String()
	}

	if strings.Contains(path, "chat/completions") || strings.Contains(path, "models") || strings.Contains(r.Header.Get("Authorization"), DummyOpenAIKey) {
		parsed, _ := url.Parse("https://api.openai.com")
		parsed.Path = path
		parsed.RawQuery = r.URL.RawQuery
		return parsed.String()
	}

	// Default fallback to Anthropic API endpoint
	parsed, _ := url.Parse("https://api.anthropic.com")
	parsed.Path = path
	parsed.RawQuery = r.URL.RawQuery
	return parsed.String()
}

func (p *ZeroTrustProxy) substituteSecret(headers http.Header, headerName, dummyValue, realValue string) {
	if realValue == "" {
		return
	}
	val := headers.Get(headerName)
	if strings.Contains(val, dummyValue) {
		headers.Set(headerName, strings.ReplaceAll(val, dummyValue, realValue))
	}
}
