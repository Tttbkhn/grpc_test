package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	// Import the generated Go code (adjust module name if needed)
	pb "grpc_test/pdf_processor"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure" // For insecure connection
)

const (
	// pythonServerAddress = "localhost:50052" // Address of the Python gRPC server - Replaced by env var
	goApiPort      = ":8080"          // Port for this Go HTTP API server
	defaultPdfPath = "hint_test2.pdf" // PDF to process (hardcoded for now)
)

// getPythonServerAddress reads the address from env var or returns default.
func getPythonServerAddress() string {
	// addr := os.Getenv("PYTHON_GRPC_ADDR")
	// if addr == "" {
	// 	log.Println("[Go gRPC Client] PYTHON_GRPC_ADDR not set, defaulting to localhost:50052")
	// 	addr = "localhost:50052"
	// }
	return "0.tcp.au.ngrok.io:1840"
}

// processPdfHandler handles HTTP requests to process the PDF.
func processPdfHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[Go API] Received HTTP request on /process-pdf")

	// --- Read the PDF file ---
	// In a real API, you might get the filename/path from the HTTP request (r).
	// Here, we use a hardcoded path.
	// Note: We are now running from cmd/server, so the relative path needs adjusting
	pdfFilename := defaultPdfPath // Adjusted path relative to cmd/server
	pdfBytes, err := os.ReadFile(pdfFilename)
	if err != nil {
		log.Printf("[Go API] Failed to read PDF file '%s': %v", pdfFilename, err)
		http.Error(w, "Failed to read PDF file", http.StatusInternalServerError)
		return
	}
	log.Printf("[Go API] Successfully read PDF file '%s', size: %d bytes", pdfFilename, len(pdfBytes))

	// --- Call Python gRPC Server ---
	pythonServerAddress := getPythonServerAddress() // Use the env var logic
	log.Printf("[Go gRPC Client] Attempting to connect to Python server at %s", pythonServerAddress)
	// Set up a connection to the server.
	// Use insecure credentials as we haven't set up TLS.
	conn, err := grpc.NewClient(pythonServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("[Go gRPC Client] Failed to connect: %v", err)
		http.Error(w, "Failed to connect to processing service", http.StatusInternalServerError)
		return
	}
	defer conn.Close() // Ensure connection is closed

	// Create a client for the PdfProcessorService
	client := pb.NewPdfProcessorServiceClient(conn)
	log.Println("[Go gRPC Client] Connected to Python server.")

	// Prepare the request
	request := &pb.ProcessPdfRequest{
		Filename:   defaultPdfPath,
		PdfContent: pdfBytes,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5*60)
	defer cancel()

	log.Println("[Go gRPC Client] Sending ProcessPdf request to Python server...")
	response, err := client.ProcessPdf(ctx, request)
	if err != nil {
		log.Printf("[Go gRPC Client] Could not process PDF via gRPC: %v", err)
		http.Error(w, "Failed to process PDF via remote service", http.StatusInternalServerError)
		return
	}

	log.Printf("[Go gRPC Client] Received structured response from Python server. Status: %s, Error: %s",
		response.GetProcessingStatus(), response.GetErrorInfo())

	w.Header().Set("Content-Type", "application/json")
	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("[Go API] Failed to marshal response to JSON: %v", err)
		http.Error(w, "Failed to format response", http.StatusInternalServerError)
		return
	}

	log.Printf("[Go API] Sending JSON response back to HTTP client.")
	w.Write(jsonBytes) // Write JSON bytes to the response
}

func main() {
	// Note: Relative path needs adjusting here too
	pdfCheckPath := defaultPdfPath
	log.Printf("[Go Main] Checking for PDF file at resolved path: %s", pdfCheckPath) // Add logging
	fileInfo, err := os.Stat(pdfCheckPath)                                           // Store result of Stat
	if err != nil {                                                                  // Check error directly
		if os.IsNotExist(err) {
			log.Fatalf("File not found: %s", pdfCheckPath)
		} else {
			// Log other potential errors from os.Stat (e.g., permission denied)
			log.Printf("[Go Main] os.Stat error (not IsNotExist): %v. Proceeding without dummy file creation.", err)
		}
	} else {
		log.Printf("[Go Main] os.Stat result: File exists. Size: %d bytes.", fileInfo.Size()) // Log result and size
		log.Printf("Using existing PDF file: %s", pdfCheckPath)
	}

	// --- Setup HTTP Server ---
	http.HandleFunc("/process-pdf", processPdfHandler) // Register handler for the /process-pdf endpoint

	log.Printf("[Go API] Starting HTTP server, listening on port %s", goApiPort)
	log.Printf("[Go API] Access http://localhost%s/process-pdf to trigger processing.", goApiPort)
	log.Printf("[Go gRPC Client] Target Python server address: %s", getPythonServerAddress()) // Log the target address

	// Start the HTTP server
	if err := http.ListenAndServe(goApiPort, nil); err != nil {
		log.Fatalf("[Go API] Failed to start HTTP server: %v", err)
	}
}
