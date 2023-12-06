package crdt

import (
	"CloudShoppingList/causalcontext"
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"os"
)

// ORMap represents an Observed-Remove Map.
type List struct {
	data      map[string]*DotStore
	cc        *causalcontext.CausalContext
	replicaID string
}

type DotStore struct {
	data map[Dot]Counter
}

type Counter struct {
	Positive int
	Negative int
}

type Dot struct {
	ReplicaID string
	Counter   int
}

func NewList(id string) *List {
	list := &List{
		data:      make(map[string]*DotStore),
		cc:        causalcontext.NewCausalContext(map[string]int{id: 0}),
		replicaID: id,
	}

	for key := range list.data {
		list.data[key] = &DotStore{data: make(map[Dot]Counter)}
	}
	return list
}

func (list *List) Increment(key string) {
	if list.data[key] == nil {
		list.data[key] = &DotStore{data: make(map[Dot]Counter)}
	}
	list.data[key].update(list.replicaID, Counter{Positive: 1, Negative: 0}, list.cc)
}

func (list *List) Decrement(key string) {
	if list.data[key] == nil {
		list.data[key] = &DotStore{data: make(map[Dot]Counter)}
	}
	list.data[key].update(list.replicaID, Counter{Positive: 0, Negative: 1}, list.cc)
}

func (list *List) Remove(key string) {
	delete(list.data, key)
}

func (DotStore *DotStore) update(replicaID string, change Counter, cc *causalcontext.CausalContext) {
	version := -1
	for dot := range DotStore.data {
		if dot.ReplicaID == replicaID {
			version = dot.Counter
		}
	}
	currentCasualContexValue := cc.Current(replicaID)

	if currentCasualContexValue != version {
		DotStore.fresh(replicaID, cc)
		currentCasualContexValue = cc.Current(replicaID)
	}

	counter := DotStore.data[Dot{ReplicaID: replicaID, Counter: currentCasualContexValue}]
	counter.Positive += change.Positive
	counter.Negative += change.Negative
	DotStore.data[Dot{ReplicaID: replicaID, Counter: currentCasualContexValue}] = counter
}

func (DotStore *DotStore) fresh(replicaID string, cc *causalcontext.CausalContext) {
	pair := cc.Next(replicaID)
	DotStore.data[Dot{ReplicaID: pair.Key, Counter: pair.Value}] = Counter{Positive: 0, Negative: 0}
	cc.MakeDot(pair.Key)
}

func (DotStore *DotStore) value() int {
	value := 0
	if len(DotStore.data) == 0 {
		return value
	}
	for _, counter := range DotStore.data {
		value += counter.Positive - counter.Negative
	}
	return value
}

func (list *List) join(other *List) {
	originalData := make(map[string]*DotStore)
	for key, dotStore := range list.data {
		newDotStore := &DotStore{data: make(map[Dot]Counter)}
		for dot, counter := range dotStore.data {
			newDotStore.data[dot] = counter
		}
		originalData[key] = newDotStore
	}

	for key, dotStore := range other.data {
		if _, exists := list.data[key]; exists {
			for dot, counter := range dotStore.data {
				if _, exists := list.data[key].data[dot]; exists {
					list.data[key].data[dot] = max(list.data[key].data[dot], counter)
				} else {
					if dot.Counter > list.cc.Current(dot.ReplicaID) {
						list.data[key].add(dot, counter)
					}
				}
			}
		} else {
			newDotStore := &DotStore{data: make(map[Dot]Counter)}
			for dot, counter := range dotStore.data {
				newDotStore.data[dot] = counter
			}
			list.data[key] = newDotStore
		}
	}

	for key, dotStore := range originalData {
		if _, exists := other.data[key]; !exists {
			for dot := range dotStore.data {
				if dot.Counter <= other.cc.Current(dot.ReplicaID) {
					list.data[key].remove(dot)
				}
			}
		}
	}

	list.cc.Join(other.cc)
	for _, dotStore := range other.data {
		dotStore.fresh(other.replicaID, other.cc)
	}
}

func (DotStore *DotStore) getDot() {
	for dot := range DotStore.data {
		print(dot.ReplicaID)
		print(dot.Counter)
		println("Here")
	}
}

func (DotStore *DotStore) add(dot Dot, counter Counter) {
	DotStore.data[dot] = counter
}

func (DotStore *DotStore) remove(dot Dot) {
	delete(DotStore.data, dot)
}

func FromGOB64(s string) *List {
	list := &List{}
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		fmt.Println("Error decoding list:", err)
	}
	b := bytes.Buffer{}
	b.Write(data)
	d := gob.NewDecoder(&b)
	err = d.Decode(list)
	if err != nil {
		fmt.Println("Error decoding list:", err)
	}
	return list
}

func (list *List) ToGOB64() string {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(list)
	if err != nil {
		fmt.Println("Error encoding list:", err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

func printList(list *List) {
	println("List: ", list.replicaID)
	for key, dotStore := range list.data {
		print("  ", key)
		print(":")
		println(dotStore.value())
	}
	println()
}

func (list *List) init() {
	gob.Register(&List{})
	gob.Register(&DotStore{})
	gob.Register(&Counter{})
	gob.Register(&Dot{})
}

func (list *List) SaveToFile(filename string, clientID string) {
	list.init()
	data := list.ToGOB64()
	fmt.Println("Saving list to file...")
	err := os.WriteFile("../list_storage/"+clientID+"/"+filename, []byte(data), 0644)
	if err != nil {
		fmt.Println("Error saving list to file:", err)
	}
	fmt.Println("Saved list to file successfully")
}

func LoadFromFile(filename string, clientID string) *List {
	list := &List{}
	data, err := os.ReadFile("../list_storage/" + clientID + "/" + filename)
	if err != nil {
		fmt.Println("Error loading list from file:", err)
	}
	list.init()
	list = FromGOB64(string(data))
	return list
}

func Test() {
	list1 := NewList("1")
	list1.Increment("friend")
	list1.Increment("friend")
	list1.Increment("newItem")
	// list1.Increment("newItem2")

	list2 := NewList("2")
	list2.join(list1)

	printList(list1)
	printList(list2)

	list2.Remove("friend")
	list2.Increment("newItem2")
	println("Remove friend from list2 and add newItem2:")
	printList(list2)

	list1.Increment("friend")
	list1.Increment("friend")
	list1.Increment("friend")
	println("Add 3 friend to list1:")
	printList(list1)
	list1.Decrement("friend")
	println("Remove 1 friend from list1:")
	printList(list1)
	list1.join(list2)

	printList(list1)
	println()
}

func max(c1 Counter, c2 Counter) Counter {
	return Counter{Positive: maxInt(c1.Positive, c2.Positive), Negative: maxInt(c1.Negative, c2.Negative)}
}

func maxInt(i1 int, i2 int) int {
	if i1 > i2 {
		return i1
	}
	return i2
}
