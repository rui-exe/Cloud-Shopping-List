package main

import (
	"CloudShoppingList/consistent_hashing"
)

func main() {
	ring := consistent.NewRing()
	ring.AddNode("node1")
	ring.AddNode("node2")
	ring.PrintNodes()
}