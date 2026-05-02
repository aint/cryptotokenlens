package internal

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"
)

//go:embed timeseries_chart.html
var timeseriesChartHTML []byte

// chartDataPlaceholder must match timeseries_chart.html exactly.
var chartDataPlaceholder = []byte("__CHART_DATA_JSON__")

func WriteDailySeriesHTML(path string, token Token, series []DailyPoint, etas []ETA) {
	payload := buildChartPayload(series, etas, token)

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

func buildChartPayload(series []DailyPoint, etas []ETA, token Token) chartPayload {
	payload := chartPayload{
		Labels:     make([]string, 0, len(series)),
		Daily:      make([]float64, 0, len(series)),
		Cumulative: make([]float64, 0, len(series)),
		Title:      fmt.Sprintf("Daily buys — %s", token.Name),
		ETAs:       make([]chartETA, 0, len(etas)),
	}
	for _, p := range series {
		payload.Labels = append(payload.Labels, p.Day.UTC().Format(timeDateOnly))
		payload.Daily = append(payload.Daily, rawToHumanFloat(p.Value, token.Decimal))
		payload.Cumulative = append(payload.Cumulative, rawToHumanFloat(p.CumValue, token.Decimal))
	}
	for _, e := range etas {
		payload.ETAs = append(payload.ETAs, chartETA{
			Window: e.Window,
			Rate:   e.Rate,
			Days:   e.Days,
			Date:   e.Time.UTC().Format(time.DateOnly),
		})
	}
	return payload
}

type chartPayload struct {
	Labels      []string   `json:"labels"`
	Daily       []float64  `json:"daily"`
	Cumulative  []float64  `json:"cumulative"`
	Title       string     `json:"title"`
	ETAs        []chartETA `json:"etas"`
}

type chartETA struct {
	Window string `json:"window"`
	Rate   string `json:"rate"`
	Days   int64  `json:"days"`
	Date   string `json:"date"`
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