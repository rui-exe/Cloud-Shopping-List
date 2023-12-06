package causalcontext

import (
	"CloudShoppingList/dotcloud"
)

type CausalContext struct {
	Cc map[string]int
	Dc *dotcloud.DotCloud
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
		Cc: cc,
		Dc: dotcloud.NewCustomSet(),
	}
}

func (ctx *CausalContext) DotIn(dot Pair) bool {
	key, value := dot.Key, dot.Value
	_, exists := ctx.Cc[key]
	if exists {
		return true
	}
	if ctx.Dc.Has(key, value) {
		return true
	}
	return false
}

func (ctx *CausalContext) Compact() *CausalContext {
	flag := true
	for flag {
		flag = false
		for _, dot := range ctx.Dc.Values() {
			key, value := dot.Key, dot.Value
			casual_context_value, exists := ctx.Cc[key]
			if !exists {
				if value == 1 {
					ctx.Cc[key] = value
					ctx.Dc.Delete(key, value)
					flag = true
				}
			} else {
				if value == casual_context_value+1 {
					ctx.Cc[key] = value
					ctx.Dc.Delete(key, value)
					flag = true
				} else if value <= casual_context_value {
					ctx.Dc.Delete(key, value)
				}
			}
		}
	}
	return ctx
}

func (ctx *CausalContext) Next(id string) Pair {
	value, exists := ctx.Cc[id]
	if !exists {
		value = 0
	}
	newValue := value + 1
	return Pair{Key: id, Value: newValue}
}

func (ctx *CausalContext) MakeDot(id string) Pair {
	n := ctx.Next(id)
	ctx.Cc[id] = n.Value
	return n
}

func (ctx *CausalContext) InsertDot(key string, value int, compactNow bool) {

	ctx.Dc.Add(key, value)
	if compactNow {
		ctx.Compact()
	}
}

func (ctx *CausalContext) Current(id string) int {
	value, exists := ctx.Cc[id]
	if !exists {
		return 0
	}
	return value
}

func (ctx *CausalContext) Join(other *CausalContext) {
	if other == ctx {
		return
	}

	for key, value := range ctx.Cc {
		other_value, exists := other.Cc[key]
		if exists {
			ctx.Cc[key] = max(value, other_value)
		}
	}

	for key, value := range other.Cc {
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
