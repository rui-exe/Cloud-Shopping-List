package main

import (
	"CloudShoppingList/shopping_list"
	"fmt"
	"net"
	"os"
)

func push(email string, filename string) {
	// Connect to the server at localhost:8080
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}
	defer conn.Close()

	//Read from File
	file_contents, err := os.ReadFile("list_storage/" + filename)

	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// Send a message to the server
	message := email + "," + filename + "," + "push," + string(file_contents)
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

func pull() {
	// Connect to the server at localhost:8080
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}
	defer conn.Close()

	// Send a message to the server
	message := "pull"
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

func makeShoppingList(email string, items map[string]int) shopping_list.ShoppingList {
	shoppingList := shopping_list.NewShoppingList(email)
	for item, quantity := range items {
		shoppingList.AddItem(item, quantity)
	}
	shoppingList.SaveToFile(email + ".json")
	return *shoppingList
}

func main() {
	s_list := makeShoppingList("email", map[string]int{"item": 1})
	fmt.Println(s_list.Items)
	push("email", "email.json")
}
