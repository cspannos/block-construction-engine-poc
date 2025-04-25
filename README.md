## Block Construction Engine PoC

This PoC simulates a Proof-of-Liquidity MEV-aware block builder that:

- Selects high-profit transactions based on gas, MEV, and PoL incentives
- Handles conflicting txs
- Demonstrates dynamic block packing

At the heart of this block builder is a max-heap based priority queue, sorted by the profit score. This allows the engine to efficiently select the highest-value transactions first, in this case aligned with proof-of-liquidity incentives:
`Profit(tx) = GasPrice * GasLimit + MEVBonus + PoLBonus`

To run:

```bash
go run main.go

