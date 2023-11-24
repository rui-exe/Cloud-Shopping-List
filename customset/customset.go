package customset

import (
	"encoding/json"
)

type CustomSet struct {
	refs map[string]interface{}
}

func NewCustomSet() *CustomSet {
	return &CustomSet{
		refs: make(map[string]interface{}),
	}
}

func (cs *CustomSet) Add(o interface{}) {
	key := KeyFor(o)
	if _, exists := cs.refs[key]; !exists {
		cs.refs[key] = o
	}
}

func (cs *CustomSet) Delete(o interface{}) {
	key := KeyFor(o)
	delete(cs.refs, key)
}

func (cs *CustomSet) Has(o interface{}) bool {
	key := KeyFor(o)
	_, exists := cs.refs[key]
	return exists
}

func (cs *CustomSet) Values() []interface{} {
	values := make([]interface{}, 0, len(cs.refs))
	for _, value := range cs.refs {
		values = append(values, value)
	}
	return values
}

func KeyFor(o interface{}) string {
	key, _ := json.Marshal(o)
	return string(key)
}
