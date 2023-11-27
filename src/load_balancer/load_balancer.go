package main

import (
	"CloudShoppingList/consistent_hashing"
	"fmt"
	"io"
	"log"
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

func (lb *LoadBalancer) Put(email string) (string, error) {
	return lb.Ring.Put(email)
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
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// If there is an error reading the body, respond with a bad request error
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	// Split the body into parts using commas
	parts := strings.Split(string(body), ",")

	fmt.Println("Parts:", parts)

	// Check if the number of parts is not equal to 3
	if len(parts) != 3 {
		// If the format is invalid, respond with a bad request error
		http.Error(w, "Invalid request body format", http.StatusBadRequest)
		return
	}

	// Extract email, filename, and contents from the parts
	email := parts[0]
	filename := parts[1]
	contents := parts[2]
	fmt.Println(email, filename, contents)

	// Get the node ID for the email
	nodeID, err := lb.Put(email)
	if err != nil {
		// If there is an error getting the node ID, respond with an internal server error
		http.Error(w, "Error getting node ID", http.StatusInternalServerError)
		return
	}

	// Get the server address for the node ID
	serverAddress, _ := lb.Ring.Get(nodeID)
	fmt.Println("Server address:", serverAddress)
	
	// Connect to the server
	resp, err := http.Post("http://"+serverAddress+"/putListServer", "text/plain", strings.NewReader(fmt.Sprintf("%s,%s", filename, contents)))
	if err != nil {
		http.Error(w, "Error connecting to server:", http.StatusInternalServerError)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			http.Error(w, "Error closing connection to server:", http.StatusInternalServerError)
		}
	}(resp.Body)

	// Send a success response (HTTP 200 OK) to the client
	w.WriteHeader(http.StatusOK)
}


func (lb *LoadBalancer) HandleShoppingListGet(w http.ResponseWriter, r *http.Request) {

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// If there is an error reading the body, respond with a bad request error
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	// Split the body into parts using commas
	parts := strings.Split(string(body), ",")
	fmt.Println("Parts:", parts)

	// Check if the number of parts is not equal to 2
	if len(parts) != 2 {
		// If the format is invalid, respond with a bad request error
		http.Error(w, "Invalid request body format", http.StatusBadRequest)
		return
	}

	// Extract email and filename from the parts
	email := parts[0]
	filename := parts[1]
	fmt.Println(email, filename)

	// Get the node ID for the email
	nodeID, err := lb.Put(email)
	if err != nil {
		// If there is an error getting the node ID, respond with an internal server error
		http.Error(w, "Error getting node ID", http.StatusInternalServerError)
		return
	}

	// Get the server address for the node ID
	serverAddress, _ := lb.Ring.Get(nodeID)
	fmt.Println("Server address:", serverAddress)
	
	// Connect to the server
	resp, err := http.Post("http://"+serverAddress+"/getListServer", "text/plain", strings.NewReader(filename))
	if err != nil {
		http.Error(w, "Error connecting to server:", http.StatusInternalServerError)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			http.Error(w, "Error closing connection to server:", http.StatusInternalServerError)
		}
	}(resp.Body)

	if resp.StatusCode == http.StatusOK {
		// Read the file contents from the response from the server
		file_contents, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}

		// Send file to the client
		w.Write(file_contents)
		return
	}
	http.Error(w, "Error reading file", http.StatusInternalServerError)
}


func main() {
	loadBalancer := NewLoadBalancer()

	// Set up HTTP handler for load balancer
	http.HandleFunc("/connect-node", loadBalancer.HandleNodeConnection)
	http.HandleFunc("/putList", loadBalancer.HandleShoppingListPut)
	http.HandleFunc("/getList", loadBalancer.HandleShoppingListGet)
	// Start the load balancer on port 8080
	fmt.Println("Load balancer listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
