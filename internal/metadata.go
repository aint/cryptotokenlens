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