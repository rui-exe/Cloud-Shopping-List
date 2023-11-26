package main

import (
	"fmt"
	"CloudShoppingList/causalcontext"
)

// ORMap represents an Observed-Remove Map.
type ORMap struct {
	data map[string]RWCounter
	cc *causalcontext.CausalContext
	id int
}

func NewOrmMap(new_id int) *ORMap {
	return &ORMap{
		data: make(map[string]RWCounter),
		cc: causalcontext.NewCausalContext(nil),
		id: new_id,
	}
}

func NewAuxOrmMap() *ORMap {
	return &ORMap{
		data: make(map[string]RWCounter),
		cc: causalcontext.NewCausalContext(nil),
	}
}

func (m *ORMap) getCasualContext() *causalcontext.CausalContext {
	return m.cc
}

func (m *ORMap) erase(key string) ORMap {
	new_orm_map := NewAuxOrmMap()
	_, exists := m.data[key]
	if exists {
		rwcounter := m.data[key]
		aux_counter := rwcounter.reset()
		new_orm_map.cc = aux_counter.cc
		m.erase(key)
	}
	return *new_orm_map
}

func (m *ORMap) reset() ORMap {
	new_orm_map := NewAuxOrmMap()
	if len(m.data) > 0 {
		for key, _ := range m.data {
			rwcounter := m.data[key]
			aux_counter := rwcounter.reset()
			new_orm_map.cc.Join(aux_counter.cc)
		}
		m.data = make(map[string]RWCounter)
	}
	return *new_orm_map
}

func (m *ORMap) get(key string) RWCounter{
	_, exists := m.data[key]
	if exists {
		return m.data[key]
	} else {
		new_rwcounter := NewRWCounter(m.id, m.cc)
		m.data[key] = *new_rwcounter
		return *new_rwcounter
	}
}

func (m *ORMap) join(other *ORMap) {
	ic := *m.cc

    for key, otherRWCounter := range other.data {
        if rwcounter, exists := m.data[key]; exists {
            rwcounter.join(&otherRWCounter)
            m.data[key] = rwcounter
			m.cc = &ic
        } else {
            m.data[key] = otherRWCounter
			m.cc = &ic
        }
    }

    for key, rwcounter := range m.data {
        if _, exists := other.data[key]; !exists {
            emptyRWCounter := NewRWCounter(m.id, other.cc)
            rwcounter.join(emptyRWCounter)
            m.data[key] = rwcounter
			m.cc = &ic
        }
    }

    m.cc.Join(other.cc)
}

func main() {
	
}

func printState(state map[string]int) {
	for article, quantity := range state {
		fmt.Printf("%s: %d\n", article, quantity)
	}
}

