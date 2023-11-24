package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 2048)
	for {
		// Read data from the client
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("Client closed the connection.")
			} else {
				fmt.Println("Error reading:", err)
			}
			return
		}

		// parse message
		message := string(buffer[:n])
		messageParts := strings.Split(message, ",")
		email := messageParts[0]
		filename := messageParts[1]
		command := messageParts[2]
		fileContents := messageParts[3]
		fmt.Println(email)
		fmt.Println(filename)
		fmt.Println(command)
		fmt.Println(fileContents)

		// send response
		_, err = conn.Write([]byte("Message received by server by email: " + email + " with filename: " + filename + " with command: " + command + "\n"))
		if err != nil {
			fmt.Println("Error writing:", err)
			return
		}
	}
}

func main() {
	// Listen for incoming connections on port 8080
	listener, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		fmt.Printf("Received signal: %v. Shutting down...\n", sig)
		listener.Close()
		os.Exit(0)
	}()

	fmt.Println("Server listening on localhost:8080")

	for {
		// Accept a new connection
		conn, err := listener.Accept()
		if conn != nil && err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		defer conn.Close()

		// Handle the connection in a new goroutine
		go handleConnection(conn)
	}
}
