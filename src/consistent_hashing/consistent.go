package consistent

import (
    "hash/crc32"
    "sort"
    "sync"
	"fmt"
)

type Ring struct {
    Nodes Nodes
    sync.RWMutex
}

type Nodes []Node

type Node struct {
    Id     string
    HashId uint32
}

func (n Nodes) Len() int           { return len(n) }
func (n Nodes) Less(i, j int) bool { return n[i].HashId < n[j].HashId }
func (n Nodes) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

func NewRing() *Ring {
	return &Ring{
		Nodes: Nodes{},
	}
}

func NewNode(id string) *Node {
    return &Node{
        Id:     id,
        HashId: crc32.ChecksumIEEE([]byte(id)),
    }
}

func (r *Ring) AddNode(id string) {
    r.Lock()
    defer r.Unlock()

    node := NewNode(id)
    r.Nodes = append(r.Nodes, *node)
    sort.Sort(r.Nodes)
}

func (r *Ring) Get(key string) (string, error) {
    r.RLock()
    defer r.RUnlock()

    if len(r.Nodes) == 0 {
        return "", fmt.Errorf("ring is empty")
    }

    searchfn := func(i int) bool {
        return r.Nodes[i].HashId >= crc32.ChecksumIEEE([]byte(key))
    }

    i := sort.Search(r.Nodes.Len(), searchfn)
    if i >= r.Nodes.Len() {
        i = 0
    }

    return r.Nodes[i].Id, nil
}

func (r *Ring) PrintNodes() {
	for _, node := range r.Nodes {
		fmt.Printf("Node %s with hash %d\n", node.Id, node.HashId)
	}
}