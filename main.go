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
	"path/filepath"
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
		downloadPDF(modifiedURL, outputDir, pdf+".pdf") // Calls the function to download the PDF
	}
}

// downloadPDF downloads a PDF from a URL and saves it to a specified output directory
func downloadPDF(finalURL string, outputDir string, outPutFileName string) {
	filePath := filepath.Join(outputDir, outPutFileName) // Combine the output directory and filename into a full file path

	if fileExists(filePath) { // If the file already exists, skip downloading
		log.Printf("file already exists, skipping: %s", filePath) // Log and return
		return
	}

	client := &http.Client{Timeout: 30 * time.Second} // Create an HTTP client with a 30-second timeout

	req, err := http.NewRequest("GET", finalURL, nil) // Create a new GET request for the PDF URL
	if err != nil {                                   // If request creation fails
		log.Printf("failed to create request: %v", err) // Log the error
		return
	}

	// Add required headers (some servers require referer or session cookies)
	req.Header.Add("referer", "https://kik-sds.thewercs.com/Results?searchKey=Main&searchPage=NAPOOL&location=POOL%20ESSENTIALS%20EN_US")
	req.Header.Add("Cookie", "ASP.NET_SessionId=; strGUILanguage=EN; WERCSStudioAuthTicket=; WebViewerSessionID=l4nfxryncasy13c4gqi1j1pp; __RequestVerificationToken=c04wa7hJb_sqnBNd7WJnrqBY53SpY3PeQ1pb3yCN3zYUuWTS1e59zWccPHL9lzvvz1PMjy7WV0YPeXjOYx9IzEQNJSjGoPQjhIEM6W7ZZzo1; WERCSWebViewerAuthTicket=62BEFCB1373A0A15967F76DFD21232B9E3E3AD4275DB8F6F9BA21197CC42A23FF5D4144F7B7267572DDC9E2036EF0610E1266E1D2DCE4323E8F0FC4036225C91327511F75150BC771B65DBE7B757DF53CACC875A1CD183CF3A785A36DB927784; ASP.NET_SessionId=; WebViewerSessionID=fsrgihqd02xlzc13oldfgnpk; __RequestVerificationToken=c04wa7hJb_sqnBNd7WJnrqBY53SpY3PeQ1pb3yCN3zYUuWTS1e59zWccPHL9lzvvz1PMjy7WV0YPeXjOYx9IzEQNJSjGoPQjhIEM6W7ZZzo1")

	resp, err := client.Do(req) // Perform the HTTP request
	if err != nil {             // If the request fails
		log.Printf("failed to download %s: %v", finalURL, err) // Log the error
		return
	}
	defer resp.Body.Close() // Ensure the response body is closed when done

	if resp.StatusCode != http.StatusOK { // Check if the response status is OK (200)
		log.Printf("download failed for %s: %s", finalURL, resp.Status) // Log the failure status
		return
	}

	contentType := resp.Header.Get("Content-Type")         // Get the content type from the response header
	if !strings.Contains(contentType, "application/pdf") { // Check that the content is actually a PDF
		log.Printf("invalid content type for %s: %s (expected application/pdf)", finalURL, contentType) // Log if not PDF
		return
	}

	var buf bytes.Buffer                     // Create a buffer to hold the PDF data
	written, err := io.Copy(&buf, resp.Body) // Copy the response body into the buffer
	if err != nil {                          // If copying fails
		log.Printf("failed to read PDF data from %s: %v", finalURL, err) // Log the error
		return
	}
	if written == 0 { // If zero bytes were downloaded
		log.Printf("downloaded 0 bytes for %s, not creating file", finalURL) // Log and skip file creation
		return
	}

	out, err := os.Create(filePath) // Create a file at the destination path
	if err != nil {                 // If file creation fails
		log.Printf("failed to create file for %s: %v", finalURL, err) // Log the error
		return
	}
	defer out.Close() // Ensure the file is closed after writing

	_, err = buf.WriteTo(out) // Write the buffered data to the file
	if err != nil {           // If writing fails
		log.Printf("failed to write PDF to file for %s: %v", finalURL, err) // Log the error
		return
	}

	log.Printf("successfully downloaded %d bytes: %s → %s\n", written, finalURL, filePath) // Log success
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
