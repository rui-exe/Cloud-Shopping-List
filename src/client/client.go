package main

import (
	"CloudShoppingList/crdt"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
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

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		err := writer.WriteField("email", filename)
		if err != nil {
			fmt.Println("Error writing to form field:", err)
			return 0
		}

		part, err := writer.CreateFormFile("file", filename)

		if err != nil {
			fmt.Println("Error creating form file:", err)
			return -1
		}
		_, err = part.Write(file_contents)

		if err != nil {
			fmt.Println("Error writing to form file:", err)
			return -1
		}

		err = writer.Close()

		if err != nil {
			fmt.Println("Error closing writer:", err)
			return -1
		}

		fmt.Println("Body:", body)

		req, err := http.NewRequest("POST", url, body)

		if err != nil {
			fmt.Println("Error creating request:", err)
			return -1
		}

		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := http.DefaultClient.Do(req)

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
		fmt.Printf("Error pushing to the server: %d.\n", resp.StatusCode)
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

func (c *Client) makeShoppingList(email string) {
	list := crdt.NewList(email)
	fmt.Print("How many items do you want the list to have: ")
	var numItems int
	_, err := fmt.Scanln(&numItems)
	if err != nil {
		fmt.Println("Error scanning input:", err)
		return
	}
	for i := 0; i < numItems; i++ {
		fmt.Print("Enter item name: ")
		var itemName string
		_, err := fmt.Scanln(&itemName)
		if err != nil {
			fmt.Println("Error scanning input:", err)
			return
		}
		fmt.Print("Enter item quantity: ")
		var itemQuantity int
		_, err2 := fmt.Scanln(&itemQuantity)
		if err2 != nil {
			fmt.Println("Error scanning input:", err2)
			return
		}
		for j := 0; j < itemQuantity; j++ {
			list.Increment(itemName)
		}
	}
	//save list to file
	list.SaveToFile(email)
}

func (c *Client) menu() {
	fmt.Println("")
	fmt.Println("Welcome to the shopping list app, " + c.email)
	fmt.Println("1. Make shopping list")
	fmt.Println("2. Show shopping list from file")
	fmt.Println("3. Push shopping list")
	fmt.Println("4. Pull shopping list")
	fmt.Println("5. Exit")
	fmt.Println("")
	// receive input
	fmt.Print("Enter your choice: ")
	var choice int
	_, err := fmt.Scanln(&choice)
	if err != nil {
		fmt.Println("Error scanning input:", err)
		return
	}
	switch choice {
	case 1:
		fmt.Println("")
		fmt.Println("Do you want the list associated with your email? (y/n)")
		var answer string
		_, err := fmt.Scanln(&answer)
		if err != nil {
			fmt.Println("Error scanning input:", err)
			return
		}
		if answer == "n" {
			fmt.Print("Enter the email of the list you want to make: ")
			var email string
			_, err := fmt.Scanln(&email)
			if err != nil {
				fmt.Println("Error scanning input:", err)
				return
			}
			fmt.Println("Making shopping list for", email+"...")
			c.makeShoppingList(email)
		}
		if answer == "y" {
			c.makeShoppingList(c.email)
		}
		break
	case 2:
		fmt.Println("")
		fmt.Print("Enter the email of the list you want to show: ")
		var email string
		_, err := fmt.Scanln(&email)
		if err != nil {
			fmt.Println("Error scanning input:", err)
			return
		}
		fmt.Println("Showing shopping list for", email+"...")
		list := crdt.LoadFromFile(email)
		for key, value := range list.Data {
			fmt.Println(key, value.Value())
		}
	case 3:
		fmt.Println("")
		fmt.Print("Enter the email of the list you want to push: ")
		var email string
		_, err := fmt.Scanln(&email)
		if err != nil {
			fmt.Println("Error scanning input:", err)
			return
		}
		fmt.Println("Pushing shopping list for", email+"...")
		c.push(email, 3, time.Second*2)
		fmt.Println("Pushed shopping list for", email+" successfully")
		break
	case 4:
		break
	case 5:
		os.Exit(0)
	default:
		fmt.Println("Invalid choice")
	}
}

func main() {
	fmt.Print("Enter your email: ")
	var email string
	_, err := fmt.Scanln(&email)
	if err != nil {
		fmt.Println("Error scanning input:", err)
		return
	}
	client := NewClient(email)
	for {
		client.menu()
	}
}
