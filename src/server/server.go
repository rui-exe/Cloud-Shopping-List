package main

import (
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Server struct {
	port           string
	nodeID         string
	loadBalancerIP string
}

func NewServer(port string) *Server {
	// generate a random node uuid
	return &Server{port: port, nodeID: uuid.New().String(), loadBalancerIP: "localhost:8080"}
}

func (s *Server) Run() {
	// Print the node ID
	fmt.Println("Node ID:", s.nodeID)
	// Connect to the load balancer with retries
	status := s.connectToLoadBalancerWithRetries(3, time.Second*2)
	if status != http.StatusOK {
		fmt.Println("Exiting...")
		return
	}
	// await incoming http messages
	http.HandleFunc("/putList", s.HandleShoppingListPut)
	fmt.Println("Server listening on port " + s.port)
	err := http.ListenAndServe(":"+s.port, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func (s *Server) connectToLoadBalancerWithRetries(maxRetries int, retryInterval time.Duration) int {
	url := fmt.Sprintf("http://%s/connect-node", s.loadBalancerIP)

	for retry := 0; retry < maxRetries; retry++ {
		// Create a buffer with the node ID and server port
		body := strings.NewReader(fmt.Sprintf("%s,%s", s.nodeID, "localhost:"+s.port))

		resp, err := http.Post(url, "text/plain", body)
		if err != nil {
			fmt.Printf("Error connecting to the load balancer (retry %d/%d): %v\n", retry+1, maxRetries, err)
			if retry == maxRetries-1 {
				break
			}
			time.Sleep(retryInterval)
			retryInterval *= 2
			continue
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(resp.Body)

		if resp.StatusCode == http.StatusOK {
			fmt.Println("Connected to the load balancer successfully.")
			return resp.StatusCode
		}
	}

	fmt.Printf("Max retries reached. Could not connect to the load balancer after %d attempts.\n", maxRetries)
	return http.StatusInternalServerError
}

func (s *Server) HandleShoppingListPut(writer http.ResponseWriter, request *http.Request) {
	fmt.Println("Received shopping list put")
}

func (s *Server) HandleShoppingListGet(writer http.ResponseWriter, request *http.Request) {

	// Read the filename from the request body
	filename, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Error reading request body", http.StatusInternalServerError)
		return
	}

	// Read the file contents
	file_contents, err := os.ReadFile("../list_storage/" + string(filename))
	if err != nil {
		http.Error(writer, "Error reading file", http.StatusInternalServerError)
		return
	}

	// Send file to the client
	_, err = writer.Write(file_contents)
	if err != nil {
		http.Error(writer, "Error writing file", http.StatusInternalServerError)
		return
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./server <port>")
		os.Exit(1)
	}
	// create an HTTP server with the specified port
	server := NewServer(os.Args[1])
	http.HandleFunc("/putListServer", server.HandleShoppingListPut)
	http.HandleFunc("/getListServer", server.HandleShoppingListGet)
	server.Run()
}