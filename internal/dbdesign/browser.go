package dbdesign

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

// BrowserURL returns URL for opening in browser
func BrowserURL(host, port string) string {
	h := host
	if h == "" || h == "0.0.0.0" || h == "::" {
		h = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(h, port)
}

func waitForServer(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode < 500 {
				return true
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// OpenBrowserWhenReady waits then opens browser
func OpenBrowserWhenReady(url string) error {
	if !waitForServer(url, 8*time.Second) {
		return fmt.Errorf("server did not become ready in time")
	}
	return openBrowser(url)
}
