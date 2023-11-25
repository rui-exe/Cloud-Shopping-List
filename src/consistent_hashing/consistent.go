package consistent

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"sync"
)

type Ring struct {
	Nodes Nodes
	sync.RWMutex
	virtualNodes  int
	RealToVirtual map[string][]string
}

type Nodes []Node

type Node struct {
	Id        string
	HashId    []byte
	Server    string
	IsVirtual bool
}

func (n Nodes) Len() int           { return len(n) }
func (n Nodes) Less(i, j int) bool { return bytes.Compare(n[i].HashId, n[j].HashId) == -1 }
func (n Nodes) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

func NewRing() *Ring {
	return &Ring{
		Nodes:         Nodes{},
		virtualNodes:  3,
		RealToVirtual: make(map[string][]string),
	}
}

func NewNode(id, server string, isVirtual bool) *Node {
	hash := sha256.New()
	hash.Write([]byte(id))
	hashId := hash.Sum(nil)

	return &Node{
		Id:        id,
		HashId:    hashId,
		Server:    server,
		IsVirtual: isVirtual,
	}
}

func (r *Ring) AddNode(id, server string) {
	r.Lock()
	defer r.Unlock()
	realNode := NewNode(id, server, false)
	r.Nodes = append(r.Nodes, *realNode)
	r.RealToVirtual[id] = []string{}
	for i := 0; i < r.virtualNodes; i++ {
		virtualId := id + strconv.Itoa(i)
		node := NewNode(virtualId, server, true)
		r.Nodes = append(r.Nodes, *node)
		r.RealToVirtual[id] = append(r.RealToVirtual[id], virtualId)
	}
	sort.Sort(r.Nodes)
}

func (r *Ring) Get(key string) (string, error) {
	r.RLock()
	defer r.RUnlock()

	if len(r.Nodes) == 0 {
		return "", fmt.Errorf("ring is empty")
	}

	hash := sha256.New()
	hash.Write([]byte(key))
	keyHash := hash.Sum(nil)

	searchfn := func(i int) bool {
		return bytes.Compare(r.Nodes[i].HashId, keyHash) != -1
	}

	i := sort.Search(r.Nodes.Len(), searchfn)
	if i >= r.Nodes.Len() {
		i = 0
	}

	return r.Nodes[i].Server, nil
}

func (r *Ring) Put(email string) (string, error) {
	r.RLock()
	defer r.RUnlock()

	if len(r.Nodes) == 0 {
		return "", fmt.Errorf("ring is empty")
	}

	hash := sha256.New()
	hash.Write([]byte(email))
	emailHash := hash.Sum(nil)

	searchfn := func(i int) bool {
		return bytes.Compare(r.Nodes[i].HashId, emailHash) != -1
	}

	i := sort.Search(r.Nodes.Len(), searchfn)
	if i >= r.Nodes.Len() {
		i = 0
	}

	fmt.Println("email: ", email)
	fmt.Println("hash: ", hex.EncodeToString(emailHash))
	fmt.Println("node: ", r.Nodes[i].Id)
	fmt.Println("server: ", r.Nodes[i].Server)

	return r.Nodes[i].Server, nil
}

func (r *Ring) PrintNodes() {
	// iterate over the realtoVirtual map
	for k, v := range r.RealToVirtual {
		fmt.Println("Real node: ", k)
		for _, virtualNode := range v {
			fmt.Println("Virtual node: ", virtualNode)
		}
	}
}
