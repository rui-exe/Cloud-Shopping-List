package main

import (
	"CloudShoppingList/causalcontext"
)

// ORMap represents an Observed-Remove Map.
type List struct {
	data map[string]*DotStore
	cc *causalcontext.CausalContext
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
	Counter int
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
	list.data[key].update(list.replicaID , Counter{Positive:1, Negative:0}, list.cc)
}

func (list *List) Decrement(key string) {
	if list.data[key] == nil {
		list.data[key] = &DotStore{data: make(map[Dot]Counter)}
	}
	list.data[key].update(list.replicaID , Counter{Positive:0, Negative:1}, list.cc)
}

func (list *List) Remove(key string) {
	delete(list.data, key)
}

func (DotStore *DotStore) update(replicaID string, change Counter, cc *causalcontext.CausalContext) {
	version := -1
	for dot, _ := range DotStore.data {
		if (dot.ReplicaID == replicaID) {
			version = dot.Counter
		}
	}
	currentCasualContexValue := cc.Current(replicaID)

	if (currentCasualContexValue != version) {
		DotStore.fresh(replicaID, cc)
		currentCasualContexValue = cc.Current(replicaID)
	}

	counter := DotStore.data[Dot{ReplicaID:replicaID, Counter:currentCasualContexValue}]
	counter.Positive += change.Positive
	counter.Negative += change.Negative
	DotStore.data[Dot{ReplicaID:replicaID, Counter:currentCasualContexValue}] = counter
}

func (DotStore *DotStore) fresh(replicaID string, cc *causalcontext.CausalContext) {
	pair := cc.Next(replicaID)
	DotStore.data[Dot{ReplicaID:pair.Key, Counter:pair.Value}] = Counter{Positive:0, Negative:0}
	cc.MakeDot(pair.Key)
}

func (DotStore *DotStore) value() int {
	value := 0
	if (len(DotStore.data) == 0) {
		return value
	}
	for _, counter := range DotStore.data {
		value += counter.Positive - counter.Negative
	}
	return value
}

func (list *List) join (other *List) {
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
					if (dot.Counter > list.cc.Current(dot.ReplicaID)) {
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
			for dot, _ := range dotStore.data {
				if (dot.Counter <= other.cc.Current(dot.ReplicaID)) {
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

func (DotStore *DotStore) getDot (){
	for dot, _ := range DotStore.data {
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

func printList(list *List) {
	println("List: ", list.replicaID)
	for key, dotStore := range list.data {
		print("  ", key)
		print(":")
		println(dotStore.value())
	}
	println()
}

func main() {
	list1 := NewList("1")
	list1.Increment("friend")
	list1.Increment("friend")
	list1.Increment("newItem")
	list1.Increment("newItem2")
    list2 := NewList("2")
	list2.join(list1)
	printList(list1)
	printList(list2)
	list2.Remove("friend")
	printList(list2)
	list1.Increment("friend")
	list1.Increment("friend")
	list1.Increment("friend")
	list1.Decrement("friend")
	list1.join(list2)
	println(list1.data["friend"].value())
	printList(list1)
	println()	
}

func max(c1 Counter, c2 Counter) Counter {
	return Counter{Positive:maxInt(c1.Positive, c2.Positive), Negative:maxInt(c1.Negative, c2.Negative)}
}

func maxInt(i1 int, i2 int) int {
	if (i1 > i2) {
		return i1
	} else {
		return i2
	}
}


