package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

type Server struct {
	port string
}

func NewServer(port string) *Server {
	return &Server{port: port}
}

func handleConnections(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 2048)
	n, err := conn.Read(buffer)
	if err != nil {
		if err.Error() == "EOF" {
			fmt.Println("Client closed the connection.")
		} else {
			_, err = conn.Write([]byte("Error reading, please try again.\n"))
			if err != nil {
				fmt.Println("Error writing:", err)
			}
		}
		return
	}

	message := string(buffer[:n])
	messageParts := strings.Split(message, ",")
	if len(messageParts) < 4 {
		_, err = conn.Write([]byte("Invalid message format, please try again.\n"))
		if err != nil {
			fmt.Println("Error writing:", err)
		}
		return
	}

	email := messageParts[0]
	filename := messageParts[1]
	command := messageParts[2]
	contents := messageParts[3]
	fmt.Println(email, filename, command, contents)

	switch command {
	case "push":
		// Write to file
		file, err := os.Create("../list_storage/" + filename)
		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}
		defer file.Close()

		_, err = file.WriteString(string(contents))
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}

		// Send response to client
		_, err = conn.Write([]byte("File successfully written.\n"))
		if err != nil {
			fmt.Println("Error writing:", err)
			return
		}

	case "pull":
		// Read from file
		file_contents, err := os.ReadFile("../list_storage/" + filename)
		if err != nil {
			fmt.Println("Error reading file:", err)
			return
		}

		// Send file to the client
		_, err = conn.Write(file_contents)
		if err != nil {
			fmt.Println("Error writing:", err)
			return
		}
	}	
}

func (s *Server) Run() {
	listener, err := net.Listen("tcp", "localhost:"+s.port)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	isShuttingDown := false

	go func() {
		sig := <-sigCh
		fmt.Printf("Received signal: %v. Shutting down...\n", sig)
		isShuttingDown = true
		listener.Close()
		os.Exit(0)
	}()

	fmt.Println("Server listening on localhost:" + s.port)

	for {
		if isShuttingDown {
			break
		}

		conn, err := listener.Accept()
		if err != nil {
			if !isShuttingDown {
				fmt.Println("Error accepting connection:", err)
			}
			continue
		}

		go handleConnections(conn)
	}
}

func main() {
	// create a server with port specified in command line
	server := NewServer(os.Args[1])
	server.Run()
}