package main

import (
	"CloudShoppingList/crdt"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
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

	file_contents, err := os.ReadFile("../list_storage/" + c.email + "/" + filename)

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
		time.Sleep(time.Duration(time.Second * 2))
		retryInterval *= 2
	}

	fmt.Printf("Max retries reached. Could not connect to the load balancer after %d attempts.\n", maxRetries)
	return http.StatusInternalServerError
}

func (c *Client) pull(filename string, maxRetries int, retryInterval time.Duration) int {

	url := fmt.Sprintf("http://%s/list/%s", c.loadBalancerIP, filename)

	for retry := 0; retry < maxRetries; retry++ {

		req, err := http.NewRequest("GET", url, nil)

		if err != nil {
			fmt.Println("Error creating request:", err)
			return -1
		}

		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			fmt.Printf("Error connecting to the server (retry %d/%d): %v\n", retry+1, maxRetries, err)
			if retry == maxRetries-1 {
				break
			}
			time.Sleep(time.Duration(time.Second * 2))
			retryInterval *= 2
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Println("Pulled from the server successfully.")
			//save to file
			file, err := os.Create("../list_storage/" + c.email + "/" + filename)
			if err != nil {
				fmt.Println("Error creating file:", err)
				return -1
			}
			defer file.Close()
			_, err = io.Copy(file, resp.Body)
			if err != nil {
				fmt.Println("Error copying response body to file:", err)
				return -1
			}

			fmt.Println("Saved to file successfully.")
			return resp.StatusCode
		}
		fmt.Printf("Error pulling from the server: %d.\n", resp.StatusCode)
		time.Sleep(time.Duration(time.Second * 2))
		retryInterval *= 2
	}

	fmt.Printf("Max retries reached. Could not connect to the load balancer after %d attempts.\n", maxRetries)
	return http.StatusInternalServerError
}

func (c *Client) makeShoppingList(email string) {
	list := crdt.NewList(c.email)
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
	list.SaveToFile(email, c.email)
}

func (c *Client) editList(list *crdt.List) (*crdt.List, error) {
	for {
		fmt.Println("")
		fmt.Println("Current list:")
		for key, value := range list.Data {
			fmt.Println(key, value.Value())
		}
		fmt.Println("")
		fmt.Println("1. Add item")
		fmt.Println("2. Remove item")
		fmt.Println("3. Edit item quantity")
		fmt.Println("4. Exit")
		fmt.Println("")
		// receive input
		fmt.Print("Enter your choice: ")
		var choice int
		_, err := fmt.Scanln(&choice)
		if err != nil {
			fmt.Println("Error scanning input:", err)
			return list, err
		}
		switch choice {
		case 1:
			fmt.Print("Enter item name: ")
			var itemName string
			_, err := fmt.Scanln(&itemName)
			if err != nil {
				fmt.Println("Error scanning input:", err)
				return list, err
			}
			fmt.Print("Enter item quantity: ")
			var itemQuantity int
			_, err2 := fmt.Scanln(&itemQuantity)
			if err2 != nil {
				fmt.Println("Error scanning input:", err2)
				return list, err2
			}
			for j := 0; j < itemQuantity; j++ {
				list.Increment(itemName)
			}
			break
		case 2:
			fmt.Print("Enter item name: ")
			var itemName string
			_, err := fmt.Scanln(&itemName)
			if err != nil {
				fmt.Println("Error scanning input:", err)
				return list, err
			}
			list.Remove(itemName)
			break
		case 3:
			fmt.Print("Enter item name: ")
			var itemName string
			_, err := fmt.Scanln(&itemName)
			if err != nil {
				fmt.Println("Error scanning input:", err)
				return list, err
			}
			fmt.Print("Current quantity: ")
			fmt.Println(list.Data[itemName].Value())
			fmt.Print("Increment or decrement? (i/d): ")
			var answer string
			_, err2 := fmt.Scanln(&answer)
			if err2 != nil {
				fmt.Println("Error scanning input:", err2)
				return list, err2
			}
			if answer == "i" {
				fmt.Print("Enter quantity to increment by: ")
				var itemQuantity int
				_, err3 := fmt.Scanln(&itemQuantity)
				if err3 != nil {
					fmt.Println("Error scanning input:", err3)
					return list, err3
				}
				for j := 0; j < itemQuantity; j++ {
					list.Increment(itemName)
				}
			}
			if answer == "d" {
				fmt.Print("Enter quantity to decrement by: ")
				var itemQuantity int
				_, err3 := fmt.Scanln(&itemQuantity)
				if err3 != nil {
					fmt.Println("Error scanning input:", err3)
					return list, err3
				}
				for j := 0; j < itemQuantity; j++ {
					list.Decrement(itemName)
				}
			}
		case 4:
			return list, nil
		default:
			fmt.Println("Invalid choice")
		}
	}
}

func (c *Client) menu() {
	fmt.Println("")
	fmt.Println("Welcome to the shopping list app, " + c.email)
	fmt.Println("1. Make shopping list")
	fmt.Println("2. Edit existing shopping list")
	fmt.Println("3. Show shopping list from file")
	fmt.Println("4. Push shopping list")
	fmt.Println("5. Pull shopping list")
	fmt.Println("6. Exit")
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
		fmt.Print("Enter the email of the list you want to edit: ")
		var email string
		_, err := fmt.Scanln(&email)
		if err != nil {
			fmt.Println("Error scanning input:", err)
			return
		}
		fmt.Println("Editing shopping list for", email+"...")
		list := crdt.LoadFromFile(email, c.email)
		newList, err := c.editList(list)
		if err != nil {
			fmt.Println("Error editing list:", err)
			return
		}
		newList.SaveToFile(email, c.email)
	case 3:
		fmt.Println("")
		fmt.Print("Enter the email of the list you want to show: ")
		var email string
		_, err := fmt.Scanln(&email)
		if err != nil {
			fmt.Println("Error scanning input:", err)
			return
		}
		fmt.Println("Showing shopping list for", email+"...")
		list := crdt.LoadFromFile(email, c.email)
		for key, value := range list.Data {
			fmt.Println(key, value.Value())
		}
	case 4:
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
		break
	case 5:
		fmt.Println("")
		fmt.Print("Enter the email of the list you want to pull: ")
		var email string
		_, err := fmt.Scanln(&email)
		if err != nil {
			fmt.Println("Error scanning input:", err)
			return
		}
		fmt.Println("Pulling shopping list for", email+"...")
		c.pull(email, 3, time.Second*2)
		break
	case 6:
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
	//create client dir inside list_storage folder if it doesn't exist
	if _, err := os.Stat("../list_storage/" + email); os.IsNotExist(err) {
		err := os.Mkdir("../list_storage/"+email, 0755)
		if err != nil {
			fmt.Println("Error creating client directory:", err)
			return
		}
	}
	// menu loop
	for {
		client.menu()
	}
}
