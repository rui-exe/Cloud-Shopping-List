package crdt

import (
	"fmt"
	"sync"
)

// CCounter represents a Convergent Counter.
type CCounter struct {
	count int
}

// Increment increments the counter value.
func (c *CCounter) Increment() {
	c.count++
}

// Decrement decrements the counter value.
func (c *CCounter) Decrement() {
	c.count--
}

// GetValue returns the current value of the counter.
func (c *CCounter) GetValue() int {
	return c.count
}

// ORMap represents an Observed-Remove Map.
type ORMap struct {
	data map[string]CCounter
}

// AddArticle adds an article to the map with an initial quantity.
func (m *ORMap) AddArticle(article string, quantity int, done chan struct{}) {
	if _, ok := m.data[article]; !ok {
		m.data[article] = CCounter{}
	}

	// Increment the quantity using the CCounter.
	for i := 0; i < quantity; i++ {
		m.data[article].Increment()
	}

	done <- struct{}{}
}

// RemoveArticle removes an article from the map with a specified quantity.
func (m *ORMap) RemoveArticle(article string, quantity int, done chan struct{}) {
	if _, ok := m.data[article]; !ok {
		done <- struct{}{}
		return // Article not found.
	}

	// Ensure the quantity to remove does not exceed the current quantity.
	currentQuantity := m.data[article].GetValue()
	if quantity > currentQuantity {
		quantity = currentQuantity
	}

	// Decrement the quantity using the CCounter.
	for i := 0; i < quantity; i++ {
		m.data[article].Decrement()
	}

	// If the quantity becomes zero, remove the article from the map.
	if m.data[article].GetValue() == 0 {
		delete(m.data, article)
	}

	done <- struct{}{}
}

// GetState returns the current state of the ORMap.
func (m *ORMap) GetState() map[string]int {
	state := make(map[string]int, len(m.data))
	for article, counter := range m.data {
		state[article] = counter.GetValue()
	}
	return state
}

// Merge merges two ORMaps into a single map.
func (m *ORMap) Merge(other *ORMap, done chan struct{}) {
	// Merge the two maps.
	for article, counter := range other.data {
		if _, ok := m.data[article]; !ok {
			m.data[article] = CCounter{}
		}
		m.data[article].count += counter.count
	}

	done <- struct{}{}
}

func main() {
	// Create two instances of ORMap representing two users.
	user1Map := &ORMap{data: make(map[string]CCounter)}
	user2Map := &ORMap{data: make(map[string]CCounter)}

	// Channels to signal the completion of operations.
	user1Done := make(chan struct{})
	user2Done := make(chan struct{})
	mergeDone := make(chan struct{})

	// User 1 adds items to the shopping list concurrently.
	go user1Map.AddArticle("apple", 2, user1Done)
	go user1Map.AddArticle("banana", 3, user1Done)

	// User 2 adds and removes items from the shopping list concurrently.
	go user2Map.AddArticle("apple", 1, user2Done)
	go user2Map.RemoveArticle("banana", 1, user2Done)

	// Wait for user operations to complete.
	<-user1Done
	<-user1Done
	<-user2Done
	<-user2Done

	// Merge the two maps to synchronize changes.
	go user1Map.Merge(user2Map, mergeDone)
	<-mergeDone

	// Get the final state of the shopping list for user 1.
	finalState := user1Map.GetState()

	// Print the final state.
	fmt.Println("Final Shopping List State:")
	printState(finalState)
}

func printState(state map[string]int) {
	for article, quantity := range state {
		fmt.Printf("%s: %d\n", article, quantity)
	}
}
