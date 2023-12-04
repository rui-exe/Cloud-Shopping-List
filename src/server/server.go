package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type Server struct {
	port           string
	name           string
	nodeID         string
	loadBalancerIP string
	db             *sql.DB
}

func NewServer(port string, name string) *Server {
	// generate a random node uuid
	// open SQLite database
	db, err := sql.Open("sqlite3", fmt.Sprintf("../node_storage/%s.db", name))
	if err != nil {
		fmt.Println("Error opening database:", err)
		os.Exit(1)
	}
	fmt.Println("Opened database successfully")

	// create tables if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS shopping_lists (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL,
			shopping_list BLOB NOT NULL
		);
	`)
	if err != nil {
		fmt.Println("Error creating table:", err)
		os.Exit(1)
	}

	return &Server{port: port, name: name, nodeID: uuid.New().String(), loadBalancerIP: "localhost:8080", db: db}
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
	// split the request body into email and shopping list
	fmt.Println("Handling shopping list put")
	fmt.Println("")
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Error reading request body", http.StatusInternalServerError)
		return
	}
	parts := strings.Split(string(body), ",")
	if len(parts) != 2 {
		http.Error(writer, "Invalid request body format", http.StatusBadRequest)
		return
	}
	email := parts[0]
	shoppingList := parts[1]
	// insert the shopping list into the database
	_, err = s.db.Exec("INSERT INTO shopping_lists (email, shopping_list) VALUES (?, ?)", email, shoppingList)
	if err != nil {
		http.Error(writer, "Error inserting shopping list into database", http.StatusInternalServerError)
		return
	}
	// send a success response to the load balancer
	writer.WriteHeader(http.StatusOK)
	fmt.Println("Successfully inserted shopping list into database")

}

func (s *Server) HandleShoppingListGet(writer http.ResponseWriter, request *http.Request) {
	//TODO: implement
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: ./server <port> <name>")
		return
	}
	// create an HTTP server with the specified port
	server := NewServer(os.Args[1], os.Args[2])
	http.HandleFunc("/putListServer", server.HandleShoppingListPut)
	http.HandleFunc("/getListServer", server.HandleShoppingListGet)
	server.Run()
}