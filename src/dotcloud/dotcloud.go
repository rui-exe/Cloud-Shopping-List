package dotcloud

import (
	"encoding/json"
)

type Pair struct {
	Key   string
	Value int
}

type DotCloud struct {
	Refs map[string]Pair
}

func NewCustomSet() *DotCloud {
	return &DotCloud{
		Refs: make(map[string]Pair),
	}
}

func (cs *DotCloud) Add(key string, value int) {
	pair := Pair{Key: key, Value: value}
	keyStr := KeyFor(pair)
	if _, exists := cs.Refs[keyStr]; !exists {
		cs.Refs[keyStr] = pair
	}
}

func (cs *DotCloud) Delete(key string, value int) {
	pair := Pair{Key: key, Value: value}
	keyStr := KeyFor(pair)
	delete(cs.Refs, keyStr)
}

func (cs *DotCloud) Has(key string, value int) bool {
	pair := Pair{Key: key, Value: value}
	keyStr := KeyFor(pair)
	_, exists := cs.Refs[keyStr]
	return exists
}

func (cs *DotCloud) Values() []Pair {
	values := make([]Pair, 0, len(cs.Refs))
	for _, value := range cs.Refs {
		values = append(values, value)
	}
	return values
}

func KeyFor(o Pair) string {
	key, _ := json.Marshal(o)
	return string(key)
}
