package internal

import (
	"fmt"
	"maps"
	"math/big"
	"slices"
)

const zeroAddr0x = "0x0000000000000000000000000000000000000000"

type Holder struct {
	Address string
	Balance *big.Int
}

type Holders []Holder

func (t Token) getHolders() (Holders, error) {
	holderMap := make(map[string]*big.Int)
	for _, tx := range t.Txs {
		v, ok := new(big.Int).SetString(tx.Value, 10)
		if !ok {
			return nil, fmt.Errorf("parse value %q", tx.Value)
		}
		if tx.From != zeroAddr0x {
			cur := holderMap[tx.From]
			if cur == nil {
				cur = big.NewInt(0)
			}
			holderMap[tx.From] = new(big.Int).Sub(cur, v)
		}
		if tx.To != zeroAddr0x {
			cur := holderMap[tx.To]
			if cur == nil {
				cur = big.NewInt(0)
			}
			holderMap[tx.To] = new(big.Int).Add(cur, v)
		}
	}

	keys := slices.Collect(maps.Keys(holderMap))
	slices.SortFunc(keys, func(a, b string) int {
		return holderMap[b].Cmp(holderMap[a]) // descending by balance
	})

	holders := make(Holders, 0, len(holderMap))
	for _, address := range keys {
		if address == t.Address {
			// ignore token's balance
			continue
		}
		holders = append(holders, Holder{Address: address, Balance: holderMap[address]})
	}
	return holders, nil
}

func (t Token) PrintHolders(top int) error {
	idx := min(len(t.Holders), top)

	fmt.Printf("\nHolders: showing %d of %d\n", len(t.Holders[:idx]), len(t.Holders))
	fmt.Printf("%4s %-44s %32s %14s\n", "#", "address", "balance", "% of supply")
	for i, h := range t.Holders[:idx] {
		if h.Balance.Sign() == 0 {
			continue
		}
		fmt.Printf("%d. %s %32s %13s%%\n", i+1, h.Address, FormatBigInt(h.Balance, t.Decimal), PercentOf(h.Balance, t.TotalSupplyRaw))
	}

	return nil
}
