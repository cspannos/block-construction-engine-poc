package main

import (
	"container/heap"
	"fmt"
	"math/rand"
	"time"
)

// Transaction represents a simplified transaction
type Transaction struct {
	ID          string
	GasPrice    int64
	GasLimit    int64
	MEVBonus    int64
	PoLBonus    int64
	Nonce       int
	ConflictsWith []string // list of tx IDs this conflicts with
}

// Profit calculates the total profit from the tx
func (tx *Transaction) Profit() int64 {
	return tx.GasPrice*tx.GasLimit + tx.MEVBonus + tx.PoLBonus
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
	p.AllTxs[tx.ID] = tx
	heap.Push(&p.Heap, tx)
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
		usedIDs[tx.ID] = true
		selected = append(selected, tx)
	}

	return selected
}

func main() {
	rand.Seed(time.Now().UnixNano())
	pool := NewTxPool()

	// Mock some transactions
	for i := 0; i < 20; i++ {
		tx := &Transaction{
			ID:       fmt.Sprintf("tx%d", i),
			GasPrice: rand.Int63n(100),
			GasLimit: 21000 + rand.Int63n(80000),
			MEVBonus: rand.Int63n(10000),
			PoLBonus: rand.Int63n(5000),
			Nonce:    i,
		}
		pool.AddTx(tx)
	}

	blockGasLimit := int64(1000000)
	selectedTxs := pool.SelectTopTransactions(blockGasLimit)

	fmt.Printf("\nSelected Transactions for Block (Gas Limit: %d):\n", blockGasLimit)
	totalProfit := int64(0)
	for _, tx := range selectedTxs {
		txProfit := tx.Profit()
		totalProfit += txProfit
		fmt.Printf(" - %s | Profit: %d | Gas: %d\n", tx.ID, txProfit, tx.GasLimit)
	}
	fmt.Printf("\nTotal Profit: %d\n", totalProfit)
}

