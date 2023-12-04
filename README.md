# Cloud Shopping List - Distributed Systems Project

## SyncShopSquad

### Overview

This project, implemented by the SyncShopSquad team, focuses on building a distributed system for a cloud-based shopping list application. 

### Components

#### 1. Consistent Hashing Implementation (`consistent.go`)

- Defines a `Ring` structure for consistent hashing.
- Contains a `Node` structure representing a server/node in the ring.
- Provides methods for adding nodes to the ring, getting a node for a given key, putting an item in the ring, and printing the nodes.
- Implements virtual nodes to improve load balancing.
- Data is replicated on the next two nodes in the ring to improve fault tolerance.

#### 2. Load Balancer (`load_balancer.go`)

- Implements a basic load balancer using the consistent hashing ring from `consistent.go`.
- Defines a `LoadBalancer` structure that contains an instance of the `Ring`.
- Provides methods for adding nodes to the ring and handling HTTP connections for both nodes and shopping list operations.
- Starts an HTTP server for the load balancer on port 8080.

#### 3. Server (`server.go`)

- Represents a simple server that connects to the load balancer.
- Generates a random node UUID and connects to the load balancer with retries.
- Handles incoming HTTP messages, specifically for shopping list operations.
- Stores the shopping list on its own database.

### Running the System

Before running the system, make sure you have Go installed on your machine. You can download Go [here](https://golang.org/dl/).
Change your working directory to the `src` folder.

1. **Start the Load Balancer:**
    - Execute `go run load_balancer.go` to start the load balancer on port 8080.

2. **Start Servers:**
    - Execute `go run server.go <port> <name>` to start a server on the specified port with the specified name.

3. **Connect Servers to Load Balancer:**
    - Servers automatically connect to the load balancer with retries.
4. **Start Client:**
    - Execute `go run client.go` to start the client.

