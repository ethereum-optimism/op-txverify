package core

import (
	"embed"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

//go:embed web/reader.html web/lib/*
var templateFS embed.FS

// scanPort is where the scanner page and its result endpoint are served, on loopback only.
const scanPort = 8081

// ScanQRCode serves the scanner page to a local browser and returns the QR code it decodes.
func ScanQRCode() (string, error) {
	// Create a channel to receive the QR code result
	resultChan := make(chan string)

	listeners, err := listenLoopback(scanPort)
	if err != nil {
		return "", err
	}
	// Buffered, so a listener that fails after this function has returned does not leave its
	// goroutine blocked on a send nobody will read.
	errChan := make(chan error, len(listeners))

	// The port is already bound, so there is nothing to wait for: this returns once the page is
	// being served.
	if err := startCameraServer(listeners, resultChan, errChan); err != nil {
		return "", err
	}

	fmt.Println("Camera activated. Point camera at QR code...")
	fmt.Println("For multi-part QR codes, scan each code in sequence.")
	fmt.Println("A browser window should open automatically.")
	fmt.Println("Press Ctrl+C to cancel")

	// The literal loopback address, not "localhost": that resolves to whichever family the
	// resolver prefers, which may be one this process did not manage to bind.
	_ = openBrowser(fmt.Sprintf("http://%s", listeners[0].Addr().String()))

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return "", err
	case <-time.After(300 * time.Second): // Extended timeout for multi-part scanning
		return "", fmt.Errorf("timeout waiting for QR code")
	}
}

// listenLoopback binds the port on both loopback families. Binding the wildcard address instead
// puts the /result endpoint on every interface, so any host on the same network could post a
// scanner result and choose the transaction this tool goes on to verify. One family is enough:
// hosts configured for only one of them are ordinary.
func listenLoopback(port int) ([]net.Listener, error) {
	var listeners []net.Listener
	var failures []string
	for _, host := range []string{"127.0.0.1", "[::1]"} {
		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", host, err))
			continue
		}
		listeners = append(listeners, listener)
	}
	if len(listeners) == 0 {
		return nil, fmt.Errorf("could not listen on loopback port %d: %s", port, strings.Join(failures, "; "))
	}
	return listeners, nil
}

func startCameraServer(listeners []net.Listener, resultChan chan string, errChan chan error) error {
	// Create a template from the embedded file
	tmpl, err := template.ParseFS(templateFS, "web/reader.html")
	if err != nil {
		return fmt.Errorf("error creating template: %w", err)
	}

	// A private mux rather than http.DefaultServeMux: these endpoints belong to this scan, and
	// registering them globally panics on a second call and exposes them to any other server the
	// process happens to run.
	mux := http.NewServeMux()

	// Serve static files from the embedded filesystem with proper MIME types
	mux.HandleFunc("/lib/", func(w http.ResponseWriter, r *http.Request) {
		// The URL path is /lib/something, but in the embedded FS it's web/lib/something
		path := "web" + r.URL.Path

		data, err := templateFS.ReadFile(path)
		if err != nil {
			http.Error(w, "File not found: "+path, http.StatusNotFound)
			return
		}

		// Set the correct content type based on file extension
		if strings.HasSuffix(path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		} else if strings.HasSuffix(path, ".wasm") {
			w.Header().Set("Content-Type", "application/wasm")
		}

		_, _ = w.Write(data)
	})

	// Store for multi-part QR codes
	var (
		qrParts = make(map[int]string)
		qrMutex sync.Mutex
	)

	// Handle the root path
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_ = tmpl.Execute(w, nil)
	})

	// Handle the result endpoint
	mux.HandleFunc("/result", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get the QR code data
		_ = r.ParseForm()
		qrText := r.FormValue("data")

		// Check if it's a multi-part QR code
		// Format: "PART:index:total:data"
		if strings.HasPrefix(qrText, "PART:") {
			parts := strings.SplitN(qrText, ":", 4)
			if len(parts) != 4 {
				_, _ = w.Write([]byte(`{"success":false}`))
				return
			}

			partIndex, err1 := parseInt(parts[1])
			totalParts, err2 := parseInt(parts[2])
			data := parts[3]

			if err1 != nil || err2 != nil || partIndex < 1 || partIndex > totalParts {
				_, _ = w.Write([]byte(`{"success":false}`))
				return
			}

			qrMutex.Lock()
			qrParts[partIndex] = data

			// Check if we have all parts
			complete := len(qrParts) == totalParts
			remaining := totalParts - len(qrParts)
			qrMutex.Unlock()

			if complete {
				// Combine all parts
				var combinedData strings.Builder
				for i := 1; i <= totalParts; i++ {
					combinedData.WriteString(qrParts[i])
				}

				// Send the complete result
				resultChan <- combinedData.String()
				_, _ = w.Write([]byte(`{"success":true,"complete":true}`))
				return
			}

			// Send progress update
			response := fmt.Sprintf(`{"success":true,"complete":false,"partIndex":%d,"totalParts":%d,"remaining":%d}`,
				partIndex, totalParts, remaining)
			_, _ = w.Write([]byte(response))
			return
		}

		// Single QR code (not multi-part)
		resultChan <- qrText
		_, _ = w.Write([]byte(`{"success":true,"complete":true}`))
	})

	server := &http.Server{Handler: mux}
	for _, listener := range listeners {
		go func(listener net.Listener) {
			if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
				errChan <- fmt.Errorf("server error: %w", err)
			}
		}(listener)
	}
	return nil
}

// parseInt safely parses a string to an integer
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// openBrowser opens the default browser to the specified URL
func openBrowser(url string) error {
	return exec.Command("open", url).Start()
}
