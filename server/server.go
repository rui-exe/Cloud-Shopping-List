package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
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

		// Print received data
		fmt.Printf("Received from client: %s\n", buffer[:n])

		// Send the same data back to the client
		_, err = conn.Write(buffer[:n])
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
	}()

	fmt.Println("Server listening on localhost:8080")

	for {
		// Accept a new connection
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		// Handle the connection in a new goroutine
		go handleConnection(conn)
	}
}
