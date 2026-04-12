package internal

import (
	"fmt"
	"math/big"
	"time"

	"github.com/aint/cryptotokenlens/internal/polygonscan"
)

const defaultMAWindowDays = 60

// PrintETA estimates when cumulative bought reaches totalSupply by extrapolating a
// trailing simple moving average of daily Δ over the last defaultMAWindowDays calendar days
// (dense series: missing days count as zero). Uses buildTimeline from timeline.go (same metric).
func PrintETA(txs []polygonscan.TokenTransfer, tokenAddr string, decimals uint8, totalSupply, boughtAmount *big.Int) {
	remaining := new(big.Int).Sub(totalSupply, boughtAmount)
	if remaining.Sign() <= 0 {
		return
	}

	timeline := buildTimeline(txs, tokenAddr)
	if len(timeline) == 0 {
		fmt.Println("no data to calculate ETA")
		return
	}

	start, _ := time.ParseInLocation("2006-01-02", timeline[0].Day, time.UTC)
	end, _ := time.ParseInLocation("2006-01-02", timeline[len(timeline)-1].Day, time.UTC)
	entries := denseDailyDeltas(timeline, start, end)

	w := min(defaultMAWindowDays, len(entries))

	sum := big.NewInt(0)
	from := len(entries) - w
	for j := from; j < len(entries); j++ {
		sum.Add(sum, entries[j].Value)
	}
	// sum / w = avg daily Δ in the window
	avgRat := new(big.Rat).SetFrac(new(big.Int).Set(sum), big.NewInt(int64(w)))
	if avgRat.Sign() == 0 {
		fmt.Printf("trailing %d-day average daily Δ is zero (cannot extrapolate)\n", w)
		return
	}

	// daysRat = remaining / avgRat : If every future day looked like this average, how many days of sales to sell remaining tokens?”
	remRat := new(big.Rat).SetInt(remaining)
	daysRat := new(big.Rat).Quo(remRat, avgRat)
	if daysRat.Sign() < 0 {
		fmt.Println("negative days to target (unexpected)")
		return
	}

	// Ceil positive rational to whole calendar days.
	daysInt := ceilRatToInt64(daysRat)
	if daysInt < 0 {
		fmt.Println("ETA day count out of int64 range")
		return
	}

	lastDay := entries[len(entries)-1].Day
	etaUTC := lastDay.AddDate(0, 0, int(daysInt))

	avgHuman := FormatBigInt(new(big.Int).Quo(sum, big.NewInt(int64(w))), decimals)
	fmt.Printf(
		"MA model: trailing %d UTC days, avg Δ≈%s tokens/day (decimals=%d), ~%d calendar days after last data day → %s",
		w, avgHuman, decimals, daysInt, etaUTC.Format(time.RFC3339),
	)
}

type etaEntry struct {
	Day   time.Time
	Value *big.Int // daily Δ (raw units) leaving token contract that UTC day
}

// denseDailyDeltas walks every UTC calendar day from start through end inclusive and assigns
// each day's Δ from the sorted timeline (or zero if no row for that day).
func denseDailyDeltas(timeline []timelineRow, start, end time.Time) []etaEntry {
	var out []etaEntry
	ti := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		ds := d.Format("2006-01-02")
		var v *big.Int
		if ti < len(timeline) && timeline[ti].Day == ds {
			v = new(big.Int).Set(timeline[ti].Value)
			ti++
		} else {
			v = big.NewInt(0)
		}
		out = append(out, etaEntry{Day: d, Value: v})
	}
	return out
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
