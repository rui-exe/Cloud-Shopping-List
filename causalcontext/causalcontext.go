package causalcontext

import (
	"CloudShoppingList/dotcloud"
)

type CausalContext struct {
	cc map[string]int
	dc *dotcloud.DotCloud
}

type Pair struct {
	Key   string
	Value int
}

func NewCausalContext(cc map[string]int) *CausalContext {
	if cc == nil {
		cc = make(map[string]int)
	}

	return &CausalContext{
		cc: cc,
		dc: dotcloud.NewCustomSet(),
	}
}

func (ctx *CausalContext) DotIn(dot Pair) bool {
	key, value := dot.Key, dot.Value
	_, exists := ctx.cc[key]
	if (exists) {return true}
	if (ctx.dc.Has(key, value)) {return true}
	return false
}

func (ctx *CausalContext) Compact() *CausalContext {
	flag := true
	for flag {
		flag = false
		for _, dot := range ctx.dc.Values() {
			key, value := dot.Key, dot.Value
			casual_context_value, exists := ctx.cc[key]
			if !exists {
				if (value == 1) {
					ctx.cc[key] = value
					ctx.dc.Delete(key, value)
					flag = true
				}
			} else {
				if (value == casual_context_value + 1) {
					ctx.cc[key] = value
					ctx.dc.Delete(key, value)
					flag = true
				} else  if (value <= casual_context_value) {
					ctx.dc.Delete(key, value)
				}
			}
		}
	}
	return ctx
}

func (ctx *CausalContext) Next(id string) Pair {
	value, exists := ctx.cc[id]
	if !exists {
		value = 0
	}
	newValue := value + 1
	return Pair{Key: id, Value: newValue}
}

func (ctx *CausalContext) MakeDot(id string) Pair {
	n := ctx.Next(id)
	ctx.cc[id] = n.Value
	return n
}

func (ctx *CausalContext) InsertDot(key string, value int, compactNow bool) {
	ctx.dc.Add(key, value)
	if compactNow {
		ctx.Compact()
	}
}

func (ctx *CausalContext) Current(id string) int {
	value, exists := ctx.cc[id]
	if !exists {
		return 0
	}
	return value
}


func (ctx *CausalContext) Join(other *CausalContext) {
	if other == ctx {
		return 
	}

	for key, value := range ctx.cc {
		other_value, exists := other.cc[key]
		if exists {
			ctx.cc[key] = max(value, other_value)
		}
	}

	for key, value := range other.cc {
		ctx.InsertDot(key, value, false)
	} 

	ctx.Compact()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
