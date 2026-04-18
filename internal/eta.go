package internal

import (
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/aint/cryptotokenlens/internal/polygonscan"
)

// PrintETA prints up to three point estimates (7-day, 30-day, lifetime trailing average of daily Δ).
func PrintETA(txs []polygonscan.TokenTransfer, tokenAddr string, decimals uint8, totalSupply, boughtAmount *big.Int) {
	if boughtAmount == nil {
		boughtAmount = big.NewInt(0)
	}
	remaining := new(big.Int).Sub(totalSupply, boughtAmount)
	if remaining.Sign() <= 0 {
		fmt.Println("ETA: n/a (cumulative bought already ≥ totalSupply)")
		return
	}

	dailySeries, err := buildDailySeries(txs, tokenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build timeline: %v\n", err)
		return
	}
	if len(dailySeries) < 7 {
		fmt.Println("ETA: n/a (not enough data to calculate ETA)")
		return
	}

	type trailingWindow struct {
		name string
		days int
	}
	windows := []trailingWindow{
		{"last 7 UTC days", min(7, len(dailySeries))},
		{"last 30 UTC days", min(30, len(dailySeries))},
		{"full history (all calendar days)", len(dailySeries)},
	}

	fmt.Println("ETA extrapolation (constant rate after last data day; each row uses trailing w-day mean of daily Δ):")

	for _, w := range windows {
		eta, days, rate, err := etaFromTrailingWindow(dailySeries, remaining, w.days, decimals)
		if err != nil {
			fmt.Printf("  %s (w=%d): n/a — %s\n", w.name, w.days, err.Error())
			continue
		}
		fmt.Printf("  %s (w=%d): avg Δ≈%s tok/day → ~%d calendar days → %s\n", w.name, w.days, rate, days, eta.Format(time.RFC3339))
	}
}

func etaFromTrailingWindow(dailySeries []dailyPoint, remaining *big.Int, w int, decimals uint8) (time.Time, int64, string, error) {
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
	rate := FormatBigInt(new(big.Int).Quo(sum, big.NewInt(int64(w))), decimals)
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
