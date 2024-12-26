package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

// Block represents a single block in the blockchain
type Block struct {
	Index        int
	Timestamp    string
	Transactions []Transaction
	Proof        int
	PreviousHash string
}

// Transaction represents a blockchain transaction
type Transaction struct {
	Sender     string
	Recipient  string
	Amount     int
	DomainName string
}

// Blockchain represents the blockchain and peers
type Blockchain struct {
	Chain               []Block
	CurrentTransactions []Transaction
	Peers               []string // Stores peer addresses
	Mutex               sync.Mutex
}

// CreateBlock generates a new block and appends it to the chain
func (bc *Blockchain) CreateBlock(proof int, previousHash string) Block {
	bc.Mutex.Lock()
	defer bc.Mutex.Unlock()
	block := Block{
		Index:        len(bc.Chain) + 1,
		Timestamp:    time.Now().String(),
		Transactions: bc.CurrentTransactions,
		Proof:        proof,
		PreviousHash: previousHash,
	}
	bc.CurrentTransactions = []Transaction{}
	bc.Chain = append(bc.Chain, block)
	return block
}

// AddTransaction adds a transaction to the blockchain
func (bc *Blockchain) AddTransaction(sender, recipient string, amount int, domain string) {
	bc.Mutex.Lock()
	defer bc.Mutex.Unlock()
	bc.CurrentTransactions = append(bc.CurrentTransactions, Transaction{sender, recipient, amount, domain})
}

// AddPeer adds a new peer to the peer list
func (bc *Blockchain) AddPeer(address string) {
	bc.Mutex.Lock()
	defer bc.Mutex.Unlock()
	for _, peer := range bc.Peers {
		if peer == address {
			fmt.Println("Peer already exists:", address)
			return
		}
	}
	bc.Peers = append(bc.Peers, address)
	fmt.Println("New peer added:", address)
}

// ProofOfWork performs the mining proof of work
func (bc *Blockchain) ProofOfWork(lastProof int) int {
	proof := 0
	for !isValidProof(lastProof, proof) {
		proof++
	}
	return proof
}

// isValidProof validates the proof of work
func isValidProof(lastProof, proof int) bool {
	guess := fmt.Sprintf("%d%d", lastProof, proof)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(guess)))
	return hash[:4] == "0000"
}

// SyncWithPeers syncs blockchain data with peers
func (bc *Blockchain) SyncWithPeers() {
	for _, peer := range bc.Peers {
		conn, err := net.Dial("tcp", peer)
		if err != nil {
			fmt.Println("Failed to connect to peer:", peer)
			continue
		}
		defer conn.Close()
		fmt.Fprintf(conn, "{\"action\": \"sync\"}")
	}
}

// handleConnection handles incoming peer and node connections
func handleConnection(conn net.Conn, bc *Blockchain) {
	defer conn.Close()
	var message map[string]interface{}
	json.NewDecoder(conn).Decode(&message)

	switch message["action"] {
	case "mine":
		lastProof := bc.Chain[len(bc.Chain)-1].Proof
		proof := bc.ProofOfWork(lastProof)
		previousHash := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%v", bc.Chain[len(bc.Chain)-1]))))
		bc.CreateBlock(proof, previousHash)
		conn.Write([]byte("Block Mined Successfully"))

	case "add_peer":
		address, ok := message["address"].(string)
		if ok {
			bc.AddPeer(address)
			conn.Write([]byte("Peer Added Successfully"))
		}

	case "sync":
		chainJSON, _ := json.Marshal(bc.Chain)
		conn.Write(chainJSON)

	default:
		conn.Write([]byte("Unknown action"))
	}
}

// startNode starts the node server
func startNode(port string, bc *Blockchain) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error starting node:", err)
		return
	}
	defer ln.Close()
	fmt.Println("Node started on port", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}
		go handleConnection(conn, bc)
	}
}

// main starts the blockchain node
func main() {
	bc := &Blockchain{}
	bc.CreateBlock(1, "0")

	// Example: Adding initial peers (Optional)
	bc.AddPeer("127.0.0.1:8081") // Add a local peer

	go startNode("8080", bc) // Start the node on port 8080

	// Sync with peers periodically (Optional)
	go func() {
		for {
			time.Sleep(30 * time.Second)
			bc.SyncWithPeers()
		}
	}()

	select {}
}

