package internal

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/aint/cryptotokenlens/internal/polygonscan"
)

func BoughtAmount(txs []polygonscan.TokenTransfer, tokenAddr string) *big.Int {
	boughtAmount := big.NewInt(0)
	for _, t := range txs {
		v, ok := new(big.Int).SetString(t.Value, 10)
		if !ok {
			fmt.Fprintf(os.Stderr, "parse value %q\n", t.Value)
			continue
		}

		from := strings.ToLower(t.From)
		if from == tokenAddr {
			boughtAmount.Add(boughtAmount, v)
		}
	}

	return boughtAmount
}

func GetDecimal(txs []polygonscan.TokenTransfer) (uint8, error) {
	if len(txs) == 0 {
		return 0, errors.New("no transactions found")
	}
	decimalStr := strings.TrimSpace(txs[0].TokenDecimal)
	if decimalStr == "" {
		return 0, errors.New("decimal missing")
	}
	decimal, err := strconv.ParseUint(decimalStr, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("parse decimal %q: %w", decimalStr, err)
	}
	return uint8(decimal), nil
}