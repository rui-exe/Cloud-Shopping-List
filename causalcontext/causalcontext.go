package causalcontext

import (
	"CloudShoppingList/customset"
	"strconv"
)

type CausalContext struct {
	cc map[string]int
	dc *customset.CustomSet
}

func NewCausalContext(cc map[string]int) *CausalContext {
	if cc == nil {
		cc = make(map[string]int)
	}

	return &CausalContext{
		cc: cc,
		dc: customset.NewCustomSet(),
	}
}

func (ctx *CausalContext) DotIn(dot [2]string) bool {
	key, value := dot[0], dot[1]
	count, exists := ctx.cc[key]
	return value <= count || ctx.dc.Has(dot)
}

func (ctx *CausalContext) Compact() *CausalContext {
	// Compact DC to CC if possible
	for _, dot := range ctx.dc.Values() {
		key, value := dot[0], dot[1]
		existing, exists := ctx.cc[key]
		if !exists || existing < value {
			ctx.cc[key] = value
		}
		ctx.dc.Delete(dot)
	}
	return ctx
}

func (ctx *CausalContext) Next(id string) [2]string {
	value, exists := ctx.cc[id]
	if !exists {
		value = 0
	}
	newValue := value + 1
	return [2]string{id, strconv.Itoa(newValue)}
}

func (ctx *CausalContext) MakeDot(id string) [2]string {
	n := ctx.Next(id)
	ctx.cc[n[0]] = n[1]
	return n
}

func (ctx *CausalContext) InsertDot(key, value string, compactNow bool) {
	if value == "" {
		value = "null"
	}
	ctx.dc.Add([2]string{key, value})
	if compactNow {
		ctx.Compact()
	}
}

func (ctx *CausalContext) Join(other *CausalContext) *CausalContext {
	if other == nil {
		other = NewCausalContext(nil)
	} else {
		other.Compact()
	}
	ctx.Compact()

	result := make(map[string]int)

	for k := range ctx.cc {
		result[k] = max(ctx.cc[k], other.cc[k])
	}

	for k := range other.cc {
		result[k] = max(ctx.cc[k], other.cc[k])
	}

	return NewCausalContext(result)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
