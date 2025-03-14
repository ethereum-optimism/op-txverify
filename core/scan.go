package core

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

//go:embed web/reader.html web/lib/*
var templateFS embed.FS

// ScanQRCode opens the camera device and scans for a QR code
// Returns the decoded string content of the QR code
func ScanQRCode(deviceID string) (string, error) {
	// Create a channel to receive the QR code result
	resultChan := make(chan string)
	errChan := make(chan error)

	// Start a local web server to access the camera
	var wg sync.WaitGroup
	wg.Add(1)

	// Start the server
	go startCameraServer(&wg, resultChan, errChan)

	// Wait for the server to start
	wg.Wait()

	fmt.Println("Camera activated. Point camera at QR code...")
	fmt.Println("For multi-part QR codes, scan each code in sequence.")
	fmt.Println("A browser window should open automatically.")
	fmt.Println("Press Ctrl+C to cancel")

	// Open the browser
	openBrowser("http://localhost:8081")

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return "", err
	case <-time.After(120 * time.Second): // Extended timeout for multi-part scanning
		return "", fmt.Errorf("timeout waiting for QR code")
	}
}

func startCameraServer(wg *sync.WaitGroup, resultChan chan string, errChan chan error) {
	// Create a template from the embedded file
	tmpl, err := template.ParseFS(templateFS, "web/reader.html")
	if err != nil {
		errChan <- fmt.Errorf("error creating template: %w", err)
		return
	}

	// Serve static files from the embedded filesystem with proper MIME types
	http.HandleFunc("/lib/", func(w http.ResponseWriter, r *http.Request) {
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

		w.Write(data)
	})

	// Store for multi-part QR codes
	var (
		qrParts = make(map[int]string)
		qrMutex sync.Mutex
	)

	// Handle the root path
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, nil)
	})

	// Handle the result endpoint
	http.HandleFunc("/result", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get the QR code data
		r.ParseForm()
		qrText := r.FormValue("data")

		// Check if it's a multi-part QR code
		// Format: "PART:index:total:data"
		if strings.HasPrefix(qrText, "PART:") {
			parts := strings.SplitN(qrText, ":", 4)
			if len(parts) != 4 {
				w.Write([]byte(`{"success":false}`))
				return
			}

			partIndex, err1 := parseInt(parts[1])
			totalParts, err2 := parseInt(parts[2])
			data := parts[3]

			if err1 != nil || err2 != nil || partIndex < 1 || partIndex > totalParts {
				w.Write([]byte(`{"success":false}`))
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
				w.Write([]byte(`{"success":true,"complete":true}`))
				return
			}

			// Send progress update
			response := fmt.Sprintf(`{"success":true,"complete":false,"partIndex":%d,"totalParts":%d,"remaining":%d}`,
				partIndex, totalParts, remaining)
			w.Write([]byte(response))
			return
		}

		// Single QR code (not multi-part)
		resultChan <- qrText
		w.Write([]byte(`{"success":true,"complete":true}`))
	})

	// Start the server
	server := &http.Server{Addr: ":8081"}

	// Signal that the server is ready
	wg.Done()

	// Start the server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		errChan <- fmt.Errorf("server error: %w", err)
	}
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
