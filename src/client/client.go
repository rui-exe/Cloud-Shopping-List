package main

import (
	"CloudShoppingList/shopping_list"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	email          string
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

func makeShoppingList(email string) {
	shoppingList := shopping_list.NewShoppingList(email)
	shoppingList.AddItem("Milk", 2)
	// store it locally in the list_storage folder
	fmt.Println("Saving shopping list locally...")
	shoppingList.SaveToFile(email + ".json")
}

func (c *Client) menu() {
	fmt.Println("")
	fmt.Println("Welcome to the shopping list app, " + c.email)
	fmt.Println("1. Make shopping list")
	fmt.Println("2. Push shopping list")
	fmt.Println("3. Pull shopping list")
	fmt.Println("4. Exit")
	fmt.Println("")
	// receive input
	fmt.Print("Enter your choice: ")
	var choice int
	fmt.Scanln(&choice)
	switch choice {
	case 1:
		fmt.Println("")
		fmt.Println("Do you want the list associated with your email? (y/n)")
		var answer string
		fmt.Scanln(&answer)
		if answer == "n" {
			fmt.Print("Enter the email of the list you want to make: ")
			var email string
			fmt.Scanln(&email)
			fmt.Println("Making shopping list for", email+"...")
			makeShoppingList(email)
		}
		if answer == "y" {
			makeShoppingList(c.email)
		}
	case 2:
		break
	case 3:
		break
	case 4:
		os.Exit(0)
	default:
		fmt.Println("Invalid choice")
	}
}

func main() {
	fmt.Print("Enter your email: ")
	var email string
	fmt.Scanln(&email)
	client := NewClient(email)
	for {
		client.menu()
	}
}
