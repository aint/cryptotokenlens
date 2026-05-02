package internal

import (
	"fmt"
	"math/big"
	"time"
)

var trailingWindows = map[string]int{
	"last 7 UTC days": 7,
	"last 30 UTC days": 30,
	"full history (all calendar days)": -1,
}

// MovingAverageETA returns up to three point estimates (7-day, 30-day, lifetime trailing average of daily Δ).
func MovingAverageETA(dailySeries []DailyPoint, token Token) ([]ETA, error) {
	if token.BoughtRaw == nil {
		token.BoughtRaw = big.NewInt(0)
	}
	remaining := new(big.Int).Sub(token.TotalSupplyRaw, token.BoughtRaw)
	if remaining.Sign() <= 0 {
		fmt.Println("No ETA: cumulative bought already ≥ total supply")
		return nil, nil
	}

	if len(dailySeries) < 7 {
		return nil, fmt.Errorf("not enough data to calculate ETA")
	}

	// todo: map is random order
	trailingWindows = map[string]int{
		"last 7 UTC days": min(7, len(dailySeries)),
		"last 30 UTC days": min(30, len(dailySeries)),
		"full history (all calendar days)": len(dailySeries),
	}

	etas := make([]ETA, 0, len(trailingWindows))
	for name, days := range trailingWindows {
		eta, days, rate, err := etaFromTrailingWindow(dailySeries, remaining, days, token.Decimal)
		if err != nil {
			return nil, err
		}
		etas = append(etas, ETA{Time: eta, Days: days, Rate: rate, Window: name})
	}
	return etas, nil
}

type ETA struct {
	Time time.Time
	Days int64
	Rate string
	Window string
}

func etaFromTrailingWindow(dailySeries []DailyPoint, remaining *big.Int, w int, decimal uint8) (time.Time, int64, string, error) {
	sum := big.NewInt(0)
	from := len(dailySeries) - w
	for j := from; j < len(dailySeries); j++ {
		sum.Add(sum, dailySeries[j].Value)
	}
	// sum / w = avg daily Δ in the window
	avgRat := new(big.Rat).SetFrac(new(big.Int).Set(sum), big.NewInt(int64(w)))
	if avgRat.Sign() == 0 {
		return time.Time{}, 0, "", fmt.Errorf("average daily Δ is zero over last %d UTC days", w)
	}

	// daysRat = remaining / avgRat : If every future day looked like this average, how many days of sales to sell remaining tokens?”
	remRat := new(big.Rat).SetInt(remaining)
	daysRat := new(big.Rat).Quo(remRat, avgRat)
	if daysRat.Sign() < 0 {
		return time.Time{}, 0, "", fmt.Errorf("negative days (unexpected)")
	}

	daysInt := ceilRatToInt64(daysRat)
	if daysInt < 0 {
		return time.Time{}, 0, "", fmt.Errorf("day count out of int64 range")
	}

	lastDay := dailySeries[len(dailySeries)-1].Day
	eta := lastDay.AddDate(0, 0, int(daysInt))
	rate := FormatBigRat(avgRat, decimal, 1)
	return eta, daysInt, rate, nil
}

// ceilRatToInt64 returns ⌈x⌉ for x ≥ 0; for huge values beyond int64, returns -1.
func ceilRatToInt64(x *big.Rat) int64 {
	if x.Sign() <= 0 {
		return 0
	}
	num := new(big.Int).Set(x.Num())
	den := new(big.Int).Set(x.Denom())
	if den.Sign() == 0 {
		return -1
	}
	// ceil(num/den) = (num + den - 1) / den  for num ≥ 0, den > 0
	ceilNum := new(big.Int).Add(num, new(big.Int).Sub(den, big.NewInt(1)))
	q := new(big.Int).Quo(ceilNum, den)
	if !q.IsInt64() {
		return -1
	}
	return q.Int64()
}
