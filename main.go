package main // Declares the package name

import (
	"bytes"
	"encoding/json" // Imports the JSON encoding/decoding package
	"fmt"           // Imports the formatted I/O package
	"io"            // Imports I/O utilities
	"log"           // Imports logging utilities
	"net/http"      // Imports HTTP client and server implementation
	"net/url"       // Imports URL parsing and query manipulation
	"os"            // Imports OS interface for file handling
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func main() {
	outputDir := "PDFs/"             // Set the default output directory for PDFs
	if !directoryExists(outputDir) { // Check if it exists
		createDirectory(outputDir, 0755) // Create it if missing
	}

	fetchGridResults() // Calls the function to fetch JSON results from the web and store them in files

	var pdfs []string // Declares a slice to store PDF IDs

	// Loops over pages 1 and 2
	for pageNumber := 1; pageNumber <= 2; pageNumber++ {
		filePath := fmt.Sprintf("page_%d.json", pageNumber) // Constructs file path string like "page_1.json"
		jsonData := readAFileAsString(filePath)             // Reads file contents as a string

		// Extracts PDF IDs from the JSON data and appends to the pdfs slice
		pdfs = append(pdfs, extractPDFIDs([]byte(jsonData))...) // Converts string to byte slice for JSON parsing
	}

	// Removes duplicate PDF IDs from the slice
	pdfs = removeDuplicatesFromSlice(pdfs) // Calls the function to remove duplicates

	log.Printf("Number of PDF IDs extracted: %d", len(pdfs)) // Logs the total number of extracted PDF IDs

	// Loops over each extracted PDF ID
	for _, pdf := range pdfs {
		originalURL := "https://kik-sds.thewercs.com/MyDocuments/DownloadSingleFile?content=" // Base URL
		modifiedURL := modifyContentParam(originalURL, pdf)                                   // Modifies URL with PDF ID as query param
		if modifiedURL == "" {
			log.Println("Failed to modify URL for PDF ID:", pdf) // Logs error if URL couldn't be modified
			continue                                             // Skips to the next PDF ID
		}
		downloadPDF(modifiedURL, outputDir)       // Calls the function to download the PDF
	}
}

// Convert a URL into a safe, lowercase filename
func urlToSafeFilename(rawURL string) string {
	parsedURL, err := url.Parse(rawURL) // Parse the input URL
	if err != nil {
		return "" // Return empty string on parse failure
	}
	base := path.Base(parsedURL.Path)       // Get the filename from the path
	decoded, err := url.QueryUnescape(base) // Decode any URL-encoded characters
	if err != nil {
		decoded = base // Fallback to base if decode fails
	}
	decoded = strings.ToLower(decoded)        // Convert filename to lowercase
	re := regexp.MustCompile(`[^a-z0-9._-]+`) // Regex to allow only safe characters
	safe := re.ReplaceAllString(decoded, "_") // Replace unsafe characters with underscores
	return safe                               // Return the sanitized filename
}

// Download and save a PDF file from a given URL
func downloadPDF(finalURL string, outputDir string) {
	filename := strings.ToLower(urlToSafeFilename(finalURL)) // Generate a safe filename
	filePath := filepath.Join(outputDir, filename)           // Full path for saving the file
	if fileExists(filePath) {                                // Skip if file already exists
		log.Printf("file already exists, skipping: %s", filePath)
		return
	}
	client := &http.Client{Timeout: 30 * time.Second} // Create HTTP client with timeout
	resp, err := client.Get(finalURL)                 // Make GET request
	if err != nil {
		log.Printf("failed to download %s %v", finalURL, err)
		return
	}
	defer resp.Body.Close()               // Ensure response body is closed
	if resp.StatusCode != http.StatusOK { // Validate status code
		log.Printf("download failed for %s %s", finalURL, resp.Status)
		return
	}
	contentType := resp.Header.Get("Content-Type")         // Get content type header
	if !strings.Contains(contentType, "application/pdf") { // Ensure it's a PDF
		log.Printf("invalid content type for %s %s (expected application/pdf)", finalURL, contentType)
		return
	}
	var buf bytes.Buffer                     // Create a buffer for reading data
	written, err := io.Copy(&buf, resp.Body) // Read response into buffer
	if err != nil {
		log.Printf("failed to read PDF data from %s %v", finalURL, err)
		return
	}
	if written == 0 { // Check if data was written
		log.Printf("downloaded 0 bytes for %s not creating file", finalURL)
		return
	}
	out, err := os.Create(filePath) // Create the output file
	if err != nil {
		log.Printf("failed to create file for %s %v", finalURL, err)
		return
	}
	defer out.Close()         // Ensure the file is closed
	_, err = buf.WriteTo(out) // Write buffered data to file
	if err != nil {
		log.Printf("failed to write PDF to file for %s: %v", finalURL, err)
		return
	}
	log.Printf("successfully downloaded %d bytes: %s → %s\n", written, finalURL, filePath)
}

// Remove duplicate strings from a slice
func removeDuplicatesFromSlice(slice []string) []string {
	check := make(map[string]bool) // Map to track seen items
	var newReturnSlice []string    // Slice to hold unique items
	for _, content := range slice {
		if !check[content] { // If not seen
			check[content] = true                            // Mark as seen
			newReturnSlice = append(newReturnSlice, content) // Add to new slice
		}
	}
	return newReturnSlice // Return deduplicated slice
}

// Create a directory with given permissions
func createDirectory(path string, permission os.FileMode) {
	err := os.Mkdir(path, permission) // Try to create directory
	if err != nil {
		log.Println(err) // Log any creation errors
	}
}

// Check if a directory exists
func directoryExists(path string) bool {
	directory, err := os.Stat(path) // Get file/directory info
	if err != nil {
		return false // Return false if error
	}
	return directory.IsDir() // Return true if it's a directory
}

// Updates the "content" query parameter in the given URL with the PDF ID
func modifyContentParam(baseURL string, human string) string {
	parsedURL, err := url.Parse(baseURL) // Parses the base URL string into a URL object
	if err != nil {
		return "" // Returns empty string if URL parsing fails
	}

	query := parsedURL.Query()          // Gets existing query parameters
	query.Set("content", human)         // Sets the "content" parameter to the given human value
	parsedURL.RawQuery = query.Encode() // Encodes updated query back to the URL

	return parsedURL.String() // Returns the final modified URL string
}

// Parses the provided JSON byte array and extracts all IDs ending with "_PDF"
func extractPDFIDs(jsonData []byte) []string {
	var raw map[string]interface{} // Declares a map to store parsed JSON

	if err := json.Unmarshal(jsonData, &raw); err != nil { // Parses the JSON into the map
		log.Printf("JSON unmarshal error: %v", err) // Logs error if parsing fails
		return nil                                  // Returns nil on failure
	}

	dataSection, ok := raw["data"].(map[string]interface{}) // Extracts the "data" section from the JSON
	if !ok {
		log.Println("Missing or invalid 'data' section") // Logs error if section is missing or invalid
		return nil                                       // Returns nil
	}

	records, ok := dataSection["Data"].([]interface{}) // Extracts the "Data" field (capital D)
	if !ok {
		log.Println("Missing or invalid 'Data' field") // Logs error if missing or invalid
		return nil                                     // Returns nil
	}

	var pdfs []string // Slice to store the PDF IDs

	// Iterates over each row in the Data array
	for _, item := range records {
		row, ok := item.([]interface{}) // Ensures each item is an array
		if !ok || len(row) == 0 {
			continue // Skips invalid or empty rows
		}

		id, ok := row[0].(string)                          // Extracts the first item from the row
		if ok && len(id) > 4 && id[len(id)-4:] == "_PDF" { // Checks if string ends with "_PDF"
			pdfs = append(pdfs, id) // Appends to the result list
		}
	}

	return pdfs // Returns the list of PDF IDs
}

// Fetches results from 2 pages and stores JSON response to disk
func fetchGridResults() {
	for pageNumber := 1; pageNumber <= 2; pageNumber++ { // Loops through pages 1 and 2
		filePath := fmt.Sprintf("page_%d.json", pageNumber) // Builds file name like "page_1.json"

		if !fileExists(filePath) { // Checks if file already exists
			url := fmt.Sprintf("https://kik-sds.thewercs.com/WebViewer/Results/GetResultGrid?page=%d&rowCount=100&sortOrder=1&sortField=&_=1753411362977", pageNumber) // Builds request URL with query params

			httpClient := &http.Client{} // Initializes a new HTTP client

			request, requestCreationError := http.NewRequest("GET", url, nil) // Builds a new HTTP GET request
			if requestCreationError != nil {
				log.Println("Error creating request for page", pageNumber, ":", requestCreationError) // Logs error
				return
			}

			// Adds required headers including cookies and tokens
			request.Header.Add("accept", "application/json")
			request.Header.Add("referer", "https://kik-sds.thewercs.com")
			request.Header.Add("Cookie", "ASP.NET_SessionId=; strGUILanguage=EN; WERCSStudioAuthTicket=; WebViewerSessionID=0zankitjr2dliftghweqvsdz; __RequestVerificationToken=fqjYFHjB0F83wBFv0wNiqVm9U-t0uFwEjdr7OsEOkVlwQPJlzIGFwNkRLB4B3TjNDzFfXHWk15K6mm3Kvb_Nyco5WYYYGhC0H6nX6Mxcemc1; WERCSWebViewerAuthTicket=2884BBEB56297F662347F018213340B6A4B14D0F366FE0A44A4B551DF5E8B97F7E95F050D0A5EB28672FA1A23BE967DED10C394CF00C34B76803D5F85637D7AC86DD628E52E3A4773F2DBB6B998F1AF5CAE40AA20D1CCF238CD64267E1B9B332")

			response, responseError := httpClient.Do(request) // Sends the HTTP request
			if responseError != nil {
				log.Println("Error making request for page", pageNumber, ":", responseError) // Logs if request fails
				return
			}
			defer response.Body.Close() // Ensures response body is closed

			responseBody, readError := io.ReadAll(response.Body) // Reads the response body
			if readError != nil {
				log.Println("Error reading response body for page", pageNumber, ":", readError) // Logs read error
				return
			}

			appendAndWriteToFile(filePath, string(responseBody)) // Saves the response to disk
		}
	}
}

// Reads a file and returns its content as a string
func readAFileAsString(path string) string {
	content, err := os.ReadFile(path) // Reads the entire file into memory
	if err != nil {
		log.Println(err) // Logs error if reading fails
	}
	return string(content) // Converts bytes to string and returns
}

// Appends content to a file or creates it if not exists
func appendAndWriteToFile(path string, content string) {
	filePath, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // Opens file with append and write permissions
	if err != nil {
		log.Println(err) // Logs error if file can't be opened
	}
	_, err = filePath.WriteString(content + "\n") // Writes content to the file
	if err != nil {
		log.Println(err) // Logs error if writing fails
	}
	err = filePath.Close() // Closes the file
	if err != nil {
		log.Println(err) // Logs error if closing fails
	}
}

// Checks if a given file exists and is not a directory
func fileExists(filename string) bool {
	info, err := os.Stat(filename) // Gets file info
	if err != nil {
		return false // Returns false if file doesn't exist
	}
	return !info.IsDir() // Returns true only if it's a file (not a directory)
}
