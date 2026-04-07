package internal

import (
	"fmt"
	"maps"
	"math/big"
	"slices"

	"github.com/aint/cryptotokenlens/internal/polygonscan"
)

const zeroAddr0x = "0x0000000000000000000000000000000000000000"

func PrintHolders(txs []polygonscan.TokenTransfer, totalSupply *big.Int, decimals uint8, top int) error {
	balances, err := GetBalances(txs)
	if err != nil {
		return err
	}

	balances = balances.filterOutZero()
	keys := slices.Collect(maps.Keys(balances))
	slices.SortFunc(keys, func(a, b string) int {
		return balances.get(b).Cmp(balances.get(a)) // descending by balance
	})

	idx := min(len(keys), top)

	fmt.Printf("\nHolders: showing %d of %d\n", len(keys[:idx]), len(balances))
	fmt.Printf("%4s %-44s %32s %14s\n", "#", "address", "balance", "% of supply")
	for i, k := range keys[:idx] {
		b := balances.get(k)
		fmt.Printf("%d. %s %32s %13s%%\n", i+1, k, FormatBigInt(b, decimals), PercentOf(b, totalSupply))
	}

	return nil
}

func GetBalances(txs []polygonscan.TokenTransfer) (balanceMap, error) {
	balances := make(balanceMap)
	for _, tx := range txs {
		v, ok := new(big.Int).SetString(tx.Value, 10)
		if !ok {
			return nil, fmt.Errorf("parse value %q", tx.Value)
		}
		if tx.From != zeroAddr0x {
			cur := balances.get(tx.From)
			balances[tx.From] = new(big.Int).Sub(cur, v)
		}
		if tx.To != zeroAddr0x {
			cur := balances.get(tx.To)
			balances[tx.To] = new(big.Int).Add(cur, v)
		}
	}

	return balances, nil
}

type balanceMap map[string]*big.Int

func (m balanceMap) filterOutZero() balanceMap {
	filtered := make(balanceMap)
	for k, v := range m {
		if v.Sign() > 0 {
			filtered[k] = v
		}
	}
	return filtered
}

func (m balanceMap) get(key string) *big.Int {
	v := m[key]
	if v == nil {
		v = big.NewInt(0)
	}
	return v
}
