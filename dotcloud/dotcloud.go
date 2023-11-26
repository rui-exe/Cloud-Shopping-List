package dotcloud

import (
	"encoding/json"
	"fmt"
)

type Pair struct {
	Key   string
	Value string
}

type DotCloud struct {
	refs map[string]Pair
}

func NewCustomSet() *DotCloud {
	return &DotCloud{
		refs: make(map[string]Pair),
	}
}

func (cs *DotCloud) Add(key string, value int) {
	pair := Pair{Key: key, Value: value}
	keyStr := KeyFor(pair)
	if _, exists := cs.refs[keyStr]; !exists {
		cs.refs[keyStr] = pair
	}
}

func (cs *DotCloud) Delete(key string, value int) {
	pair := Pair{Key: key, Value: value}
	keyStr := KeyFor(pair)
	delete(cs.refs, keyStr)
}

func (cs *DotCloud) Has(key string, value int) bool {
	pair := Pair{Key: key, Value: value}
	keyStr := KeyFor(pair)
	_, exists := cs.refs[keyStr]
	return exists
}

func (cs *DotCloud) Values() []Pair{} {
	values := make([]Pair, 0, len(cs.refs))
	for _, value := range cs.refs {
		values = append(values, value)
	}
	return values
}

func KeyFor(o Pair) string {
	key, _ := json.Marshal(o)
	return string(key)
}
