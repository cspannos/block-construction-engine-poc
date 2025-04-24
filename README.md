## Block Construction Engine PoC

This PoC simulates a Proof-of-Liquidity MEV-aware block builder that:

- Selects high-profit transactions based on gas, MEV, and PoL incentives
- Handles conflicting txs
- Demonstrates dynamic block packing

To run:

```bash
go run main.go

