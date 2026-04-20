package internal

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/aint/cryptotokenlens/internal/polygonscan"
)

//go:embed timeseries_chart.html
var timeseriesChartHTML []byte

// chartDataPlaceholder must match timeseries_chart.html exactly.
var chartDataPlaceholder = []byte("__CHART_DATA_JSON__")

// WriteDailySeriesHTML writes a single HTML file with embedded Chart.js (CDN) and
// daily + cumulative series from buildDailySeries (human token units per decimals).
func WriteDailySeriesHTML(path string, txs []polygonscan.TokenTransfer, tokenAddr string, decimals uint8) {
	series, err := buildDailySeries(txs, tokenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build timeline: %v\n", err)
		return
	}

	payload := chartPayload{
		Labels:     make([]string, 0, len(series)),
		Daily:      make([]float64, 0, len(series)),
		Cumulative: make([]float64, 0, len(series)),
		Title:      fmt.Sprintf("Daily buys (from token) — %s", tokenAddr),
	}
	cum := big.NewInt(0)
	for _, p := range series {
		payload.Labels = append(payload.Labels, p.Day.UTC().Format(timeDateOnly))
		payload.Daily = append(payload.Daily, rawToHumanFloat(p.Value, decimals))
		cum = new(big.Int).Add(cum, p.Value)
		payload.Cumulative = append(payload.Cumulative, rawToHumanFloat(cum, decimals))
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal chart data: %v\n", err)
		return
	}

	if !bytes.Contains(timeseriesChartHTML, chartDataPlaceholder) {
		fmt.Fprintf(os.Stderr, "timeseries chart template missing %s\n", string(chartDataPlaceholder))
		return
	}
	out := bytes.ReplaceAll(timeseriesChartHTML, chartDataPlaceholder, jsonBytes)
	err = os.WriteFile(path, out, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "write file: %v\n", err)
		return
	}
	fmt.Printf("wrote daily series HTML: %s\n", path)
}

type chartPayload struct {
	Labels      []string  `json:"labels"`
	Daily       []float64 `json:"daily"`
	Cumulative  []float64 `json:"cumulative"`
	Title       string    `json:"title"`
}

func rawToHumanFloat(raw *big.Int, decimals uint8) float64 {
	if raw == nil || raw.Sign() == 0 {
		return 0
	}
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	r := new(big.Rat).SetFrac(new(big.Int).Set(raw), denom)
	f, _ := r.Float64()
	return f
}