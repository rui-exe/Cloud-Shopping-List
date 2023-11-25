package consistent

import (
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

type Ring struct {
	Nodes Nodes
	sync.RWMutex
	virtualNodes int // Add a field to store the number of virtual nodes
}

type Nodes []Node

type Node struct {
	Id     string
	HashId uint32
	Server string // Add a field to store the server information
}

func (n Nodes) Len() int           { return len(n) }
func (n Nodes) Less(i, j int) bool { return n[i].HashId < n[j].HashId }
func (n Nodes) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

func NewRing() *Ring {
	return &Ring{
		Nodes:        Nodes{},
		virtualNodes: 10, // Initialize the number of virtual nodes to 10
	}
}

func NewNode(id, server string) *Node {
	return &Node{
		Id:     id,
		HashId: crc32.ChecksumIEEE([]byte(id)),
		Server: server,
	}
}

func (r *Ring) AddNode(id, server string) {
	r.Lock()
	defer r.Unlock()

	for i := 0; i < r.virtualNodes; i++ {
		virtualId := id + strconv.Itoa(i)
		node := NewNode(virtualId, server)
		r.Nodes = append(r.Nodes, *node)
	}

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

	return r.Nodes[i].Server, nil
}

func (r *Ring) PrintNodes() {
	for _, node := range r.Nodes {
		fmt.Printf("Node %s with hash %d is associated with server %s\n", node.Id, node.HashId, node.Server)
	}
}
