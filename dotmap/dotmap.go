package dotmap

import (
	"CloudShoppingList/causalcontext"
	"CloudShoppingList/crdt"
	"CloudShoppingList/customset"
)

type DotMap struct {
	cc    *causalcontext.CausalContext
	state map[string]*crdt.CRDT // Change the type of the map value according to your needs
}

func NewDotMap(cc *causalcontext.CausalContext, state map[string]*crdt.CRDT) *DotMap {
	if cc == nil {
		cc = causalcontext.NewCausalContext()
	}

	if state == nil {
		state = make(map[string]*crdt.CRDT)
	}

	return &DotMap{
		cc:    cc,
		state: state,
	}
}

func (dm *DotMap) Dots() map[string]struct{} {
	dots := make(map[string]struct{})
	for _, dotStore := range dm.state {
		for dot := range dotStore.Dots() {
			dots[dot] = struct{}{}
		}
	}
	return dots
}

func (dm *DotMap) IsBottom() bool {
	return len(dm.state) == 0
}

func (dm *DotMap) Compact() *DotMap {
	return &DotMap{
		cc:    dm.cc.Compact(),
		state: dm.state,
	}
}

func (dm *DotMap) Join(other *DotMap) *DotMap {
	return Join(dm, other)
}

func Join(self, other *DotMap) *DotMap {
	if self == nil {
		self = DotMapFromRaw(nil)
	}

	if other == nil {
		other = DotMapFromRaw(nil)
	}

	allKeys := make(map[string]struct{})
	for key := range self.state {
		allKeys[key] = struct{}{}
	}
	for key := range other.state {
		allKeys[key] = struct{}{}
	}

	newCausalContext := self.cc.Join(other.cc)
	newMap := make(map[string]*crdt.CRDT)
	result := NewDotMap(newCausalContext, newMap)
	result.Type = self.Type // Change the type according to your needs

	for key := range allKeys {
		sub1 := self.state[key]
		if sub1 != nil {
			sub1.cc = self.cc
		}

		sub2 := other.state[key]
		if sub2 != nil {
			sub2.cc = other.cc
		}

		var newSub *crdt.CRDT

		if sub1 == nil {
			newSub = sub2
		} else if sub2 == nil {
			newSub = sub1
		} else {
			newSub = JoinCRDT(sub1, sub2)
		}

		newSub.Type = sub1.Type // Change the type according to your needs

		newSub.cc = nil
		result.state[key] = newSub
	}

	return result
}

func JoinCRDT(s1, s2 *crdt.CRDT) *crdt.CRDT {
	if s1 == nil {
		return s2
	} else if s2 == nil {
		return s1
	}

	// Implement your CRDT join logic here
	// Example: return s1.Join(s2)
	return nil
}

func DotMapFromRaw(base *DotMap) *DotMap {
	cc := causalcontext.NewCausalContext(base.cc.CC)
	if base.cc != nil && base.cc.DC != nil {
		dc := customset.NewCustomSet()
		if base.cc.DC.Refs != nil {
			dc.Refs = base.cc.DC.Refs
		}
		cc.DC = dc
	}

	stateFromRaw := make(map[string]*crdt.CRDT)
	for key, value := range base.state {
		t := crdt.Type(value.Type)
		stateFromRaw[key] = t.Join(value, t.Initial())
	}

	dotMap := NewDotMap(cc, stateFromRaw)
	dotMap.Type = base.Type // Change the type according to your needs

	return dotMap
}
