package main

import (
	"CloudShoppingList/shopping_list"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"
	"io"
	"net/http"
)

type Client struct {
	email string
	loadBalancerIP string
}

func NewClient(email string) *Client {
	return &Client{email: email, loadBalancerIP: "localhost:8080"}
}

func (c *Client) push(filename string, maxRetries int, retryInterval time.Duration) int {

	url := fmt.Sprintf("http://%s/putList", c.loadBalancerIP)

	file_contents, err := os.ReadFile("../list_storage/" + filename)

	fmt.Println("File contents:", string(file_contents))

	if err != nil {
		fmt.Println("Error reading file:", err)
		return -1
	}

	for retry := 0; retry < maxRetries; retry++ {

		body := strings.NewReader(fmt.Sprintf("%s,%s,%s", c.email, filename, hex.EncodeToString(file_contents[:])))

		fmt.Println("Body:", body)

		resp, err := http.Post(url, "text/plain", body)
		if err != nil {
			fmt.Printf("Error connecting to the server (retry %d/%d): %v\n", retry+1, maxRetries, err)
			if retry == maxRetries-1 {
				break
			}
			time.Sleep(retryInterval)
			retryInterval *= 2
			continue
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(resp.Body)

		if resp.StatusCode == http.StatusOK {
			fmt.Println("Pushed to the server successfully.")
			return resp.StatusCode
		}
	}

	fmt.Printf("Max retries reached. Could not connect to the load balancer after %d attempts.\n", maxRetries)
	return http.StatusInternalServerError
}

func (c *Client) pull(filename string, maxRetries int, retryInterval time.Duration) int {

	url := fmt.Sprintf("http://%s/getList", c.loadBalancerIP)

	for retry := 0; retry < maxRetries; retry++ {

		body := strings.NewReader(fmt.Sprintf("%s,%s", c.email, filename))

		resp, err := http.Post(url, "text/plain", body)
		if err != nil {
			fmt.Printf("Error connecting to the server (retry %d/%d): %v\n", retry+1, maxRetries, err)
			if retry == maxRetries-1 {
				break
			}
			time.Sleep(retryInterval)
			retryInterval *= 2
			continue
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(resp.Body)

		if resp.StatusCode == http.StatusOK {
			fmt.Println("Pulled from the server successfully.")
			return resp.StatusCode
		}
		fmt.Printf("Error pulling from the server: %d.\n", resp.StatusCode)
	}

	fmt.Printf("Max retries reached. Could not connect to the load balancer after %d attempts.\n", maxRetries)
	return http.StatusInternalServerError
}

func makeShoppingList(email string, items map[string]int) shopping_list.ShoppingList {
	shoppingList := shopping_list.NewShoppingList(email)
	for item, quantity := range items {
		shoppingList.AddItem(item, quantity)
	}
	shoppingList.SaveToFile(email + ".json")
	return *shoppingList
}

func main() {
	newClient := NewClient("email")
	makeShoppingList(newClient.email, map[string]int{"item": 1})
	newClient.push("email.json", 3, time.Second*2)
	//newClient.pull("email.json", 3, time.Second*2)
}
