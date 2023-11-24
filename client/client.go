package main

import (
	"fmt"
	"net"
)

func main() {
	// Connect to the server at localhost:8080
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}
	defer conn.Close()

	// Send a message to the server
	message := "Hello, server! How are you?"
	_, err = conn.Write([]byte(message))
	if err != nil {
		fmt.Println("Error writing:", err)
		return
	}

	// Receive and print the response from the server
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		fmt.Println("Error reading:", err)
		return
	}
	fmt.Printf("Received from server: %s\n", response[:n])
}
