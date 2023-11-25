package main

import (
	consistent "CloudShoppingList/consistent_hashing"
)

func main() {
	ring := consistent.NewRing()
	ring.AddNode("node1", "localhost:8080") // Add a server information
	ring.AddNode("node2", "localhost:8081")
	ring.PrintNodes()
}
