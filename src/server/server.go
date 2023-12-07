package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const replicationFactor = 2

type Node struct {
	id         string
	hashId     []byte
	frontNodes []Node
	backNodes  []Node
	server     string
}

type Server struct {
	port           string
	name           string
	loadBalancerIP string
	db             *sql.DB
	nodes          []Node
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

	return &Server{port: port, name: name, loadBalancerIP: "localhost:8080", db: db, nodes: []Node{}}
}

func (s *Server) Run() {
	// Print the node ID
	fmt.Println("Name:", s.name)
	// Connect to the load balancer with retries
	status := s.connectToLoadBalancerWithRetries(3, time.Second*2)
	if status != http.StatusOK {
		fmt.Println("Exiting...")
		return
	}
}

func (s *Server) connectToLoadBalancerWithRetries(maxRetries int, retryInterval time.Duration) int {
	url := fmt.Sprintf("http://%s/connect-node", s.loadBalancerIP)

	for retry := 0; retry < maxRetries; retry++ {
		// Create a buffer with the node ID and server port
		body := strings.NewReader(fmt.Sprintf("%s,%s", s.name, "localhost:"+s.port))

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
	err := request.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(writer, "Error parsing request body", http.StatusBadRequest)
		return
	}
	email := request.FormValue("email")
	fmt.Println("Email:", email)

	file, handler, err := request.FormFile("file")
	if err != nil {
		http.Error(writer, "Error reading file", http.StatusBadRequest)
		return
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(file)

	fmt.Println("File name:", handler.Filename)

	// read the shopping list from the file
	shoppingList, err := io.ReadAll(file)
	if err != nil {
		http.Error(writer, "Error reading file", http.StatusBadRequest)
		return
	}

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
	// get the email from the url
	fmt.Println("Handling shopping list get")
	fmt.Println("")
	email := strings.TrimPrefix(request.URL.Path, "/getListServer/")
	fmt.Println("Email:", email)
	// get the shopping list from the database
	row := s.db.QueryRow("SELECT shopping_list FROM shopping_lists WHERE email = ?", email)
	var shoppingList []byte
	err := row.Scan(&shoppingList)
	if err != nil {
		http.Error(writer, "Error getting shopping list from database", http.StatusInternalServerError)
		return
	}
	// send the shopping list to the load balancer
	writer.WriteHeader(http.StatusOK)
	_, err = writer.Write(shoppingList)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Successfully sent shopping list to load balancer")

}

func (s *Server) HandleNeighboursInformation(writer http.ResponseWriter, request *http.Request) {
	// get the neighbours information from the request body
	// "nodeId:NodehashId,frontNeighbour1:frontNeighbour1HashId,frontNeighbour2:frontNeighbour2HashId
	//,backNeighbour1:backNeighbour1HashId,backNeighbour2:backNeighbour2HashId" and so on
	fmt.Println("Handling neighbours information")
	fmt.Println("")

	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Error reading request body", http.StatusBadRequest)
		return
	}
	fmt.Println("Body:", string(body))
	// split the body into lines
	lines := strings.Split(string(body), "****")
	newNodes := []Node{}
	// iterate over the lines
	for _, line := range lines {
		node := Node{}
		attributes := strings.Split(line, ",,,")
		for i, attribute := range attributes {
			if i == 0 {
				node.id = strings.Split(attribute, ":::")[0]
				node.hashId = []byte(strings.Split(attribute, ":::")[1])
			} else if strings.Contains(attribute, "frontNeighbor1") {
				frontNeighbor1 := Node{}
				frontNeighbor1.id = strings.Split(attribute, ":::")[0]
				frontNeighbor1.server = strings.Split(attribute, ":::")[1]
				frontNeighbor1.hashId = []byte(strings.Split(attribute, ":::")[2])
				node.frontNodes = append(node.frontNodes, frontNeighbor1)
			} else if strings.Contains(attribute, "frontNeighbor2") {
				frontNeighbor2 := Node{}
				frontNeighbor2.id = strings.Split(attribute, ":::")[0]
				frontNeighbor2.server = strings.Split(attribute, ":::")[1]
				frontNeighbor2.hashId = []byte(strings.Split(attribute, ":::")[2])
				node.frontNodes = append(node.frontNodes, frontNeighbor2)
			} else if strings.Contains(attribute, "backNeighbor1") {
				backNeighbor1 := Node{}
				backNeighbor1.id = strings.Split(attribute, ":::")[0]
				backNeighbor1.server = strings.Split(attribute, ":::")[1]
				backNeighbor1.hashId = []byte(strings.Split(attribute, ":::")[2])
				node.backNodes = append(node.backNodes, backNeighbor1)
			} else if strings.Contains(attribute, "backNeighbor2") {
				backNeighbor2 := Node{}
				backNeighbor2.id = strings.Split(attribute, ":::")[0]
				backNeighbor2.server = strings.Split(attribute, ":::")[1]
				backNeighbor2.hashId = []byte(strings.Split(attribute, ":::")[2])
				node.backNodes = append(node.backNodes, backNeighbor2)
			}
		}
		newNodes = append(newNodes, node)
	}
	// update the nodes array
	s.nodes = newNodes
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: ./server <port> <name>")
		return
	}
	// create an HTTP server with the specified port
	server := NewServer(os.Args[1], os.Args[2])
	http.HandleFunc("/putListServer", server.HandleShoppingListPut)
	http.HandleFunc("/getListServer/", server.HandleShoppingListGet)
	http.HandleFunc("/shareNeighboursInformation", server.HandleNeighboursInformation)
	go server.Run()
	fmt.Println("listening on port", server.port)
	log.Fatal(http.ListenAndServe(":"+server.port, nil))
}
