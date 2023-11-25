package main

import (
	consistent "CloudShoppingList/consistent_hashing"
	shopping_list "CloudShoppingList/shopping_list"
)

func main() {
	ring := consistent.NewRing()
	ring.AddNode("node1", "localhost:8080") // Add a server information
	ring.AddNode("node2", "localhost:8081")
	ring.PrintNodes()
	// create a shopping list
	list := shopping_list.NewShoppingList("rui132@gmail.com")
	list.AddItem("apple", 1)
	list.AddItem("banana", 2)
	_, err := ring.Put(list.Email)
	if err != nil {
		panic(err)
	}
}
