package main

import (
	"CloudShoppingList/crdt"
	"bytes"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
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
			email_hash TEXT NOT NULL,
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

	hash := sha256.New()
	hash.Write([]byte(email))
	emailHash := hash.Sum(nil)

	row := s.db.QueryRow("SELECT shopping_list FROM shopping_lists WHERE email_hash = ?", string(emailHash))
	var shoppingListDatabase []byte
	row.Scan(&shoppingListDatabase)
	// read the shopping list from the file
	shoppingListClient, err := io.ReadAll(file)
	if err != nil {
		http.Error(writer, "Error reading file", http.StatusBadRequest)
		return
	}

	// Join the shopping list from the database and the shopping list from the client
	// using the CRDT implementation
	if len(shoppingListDatabase) != 0 {
		listClient := crdt.FromGOB64(string(shoppingListClient))
		listDatabase := crdt.FromGOB64(string(shoppingListDatabase))
		listDatabase.Join(listClient)
		_, err = s.db.Exec("UPDATE shopping_lists SET shopping_list = ? WHERE email_hash = ?", []byte(listDatabase.ToGOB64()), string(emailHash))
		if err != nil {
			http.Error(writer, "Error updating shopping list in database", http.StatusInternalServerError)
			return
		}
	} else {
		// insert the shopping list into the database
		_, err = s.db.Exec("INSERT INTO shopping_lists (email, email_hash, shopping_list) VALUES (?, ?, ?)", email, string(emailHash), shoppingListClient)
		if err != nil {
			http.Error(writer, "Error inserting shopping list into database", http.StatusInternalServerError)
			return
		}
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

	hash := sha256.New()
	hash.Write([]byte(email))
	emailHash := hash.Sum(nil)

	// get the shopping list from the database
	row := s.db.QueryRow("SELECT shopping_list FROM shopping_lists WHERE email_hash = ?", string(emailHash))
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

func (s *Server) HandleRequestKeys(_ http.ResponseWriter, _ *http.Request) {
	fmt.Println("Handling request keys")
	for _, node := range s.nodes {
		done := false
		for i, frontNode := range node.frontNodes {
			fmt.Println("Requesting my keys from front node number " + strconv.Itoa(i+1) + " with port " + frontNode.server)
			nodeHash := node.hashId
			firstBackNodeHash := node.backNodes[0].hashId
			// send the node id and the first back node id to the first front node
			body := strings.NewReader(fmt.Sprintf("%s,%s,%s", s.port, string(nodeHash), string(firstBackNodeHash)))
			for attempt := 1; attempt <= 3; attempt++ {
				resp, err := http.Post(fmt.Sprintf("http://%s/sendMeKeys", frontNode.server), "text/plain", body)
				if err != nil {
					fmt.Printf("Error on attempt %d: %s\n", attempt, err)
					time.Sleep(time.Second * 2) // Adjust the delay between retries as needed
					continue
				}

				defer func(Body io.ReadCloser) {
					err := Body.Close()
					if err != nil {
						fmt.Println(err)
					}
				}(resp.Body)

				if resp.StatusCode == http.StatusOK {
					fmt.Println("Successfully sent request to front node number " + strconv.Itoa(i+1) + " with port " + frontNode.server)
					done = true
					break
				}

				fmt.Printf("Attempt %d failed with status code: %d\n", attempt, resp.StatusCode)
				time.Sleep(time.Second * 2) // Adjust the delay between retries as needed
			}
			if done {
				break
			}
		}
		fmt.Println("Done requesting keys from front nodes of node" + node.id)
	}
}

func (s *Server) HandleSendMeKeys(writer http.ResponseWriter, request *http.Request) {
	// parse the request body
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Error reading request body", http.StatusBadRequest)
		return
	}
	serverPort := strings.Split(string(body), ",")[0]
	fmt.Println("Sending requested keys to server with port " + string(serverPort))
	nodeHash := strings.Split(string(body), ",")[1]
	firstBackNodeHash := strings.Split(string(body), ",")[2]
	if bytes.Compare([]byte(nodeHash), []byte(firstBackNodeHash)) == 1 {
		// query the database for all the shopping lists
		rows, err := s.db.Query("SELECT email, email_hash, shopping_list FROM shopping_lists where email_hash > ? AND email_hash <= ?", firstBackNodeHash, nodeHash)
		if err != nil {
			fmt.Println("Error querying database:", err)
			return
		}
		defer func(rows *sql.Rows) {
			err := rows.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(rows)
		// iterate over the shopping lists
		for rows.Next() {
			// send the row to the server with the specified port
			var email string
			var emailHash string
			var shoppingList []byte
			err = rows.Scan(&email, &emailHash, &shoppingList)
			if err != nil {
				fmt.Println("Error scanning row:", err)
				return
			}
			fmt.Println("Sending shopping list with email " + email + " to server with port " + serverPort)
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			err = writer.WriteField("email", email)
			if err != nil {
				fmt.Println("Error writing email field:", err)
				return
			}
			part, err := writer.CreateFormFile("file", email)
			if err != nil {
				fmt.Println("Error creating form file:", err)
				return
			}
			_, err = part.Write(shoppingList)
			if err != nil {
				fmt.Println("Error writing to form file:", err)
				return
			}
			err = writer.Close()
			if err != nil {
				fmt.Println("Error closing writer:", err)
				return
			}

			req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%s/putListServer", serverPort), body)
			if err != nil {
				fmt.Println("Error creating new request:", err)
				return
			}
			req.Header.Set("Content-Type", writer.FormDataContentType())
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("Error sending request:", err)
				return
			}
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

			fmt.Println("Successfully sent shopping list" + email + " to server with port " + serverPort)
		}
	} else {
		// query the database for all the shopping lists
		rows, err := s.db.Query("SELECT email, email_hash, shopping_list FROM shopping_lists where email_hash > ? OR email_hash < ?", firstBackNodeHash, nodeHash)
		if err != nil {
			fmt.Println("Error querying database:", err)
			return
		}
		defer func(rows *sql.Rows) {
			err := rows.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(rows)
		// iterate over the shopping lists
		for rows.Next() {
			// send the row to the server with the specified port
			var email string
			var emailHash string
			var shoppingList []byte
			err = rows.Scan(&email, &emailHash, &shoppingList)
			if err != nil {
				fmt.Println("Error scanning row:", err)
				return
			}
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			err = writer.WriteField("email", email)
			if err != nil {
				fmt.Println("Error writing email field:", err)
				return
			}
			part, err := writer.CreateFormFile("file", email)
			if err != nil {
				fmt.Println("Error creating form file:", err)
				return
			}
			_, err = part.Write(shoppingList)
			if err != nil {
				fmt.Println("Error writing to form file:", err)
				return
			}
			err = writer.Close()
			if err != nil {
				fmt.Println("Error closing writer:", err)
				return
			}

			req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%s/putListServer", serverPort), body)
			if err != nil {
				fmt.Println("Error creating new request:", err)
				return
			}
			req.Header.Set("Content-Type", writer.FormDataContentType())
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("Error sending request:", err)
				return
			}
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

			fmt.Println("Successfully sent shopping list" + email + " to server with port " + serverPort)
		}
	}
}

func (server *Server) retrieveCRDTsInRange(startHash, endHash string) ([]string, error) {
	var crdts []string
	var rows *sql.Rows
	var err error
	if startHash < endHash {
		rows, err = server.db.Query("SELECT email, email_hash, shopping_list FROM shopping_lists where email_hash > ? AND email_hash < ?", startHash, endHash)
	} else {
		rows, err = server.db.Query("SELECT email, email_hash, shopping_list FROM shopping_lists where email_hash > ? OR email_hash <= ?", startHash, endHash)
	}

	if err != nil {
		fmt.Println("Error querying database:", err)
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(rows)

	// iterate over the shopping lists
	for rows.Next() {
		// send the row to the server with the specified port
		var email string
		var emailHash string
		var shoppingList []byte
		err = rows.Scan(&email, &emailHash, &shoppingList)
		if err != nil {
			return nil, err
		}
		crdt := string(shoppingList) + "####" + email + "####" + emailHash
		crdts = append(crdts, crdt)
	}

	return crdts, nil
}

func (server *Server) Sync() {
	for _, node := range server.nodes {
		for _, frontNeighbor := range node.frontNodes {
			fmt.Println("Sending hash space between the first back neighbour and the node itself to the front neighbor with port " + frontNeighbor.server)
			var crdts []string // Use your actual CRDT type
			crdts, _ = server.retrieveCRDTsInRange(string(node.hashId), string(node.backNodes[0].hashId))

			
			additionalData := strings.Join(crdts, "++++")

			requestBody := fmt.Sprintf("%s****%s", string(node.hashId), string(node.backNodes[0].hashId))
			if len(additionalData) > 0 {
				requestBody += "****" + additionalData
			}

			body := strings.NewReader(requestBody)
			resp, err := http.Post(fmt.Sprintf("http://%s/syncShoppingList", frontNeighbor.server), "text/plain", body)
			if err != nil {
				fmt.Println("Error sending request:", err)
				continue
			}
			defer resp.Body.Close()

			responseData, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("Error reading response body:", err)
				continue
			}
			if len(responseData) != 0 {
				for _, shoppingList := range strings.Split(string(responseData), "****") {
					receivedShoppingList := crdt.FromGOB64(strings.Split(shoppingList, "####")[0])
					email := strings.Split(shoppingList, "####")[1]
					
					emailHash := strings.Split(shoppingList, "####")[2]
					row := server.db.QueryRow("SELECT shopping_list FROM shopping_lists WHERE email_hash = ?", emailHash)
					var shoppingListDatabase []byte
					row.Scan(&shoppingListDatabase)
					if len(shoppingListDatabase) != 0 {
						listDatabase := crdt.FromGOB64(string(shoppingListDatabase))
						listDatabase.Join(receivedShoppingList)
						newShoppingList := []byte(listDatabase.ToGOB64())
						_, err = server.db.Exec("UPDATE shopping_lists SET shopping_list = ? WHERE email_hash = ?", string(newShoppingList), emailHash)
						if err != nil {
							fmt.Println("Error updating shopping list in database:", err)
							continue
						}
					} else {
						_, err = server.db.Exec("INSERT INTO shopping_lists (email, email_hash, shopping_list) VALUES (?, ?, ?)", email, emailHash, string(shoppingList))
						if err != nil {
							fmt.Println("Error inserting shopping list into database:", err)
							return
						}
					}
				}
			}
		}
	}
}

func (s *Server) HandleSyncShoppingList(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	hashes := strings.Split(string(body), "****")
	if len(hashes) < 2 {
		http.Error(w, "Invalid hash range", http.StatusBadRequest)
		return
	}
	startHash, endHash := hashes[0], hashes[1]

	var additionalCRDTs []string
	if len(hashes) > 2 {
		additionalCRDTs = strings.Split(hashes[2], "++++")
	}

	var rows *sql.Rows
	if startHash < endHash {
		rows, err = s.db.Query("SELECT email, email_hash, shopping_list FROM shopping_lists where email_hash > ? AND email_hash <= ?", string(startHash), string(endHash))
	} else {
		rows, err = s.db.Query("SELECT email, email_hash, shopping_list FROM shopping_lists where email_hash > ? OR email_hash <= ?", string(startHash), string(endHash))
	}
	if err != nil {
		http.Error(w, "Error querying database", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var crdts []string
	for rows.Next() {
		var email string
		var emailHash string
		var shoppingList []byte
		err = rows.Scan(&email, &emailHash, &shoppingList)
		if err != nil {
			http.Error(w, "Error scanning row", http.StatusInternalServerError)
			return
		}
		crdt := string(shoppingList) + "####" + email + "####" + emailHash
		crdts = append(crdts, crdt)
	}

	

	response := strings.Join(crdts, "****")
	_, err = w.Write([]byte(response))
	if err != nil {
		http.Error(w, "Error writing response", http.StatusInternalServerError)
	}

	//update the database
	if len(additionalCRDTs) != 0 {
		for _, additionalCRDT := range additionalCRDTs {
			receivedCRDT := strings.Split(additionalCRDT, "####")[0]
			email := strings.Split(additionalCRDT, "####")[1]
			emailHash := strings.Split(additionalCRDT, "####")[2]
			receivedShoppingList := crdt.FromGOB64(string(receivedCRDT))
			row := s.db.QueryRow("SELECT shopping_list FROM shopping_lists WHERE email_hash = ?", emailHash)
			var shoppingListDatabase []byte
			row.Scan(&shoppingListDatabase)
			if len(shoppingListDatabase) != 0 {
				listDatabase := crdt.FromGOB64(string(additionalCRDT))
				listDatabase.Join(receivedShoppingList)
				newShoppingList := []byte(listDatabase.ToGOB64())
				_, err = s.db.Exec("UPDATE shopping_lists SET shopping_list = ? WHERE email_hash = ?", string(newShoppingList), emailHash)
				if err != nil {
					fmt.Println("Error updating shopping list in database:", err)
					continue
				}
			} else {
				_, err = s.db.Exec("INSERT INTO shopping_lists (email, email_hash, shopping_list) VALUES (?, ?, ?)", email, emailHash, string(additionalCRDT))
				if err != nil {
					fmt.Println("Error inserting shopping list into database:", err)
					return
				}
			}
		}
	}

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
	http.HandleFunc("/requestKeys", server.HandleRequestKeys)
	http.HandleFunc("/sendMeKeys", server.HandleSendMeKeys)
	http.HandleFunc("/syncShoppingList", server.HandleSyncShoppingList)
	// sync the shopping lists
	go func() {
		for {
			time.Sleep(time.Second * 5)
			server.Sync()
		}
	}()
	go server.Run()
	fmt.Println("listening on port", server.port)
	log.Fatal(http.ListenAndServe(":"+server.port, nil))
}
