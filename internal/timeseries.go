package internal

import (
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/aint/cryptotokenlens/internal/polygonscan"
)

func PrintDailySeries(txs []polygonscan.TokenTransfer, tokenAddr string, totalSupply *big.Int, decimals uint8) {
	timeline, err := buildDailySeries(txs, tokenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build timeline: %v\n", err)
		return
	}
	fmt.Printf("\nTimeline\n")
	fmt.Printf("%-12s %12s %12s\n", "day", "Δ", "% of supply")
	cum := big.NewInt(0)
	for _, r := range timeline {
		if r.Value.Sign() == 0 {
			continue
		}
		cum.Add(cum, r.Value)
		fmt.Printf("%-12s %12s %12s%%\n", r.Day.Format("2006-01-02"), FormatBigInt(r.Value, decimals), PercentOf(cum, totalSupply))
	}
}

func buildDailySeries(txs []polygonscan.TokenTransfer, tokenAddr string) ([]dailyPoint, error) {
	var start, end time.Time
	timelineMap := make(map[time.Time]*big.Int)
	for _, t := range txs {
		ts, err := strconv.ParseInt(t.TimeStamp, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp %q: %w", t.TimeStamp, err)
		}
		day := time.Unix(ts, 0).UTC().Truncate(24 * time.Hour)
		if start.IsZero() || day.Before(start) {
			start = day
		}
		if day.After(end) {
			end = day
		}

		value, ok := new(big.Int).SetString(t.Value, 10)
		if !ok {
			return nil, fmt.Errorf("parse value %q: %w", t.Value, err)
		}

		if t.From == tokenAddr {
			cur := timelineMap[day]
			if cur == nil {
				cur = big.NewInt(0)
			}
			timelineMap[day]= new(big.Int).Add(cur, value)
		}
	}

	dailySeries := make([]dailyPoint, 0, len(timelineMap))
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		value, ok := timelineMap[d]
		if !ok {
			value = big.NewInt(0)
		}
		dailySeries = append(dailySeries, dailyPoint{Day: d, Value: value})
	}

	return dailySeries, nil
}

type dailyPoint struct {
	Day   time.Time
	Value *big.Int
}