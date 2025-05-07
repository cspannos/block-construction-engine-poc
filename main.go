package main

import (
	"bytes"
	"container/heap"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Transaction represents a Berachain transaction
type Transaction struct {
	Hash          string   `json:"hash"`
	GasPrice      int64    `json:"gasPrice"`
	GasLimit      int64    `json:"gasLimit"`
	MEVBonus      int64    `json:"mevBonus"`
	PoLBonus      int64    `json:"polBonus"`
	Nonce         int      `json:"nonce"`
	ConflictsWith []string `json:"conflictsWith"`
}

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Result  []Transaction `json:"result"`
	Error   *RPCError     `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// TxHeap implements a max-heap for Transactions based on Profit
type TxHeap []*Transaction

func (h TxHeap) Len() int           { return len(h) }
func (h TxHeap) Less(i, j int) bool { return h[i].Profit() > h[j].Profit() } // max-heap
func (h TxHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *TxHeap) Push(x any) {
	*h = append(*h, x.(*Transaction))
}

func (h *TxHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// TxPool mocks a transaction pool
type TxPool struct {
	AllTxs map[string]*Transaction
	Heap   TxHeap
}

func NewTxPool() *TxPool {
	return &TxPool{
		AllTxs: make(map[string]*Transaction),
		Heap:   TxHeap{},
	}
}

func (p *TxPool) AddTx(tx *Transaction) {
	p.AllTxs[tx.Hash] = tx
	heap.Push(&p.Heap, tx)
}

// Profit calculates the total profit from the tx
func (tx *Transaction) Profit() int64 {
	return tx.GasPrice*tx.GasLimit + tx.MEVBonus + tx.PoLBonus
}

// FetchTransactions fetches pending transactions from Berachain RPC
func (p *TxPool) FetchTransactions() error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Get pending transactions from the mempool
	blockReq := RPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{"pending", true}, // "pending" to get mempool transactions
		ID:      1,
	}

	jsonData, err := json.Marshal(blockReq)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://rpc.berachain.com", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	var blockResp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			Transactions []struct {
				Hash     string `json:"hash"`
				GasPrice string `json:"gasPrice"`
				Gas      string `json:"gas"`
				Nonce    string `json:"nonce"`
			} `json:"transactions"`
		} `json:"result"`
		Error *RPCError `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &blockResp); err != nil {
		return fmt.Errorf("error unmarshaling response: %v", err)
	}

	if blockResp.Error != nil {
		return fmt.Errorf("RPC error: %s", blockResp.Error.Message)
	}

	// Convert hex values to integers and create transactions
	for _, tx := range blockResp.Result.Transactions {
		gasPrice, _ := strconv.ParseInt(tx.GasPrice[2:], 16, 64)
		gasLimit, _ := strconv.ParseInt(tx.Gas[2:], 16, 64)
		nonce, _ := strconv.ParseInt(tx.Nonce[2:], 16, 64)

		transaction := &Transaction{
			Hash:          tx.Hash,
			GasPrice:      gasPrice,
			GasLimit:      gasLimit,
			Nonce:         int(nonce),
			MEVBonus:      0, // This would need to be calculated or fetched from another source
			PoLBonus:      0, // Same as above
			ConflictsWith: []string{},
		}
		p.AddTx(transaction)
	}

	return nil
}

func (p *TxPool) SelectTopTransactions(gasLimit int64) []*Transaction {
	heap.Init(&p.Heap)
	selected := []*Transaction{}
	usedGas := int64(0)
	usedIDs := map[string]bool{}

	for p.Heap.Len() > 0 && usedGas < gasLimit {
		tx := heap.Pop(&p.Heap).(*Transaction)
		conflict := false
		for _, id := range tx.ConflictsWith {
			if usedIDs[id] {
				conflict = true
				break
			}
		}
		if conflict {
			continue
		}
		if usedGas+tx.GasLimit > gasLimit {
			continue
		}
		usedGas += tx.GasLimit
		usedIDs[tx.Hash] = true
		selected = append(selected, tx)
	}

	return selected
}

// FormatWei converts wei to a human-readable string
func FormatWei(wei int64) string {
	// Convert to float for division
	bera := float64(wei) / 1e18
	return fmt.Sprintf("%.6f BERA", bera)
}

func main() {
	pool := NewTxPool()

	// Fetch transactions from Berachain RPC
	if err := pool.FetchTransactions(); err != nil {
		fmt.Printf("Error fetching transactions: %v\n", err)
		return
	}

	blockGasLimit := int64(30000000) // https://docs.berachain.com/learn/help/faqs#what-do-berachain-s-performance-metrics-look-like
	selectedTxs := pool.SelectTopTransactions(blockGasLimit)

	fmt.Printf("\nSelected Transactions for Block (Gas Limit: %d):\n", blockGasLimit)
	totalProfit := int64(0)
	for _, tx := range selectedTxs {
		txProfit := tx.Profit()
		totalProfit += txProfit
		fmt.Printf(" - %s | Profit: %s | Gas: %d\n", tx.Hash, FormatWei(txProfit), tx.GasLimit)
	}
	fmt.Printf("\nTotal Profit: %s\n", FormatWei(totalProfit))
}
