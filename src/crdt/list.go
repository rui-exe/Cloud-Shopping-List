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
	Data      map[string]*DotStore
	Cc        *causalcontext.CausalContext
	ReplicaID string
}

type DotStore struct {
	Data map[Dot]Counter
}

type Counter struct {
	Positive int
	Negative int
}

type Dot struct {
	ReplicaID string
	Counter   int
}

func (list *List) GetID() string {
	return list.ReplicaID
}

func NewList(id string) *List {
	list := &List{
		Data:      make(map[string]*DotStore),
		Cc:        causalcontext.NewCausalContext(map[string]int{id: 0}),
		ReplicaID: id,
	}

	for key := range list.Data {
		list.Data[key] = &DotStore{Data: make(map[Dot]Counter)}
	}
	return list
}

func (list *List) Increment(key string) {
	if list.Data[key] == nil {
		list.Data[key] = &DotStore{Data: make(map[Dot]Counter)}
	}
	list.Data[key].update(list.ReplicaID, Counter{Positive: 1, Negative: 0}, list.Cc)
}

func (list *List) Decrement(key string) {
	if list.Data[key] == nil {
		list.Data[key] = &DotStore{Data: make(map[Dot]Counter)}
	}
	list.Data[key].update(list.ReplicaID, Counter{Positive: 0, Negative: 1}, list.Cc)
}

func (list *List) Remove(key string) {
	delete(list.Data, key)
}

func (DotStore *DotStore) update(replicaID string, change Counter, cc *causalcontext.CausalContext) {
	version := -1
	for dot := range DotStore.Data {
		if dot.ReplicaID == replicaID {
			version = dot.Counter
		}
	}
	currentCasualContexValue := cc.Current(replicaID)

	if currentCasualContexValue != version {
		DotStore.fresh(replicaID, cc)
		currentCasualContexValue = cc.Current(replicaID)
	}

	counter := DotStore.Data[Dot{ReplicaID: replicaID, Counter: currentCasualContexValue}]
	counter.Positive += change.Positive
	counter.Negative += change.Negative
	DotStore.Data[Dot{ReplicaID: replicaID, Counter: currentCasualContexValue}] = counter
}

func (DotStore *DotStore) fresh(replicaID string, cc *causalcontext.CausalContext) {
	pair := cc.Next(replicaID)
	DotStore.Data[Dot{ReplicaID: pair.Key, Counter: pair.Value}] = Counter{Positive: 0, Negative: 0}
	cc.MakeDot(pair.Key)
}

func (DotStore *DotStore) Value() int {
	value := 0
	if len(DotStore.Data) == 0 {
		return value
	}
	for _, counter := range DotStore.Data {
		value += counter.Positive - counter.Negative
	}
	return value
}

func (list *List) Join(other *List) {
	originalData := make(map[string]*DotStore)
	for key, dotStore := range list.Data {
		newDotStore := &DotStore{Data: make(map[Dot]Counter)}
		for dot, counter := range dotStore.Data {
			newDotStore.Data[dot] = counter
		}
		originalData[key] = newDotStore
	}

	for key, dotStore := range other.Data {
		if _, exists := list.Data[key]; exists {
			for dot, counter := range dotStore.Data {
				if _, exists := list.Data[key].Data[dot]; exists {
					list.Data[key].Data[dot] = max(list.Data[key].Data[dot], counter)
				} else {
					if dot.Counter > list.Cc.Current(dot.ReplicaID) {
						list.Data[key].add(dot, counter)
					}
				}
			}
		} else {
			newDotStore := &DotStore{Data: make(map[Dot]Counter)}
			for dot, counter := range dotStore.Data {
				newDotStore.Data[dot] = counter
			}
			list.Data[key] = newDotStore
		}
	}

	for key, dotStore := range originalData {
		if _, exists := other.Data[key]; !exists {
			for dot := range dotStore.Data {
				if dot.Counter <= other.Cc.Current(dot.ReplicaID) {
					delete(list.Data, key)
				}
			}
		}
	}

	list.Cc.Join(other.Cc)
	for _, dotStore := range other.Data {
		dotStore.fresh(other.ReplicaID, other.Cc)
	}
}

func (DotStore *DotStore) GetDot() {
	for dot := range DotStore.Data {
		print(dot.ReplicaID)
		print(dot.Counter)
		println("Here")
	}
}

func (DotStore *DotStore) add(dot Dot, counter Counter) {
	DotStore.Data[dot] = counter
}

func (DotStore *DotStore) remove(dot Dot) {
	delete(DotStore.Data, dot)
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
	println("List: ", list.ReplicaID)
	for key, dotStore := range list.Data {
		print("  ", key)
		print(":")
		println(dotStore.Value())
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
		return nil
	}
	list.init()
	list = FromGOB64(string(data))
	return list
}

func Teste2() {
	list1 := NewList("1")
	list1.Increment("arroz")
	list1.Increment("arroz")
	list1.Increment("arroz")

	list2 := NewList("2")
	list2.Increment("massa")
	
	
	list3 := FromGOB64(list1.ToGOB64())
	

	list3.Join(list2)

	list1.Join(list3)

	list2.Join(list3)

	list1.Remove("arroz")

	list3.Join(list1)	

	list2.Join(list3)

	printList(list1)
	printList(list2)
	printList(list3)

}

func Test() {
	list1 := NewList("1")
	list1.Increment("arroz")
	list1.Increment("arroz")
	list1.Increment("arroz")
	list1.Increment("arroz")
	list1.Increment("massa")
	list1.Increment("massa")
	list1.Increment("massa")
	// list1.Increment("newItem2")

	list2 := NewList("2")
	list2.Join(list1)

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
	list1.Join(list2)

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
