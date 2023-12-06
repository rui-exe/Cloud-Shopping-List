package main

import (
	"CloudShoppingList/consistent_hashing"
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
)

type LoadBalancer struct {
	Ring *consistent.Ring
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		Ring: consistent.NewRing(),
	}
}

func (lb *LoadBalancer) AddNode(id, server string) {
	lb.Ring.AddNode(id, server)
}

func (lb *LoadBalancer) Put(email string) ([]string, error) {
	return lb.Ring.Put(email)
}

func (lb *LoadBalancer) Get(email string) (string, error) {
	return lb.Ring.Get(email)
}

func (lb *LoadBalancer) GetNodeAndReplicas(email string) ([]string, error) {
	return lb.Ring.GetNodeAndReplicas(email)
}

func (lb *LoadBalancer) HandleNodeConnection(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received node connection")

	// Read the node ID and server address from the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	// Split the body into the node ID and server address using comma as the separator
	parts := strings.Split(string(body), ",")
	if len(parts) != 2 {
		http.Error(w, "Invalid request body format", http.StatusBadRequest)
		return
	}

	nodeID := parts[0]
	nodeAddress := parts[1]

	// Add the node to the ring
	lb.AddNode(nodeID, nodeAddress)
	fmt.Printf("Added node %s at address %s\n", nodeID, nodeAddress)

	// Send a success response to the node
	w.WriteHeader(http.StatusOK)
}

func (lb *LoadBalancer) HandleShoppingListPut(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}
	email := r.FormValue("email")
	fmt.Println("Email:", email)

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error reading file", http.StatusBadRequest)
		return
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)
	fmt.Printf("Received file %s\n", handler.Filename)
	// Get the node ID for the email
	servers, err := lb.Put(email)
	if err != nil {
		// If there is an error getting the node ID, respond with an internal server error
		http.Error(w, "Error getting node ID", http.StatusInternalServerError)
		return
	}
	contents, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// Send the file to all servers simultaneously
	for _, server := range servers {
		fmt.Printf("Sending file to server %s\n", server)

		// Create a new multipart form
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Write the email to the form
		err = writer.WriteField("email", email)
		if err != nil {
			fmt.Println("Error writing to form field:", err)
			return
		}

		// Create a new form file
		part, err := writer.CreateFormFile("file", handler.Filename)
		if err != nil {
			fmt.Println("Error creating form file:", err)
			return
		}

		// Write the file contents to the form file
		_, err = part.Write(contents)
		if err != nil {
			fmt.Println("Error writing to form file:", err)
			return
		}

		// Close the writer
		err = writer.Close()
		if err != nil {
			fmt.Println("Error closing writer:", err)
			return
		}

		// Create a new request
		req, err := http.NewRequest("POST", "http://"+server+"/putListServer", body)
		if err != nil {
			fmt.Println("Error creating request:", err)
			return
		}

		// Set the content type header
		req.Header.Set("Content-Type", writer.FormDataContentType())

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("Error sending request:", err)
			return
		}

		// Close the response body
		err = resp.Body.Close()
		if err != nil {
			fmt.Println("Error closing response body:", err)
			return
		}

		// Check the response status code
		if resp.StatusCode != http.StatusOK {
			fmt.Println("Error sending file to server:", resp.Status)
			return
		}

		// Print a success message
		fmt.Println("Sent file to server successfully")
	}
	// Send a success response (HTTP 200 OK) to the client
	w.WriteHeader(http.StatusOK)
}

func (lb *LoadBalancer) HandleShoppingListGet(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimPrefix(r.URL.Path, "/list/")
	fmt.Println("Email:", email)

	// Get the node ID for the email
	servers, err := lb.GetNodeAndReplicas(email)
	fmt.Println("Server:", servers)
	if err != nil {
		// If there is an error getting the node ID, respond with an internal server error
		http.Error(w, "Error getting node ID", http.StatusInternalServerError)
		return
	}

	for _, server := range servers {

		// Send the request to the server
		resp, err := http.Get("http://" + server + "/getListServer/" + email)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		// Check the response status code
		if resp.StatusCode != http.StatusOK {
			continue
		}

		// Copy the response body to the client
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			continue
		}

		// Send a success response (HTTP 200 OK) to the client
		w.WriteHeader(http.StatusOK)
		return
	}

	// If the request was not successful, send an error response (HTTP 500 Internal Server Error) to the client
	http.Error(w, "Error getting shopping list from server", http.StatusInternalServerError)
}

func main() {
	loadBalancer := NewLoadBalancer()

	// Set up HTTP handler for load balancer
	http.HandleFunc("/connect-node", loadBalancer.HandleNodeConnection)
	http.HandleFunc("/putList", loadBalancer.HandleShoppingListPut)
	http.HandleFunc("/list/", loadBalancer.HandleShoppingListGet)
	// Start the load balancer on port 8080
	fmt.Println("Load balancer listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
