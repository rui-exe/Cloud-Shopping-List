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

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 2048)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("Client closed the connection.")
			} else {
				fmt.Println("Error reading:", err)
			}
			return
		}

		message := string(buffer[:n])
		messageParts := strings.Split(message, ",")
		if len(messageParts) < 4 {
			fmt.Println("Invalid message format.")
			return
		}
		email := messageParts[0]
		filename := messageParts[1]
		command := messageParts[2]
		fileContents := messageParts[3]
		fmt.Println(email, filename, command, fileContents)

		_, err = conn.Write([]byte("Message received by server by email: " + email + " with filename: " + filename + " with command: " + command + "\n"))
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

		go handleConnection(conn)
	}
}

func main() {
	// create a server with port specified in command line
	server := NewServer(os.Args[1])
	server.Run()
}
