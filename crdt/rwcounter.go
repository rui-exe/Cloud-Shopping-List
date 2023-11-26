package main

import (
	"CloudShoppingList/causalcontext"
)


type RWCounter struct {
	id int
	cc *causalcontext.CausalContext
}

func NewRWCounter(new_id int, new_casual_context *causalcontext.CausalContext) *RWCounter {
	return &RWCounter{
		id: new_id,
		cc: new_casual_context,
	}
}

func (rwcounter *RWCounter) reset() RWCounter {
	return RWCounter{}
}

func (rwcounter *RWCounter) context() *causalcontext.CausalContext {
	return rwcounter.cc
}

//join function
func (rwcounter *RWCounter) join(other_rwcounter *RWCounter) RWCounter {
		return *rwcounter
}