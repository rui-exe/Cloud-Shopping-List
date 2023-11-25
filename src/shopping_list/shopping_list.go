package shopping_list

import (
	"encoding/json"
	"os"
)

type ShoppingList struct {
	Email string
	Items map[string]int
}

func NewShoppingList(email string) *ShoppingList {
	return &ShoppingList{
		Email: email,
		Items: make(map[string]int),
	}
}

func (sl *ShoppingList) AddItem(item string, quantity int) {
	sl.Items[item] = quantity
}

func FromJSON(data []byte) (*ShoppingList, error) {
	var sl ShoppingList
	err := json.Unmarshal(data, &sl)
	return &sl, err
}

func (sl *ShoppingList) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(sl, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile("list_storage/"+filename, data, 0644)
}

func LoadFromFile(filename string) (*ShoppingList, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return FromJSON(data)
}
