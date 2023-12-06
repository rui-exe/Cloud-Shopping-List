package consistent

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"
	"strconv"
	"sync"
)

type Ring struct {
	Nodes Nodes
	sync.RWMutex
	virtualNodes      int
	RealToVirtual     map[string][]string
	ReplicationFactor int
}

type Nodes []Node

type Node struct {
	Id         string
	HashId     []byte
	Server     string
	IsVirtual  bool
	RealNodeId string
	FrontNodes []Node
	BackNodes  []Node
}

func (n Nodes) Len() int           { return len(n) }
func (n Nodes) Less(i, j int) bool { return bytes.Compare(n[i].HashId, n[j].HashId) == -1 }
func (n Nodes) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

func NewRing() *Ring {
	return &Ring{
		Nodes:             Nodes{},
		virtualNodes:      3,
		RealToVirtual:     make(map[string][]string),
		ReplicationFactor: 2,
	}
}

func NewNode(id, server string, isVirtual bool, realNodeId string) *Node {
	hash := sha256.New()
	hash.Write([]byte(id))
	hashId := hash.Sum(nil)

	return &Node{
		Id:         id,
		HashId:     hashId,
		Server:     server,
		IsVirtual:  isVirtual,
		RealNodeId: realNodeId,
		FrontNodes: []Node{},
		BackNodes:  []Node{},
	}
}

func (r *Ring) AddNode(id, server string) {
	r.Lock()
	defer r.Unlock()
	realNode := NewNode(id, server, false, "")
	r.Nodes = append(r.Nodes, *realNode)
	r.RealToVirtual[id] = []string{}
	for i := 0; i < r.virtualNodes; i++ {
		virtualId := id + strconv.Itoa(i)
		node := NewNode(virtualId, server, true, id)
		r.Nodes = append(r.Nodes, *node)
		r.RealToVirtual[id] = append(r.RealToVirtual[id], virtualId)
	}
	sort.Sort(r.Nodes)
	for _, node := range r.Nodes {
		r.GetNodeFrontNeighbors(node.Id)
		r.GetNodeBackNeighbors(node.Id)
	}
}

func (r *Ring) RemoveNode(id string) {
	// removes a real node and its virtual nodes from the hash_ring
}

func (r *Ring) GetNodeFrontNeighbors(id string) []Node {
	if len(r.Nodes) == 0 {
		return nil
	}
	var neighbors []Node
	for i := 0; i < len(r.Nodes); i++ {
		if r.Nodes[i].Id == id {
			idx := i
			parentId := r.Nodes[i].RealNodeId
			if parentId == "" {
				parentId = r.Nodes[i].Id
			}
			forbiddenIds := make(map[string]bool)
			forbiddenIds[parentId] = true
			// Get the next 2 nodes in the nodes array excluding virtual nodes from the same real node
			for j := 1; j <= r.ReplicationFactor; {
				nextIndex := (idx + j) % len(r.Nodes)
				if nextIndex == i {
					break
				}
				nextNode := r.Nodes[nextIndex]
				if nextNode.IsVirtual {
					//check if the virtual node belongs to the same real node
					if forbiddenIds[nextNode.RealNodeId] {
						idx++
						continue
					}
					neighbors = append(neighbors, nextNode)
					forbiddenIds[nextNode.RealNodeId] = true
					j++
				} else {
					if forbiddenIds[nextNode.Id] {
						idx++
						continue
					}
					neighbors = append(neighbors, nextNode)
					forbiddenIds[nextNode.Id] = true
					j++
				}
			}
			r.Nodes[i].FrontNodes = neighbors
			break
		}
	}
	return neighbors
}

func (r *Ring) GetNodeBackNeighbors(id string) []Node {
	if len(r.Nodes) == 0 {
		return nil
	}
	var neighbors []Node
	for i := 0; i < len(r.Nodes); i++ {
		if r.Nodes[i].Id == id {
			idx := i
			parentId := r.Nodes[i].RealNodeId
			if parentId == "" {
				parentId = r.Nodes[i].Id
			}
			forbiddenIds := make(map[string]bool)
			forbiddenIds[parentId] = true
			// Get the next 2 nodes in the nodes array excluding virtual nodes from the same real node
			for j := 1; j <= r.ReplicationFactor; {
				nextIndex := idx - j
				if nextIndex < 0 {
					nextIndex = len(r.Nodes) + nextIndex
				}
				if nextIndex == i {
					break
				}
				nextNode := r.Nodes[nextIndex]
				if nextNode.IsVirtual {
					//check if the virtual node belongs to the same real node
					if forbiddenIds[nextNode.RealNodeId] {
						idx--
						continue
					}
					neighbors = append(neighbors, nextNode)
					forbiddenIds[nextNode.RealNodeId] = true
					j++
				} else {
					if forbiddenIds[nextNode.Id] {
						idx--
						continue
					}
					neighbors = append(neighbors, nextNode)
					forbiddenIds[nextNode.Id] = true
					j++
				}
			}
			r.Nodes[i].BackNodes = neighbors
			break
		}
	}
	return neighbors
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

func (r *Ring) Put(email string) ([]string, error) {
	r.RLock()
	defer r.RUnlock()

	if len(r.Nodes) == 0 {
		return nil, fmt.Errorf("ring is empty")
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
	servers := []string{r.Nodes[i].Server}

	parentId := r.Nodes[i].Id

	if r.Nodes[i].IsVirtual {
		parentId = r.Nodes[i].RealNodeId
	}

	forbiddenIds := make(map[string]bool)
	forbiddenIds[parentId] = true

	// Determine the next two nodes for replication
	for j := 1; j <= r.ReplicationFactor; {
		idToCheck := ""
		if r.Nodes[(i+j)%len(r.Nodes)].IsVirtual {
			idToCheck = r.Nodes[(i+j)%len(r.Nodes)].RealNodeId
		} else {
			idToCheck = r.Nodes[(i+j)%len(r.Nodes)].Id
		}
		if !forbiddenIds[idToCheck] {
			servers = append(servers, r.Nodes[(i+j)%len(r.Nodes)].Server)
			forbiddenIds[idToCheck] = true
			j++
		} else {
			i++
		}
	}

	return servers, nil
}

func (r *Ring) PrintNodes() {
	// iterate over the nodes array
	fmt.Println("Printing nodes")
	for _, node := range r.Nodes {
		fmt.Print("Node ID: ", node.Id)
		fmt.Println(node.HashId)
	}
}
func (r *Ring) PrintNeighbors() {
	// iterate over the nodes array
	fmt.Println("Printing neighbors")
	for _, node := range r.Nodes {
		fmt.Print("Node ID: ", node.Id)
		fmt.Println(node.HashId)
		fmt.Println("Front neighbors: ")
		for _, frontNode := range node.FrontNodes {
			fmt.Print(frontNode.Id + " ")
			fmt.Println(frontNode.HashId)

		}
		fmt.Println("Back neighbors: ")
		for _, backNode := range node.BackNodes {
			fmt.Print(backNode.Id + " ")
			fmt.Println(backNode.HashId)
		}
	}
}
