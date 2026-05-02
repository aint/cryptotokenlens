package internal

import (
	"fmt"
	"math/big"
	"strconv"
	"time"
)

type DailyPoint struct {
	Day        time.Time
	Value      *big.Int
	CumValue   *big.Int
	CumPercent float64
}

func DailySeries(token Token) ([]DailyPoint, error) {
	var start, end time.Time
	timelineMap := make(map[time.Time]*big.Int)
	for _, t := range token.Txs {
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

		if t.From == token.Address {
			cur := timelineMap[day]
			if cur == nil {
				cur = big.NewInt(0)
			}
			timelineMap[day]= new(big.Int).Add(cur, value)
		}
	}

	cumValue := big.NewInt(0)
	dailySeries := make([]DailyPoint, 0, len(timelineMap))
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		value, ok := timelineMap[d]
		if !ok {
			value = big.NewInt(0)
		}
		cumValue = new(big.Int).Add(cumValue, value)
		pct, _ := new(big.Rat).Mul(
			new(big.Rat).SetFrac(cumValue, token.TotalSupplyRaw),
			big.NewRat(100, 1),
		).Float64()
		dailySeries = append(dailySeries, DailyPoint{Day: d, Value: value, CumValue: cumValue, CumPercent: pct})
		if cumValue.Cmp(token.TotalSupplyRaw) == 0 {
			break
		}
	}

	return dailySeries, nil
}
