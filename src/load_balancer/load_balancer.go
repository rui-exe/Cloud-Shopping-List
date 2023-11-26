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
	fmt.Println("Received shopping list put")
}

func main() {
	loadBalancer := NewLoadBalancer()

	// Set up HTTP handler for load balancer
	http.HandleFunc("/connect-node", loadBalancer.HandleNodeConnection)
	http.HandleFunc("/putList", loadBalancer.HandleShoppingListPut)
	// Start the load balancer on port 8080
	fmt.Println("Load balancer listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
