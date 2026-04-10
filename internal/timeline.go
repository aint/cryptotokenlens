package internal

import (
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"
	"maps"
	"slices"

	"github.com/aint/cryptotokenlens/internal/polygonscan"
)

func TxTimeBounds(txs []polygonscan.TokenTransfer) (int64, int64) {
	var first, last int64
	for _, t := range txs {
		ts, err := strconv.ParseInt(t.TimeStamp, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse timeStamp %q: %v\n", t.TimeStamp, err)
			continue
		}
		if first == 0 || ts < first {
			first = ts
		}
		if last == 0 || ts > last {
			last = ts
		}
	}
	return first, last
}

// TODO: switch to weeks/months if number of days > 90
func PrintTimeline(txs []polygonscan.TokenTransfer, tokenAddr string, decimals uint8) {
	timeline := buildTimeline(txs, tokenAddr)
	fmt.Printf("\nTimeline\n")
	fmt.Printf("%-12s %12s\n", "day", "value")
	for _, r := range timeline {
		fmt.Printf("%-12s %12s\n", r.Day, FormatBigInt(r.Value, decimals))
	}
}

func buildTimeline(txs []polygonscan.TokenTransfer, tokenAddr string) []timelineRow {
	timelineMap := make(stringBigIntMap)
	for _, t := range txs {
		ts, err := strconv.ParseInt(t.TimeStamp, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse timeStamp %q: %v\n", t.TimeStamp, err)
			continue
		}
		dayStr := time.Unix(ts, 0).UTC().Format("2006-01-02")

		v, ok := new(big.Int).SetString(t.Value, 10)
		if !ok {
			fmt.Fprintf(os.Stderr, "parse value %q: %v\n", t.Value, err)
			continue
		}

		if t.From == tokenAddr {
			cur := timelineMap.get(dayStr)
			timelineMap[dayStr]= new(big.Int).Add(cur, v)
		}
	}

	timeline := make([]timelineRow, 0, len(timelineMap))
	keys := slices.Collect(maps.Keys(timelineMap))
	slices.Sort(keys)
	for _, k := range keys {
		timeline = append(timeline, timelineRow{Day: k, Value: timelineMap.get(k)})
	}
	return timeline
}

type timelineRow struct {
	Day   string
	Value *big.Int
}