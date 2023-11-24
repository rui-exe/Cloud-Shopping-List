package main

import (
	"CloudShoppingList/shopping_list"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
)

type HashRing struct {
	servers   []string
	replicas  int
	serverMap map[string]map[string]shopping_list.ShoppingList
	/*
		serverMap:
		{
			"server1": {
				"shoppinglist1": {
					"item1": 1,
					"item2": 2,
				},
				"shoppinglist2": {
					"item1": 1,
					"item2": 2,
				},
			},
			"server2": {
				"shoppinglist1": {
					"item1": 1,
					"item2": 2,
				},
				"shoppinglist2": {
					"item1": 1,
					"item2": 2,
				},
			},
		}
	*/
}

func newHashRing(replicas int) *HashRing {
	return &HashRing{
		replicas:  replicas,
		serverMap: make(map[string]map[string]shopping_list.ShoppingList),
	}
}

func (hr *HashRing) addServer(server string) {

	for i := 0; i < hr.replicas; i++ {
		key := hashKey(fmt.Sprintf("%s-%d", server, i))
		hr.servers = append(hr.servers, server)
		hr.serverMap[key] = make(map[string]shopping_list.ShoppingList)
	}

	sort.Strings(hr.servers)
}

func (hr *HashRing) removeServer(server string) {

	var newServers []string
	for _, n := range hr.servers {
		if n != server {
			newServers = append(newServers, n)
		}
	}

	hr.servers = newServers
	for i := 0; i < hr.replicas; i++ {
		key := hashKey(fmt.Sprintf("%s-%d", server, i))
		delete(hr.serverMap, key)
	}
}

func (hr *HashRing) getServer(key string) string {

	if len(hr.servers) == 0 {
		return ""
	}

	hash := hashKey(key)
	index := sort.Search(len(hr.servers), func(i int) bool {
		return hr.servers[i] >= hash
	})

	if index == len(hr.servers) {
		index = 0
	}

	return hr.servers[index]
}

func (hr *HashRing) addShoppingListToServer(shoppingList shopping_list.ShoppingList, server string) {
	hr.serverMap[server]["shoppingList"+strconv.FormatInt(int64(len(hr.serverMap[server])), 2)] = shoppingList
}

func hashKey(key string) string {
	hash := sha512.Sum512([]byte(key))
	return hex.EncodeToString(hash[:])
}

func main() {
	hashRing := newHashRing(3)

	hashRing.addServer("S0")
	hashRing.addServer("S1")
	hashRing.addServer("S2")

	fmt.Println("Hash ring: ", hashRing.serverMap)
}
