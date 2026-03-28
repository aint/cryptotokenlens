package polygonscan

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"strings"
)

// ERC20MetaFromExplorer returns decimals and total supply (raw/smallest units) via the explorer API.
func (c *Client) ERC20MetaFromExplorer(contract string) (decimals uint8, supply *big.Int, err error) {
	contract = strings.ToLower(strings.TrimSpace(contract))
	supply, err = c.TokenSupplyRaw(contract)
	if err != nil {
		return 0, nil, fmt.Errorf("tokensupply: %w", err)
	}
	decimals, err = c.decimalsFromFirstTokenTx(contract)
	if err != nil {
		return 0, nil, fmt.Errorf("decimals: %w", err)
	}
	return decimals, supply, nil
}

// TokenSupplyRaw calls stats/tokensupply (raw integer in smallest units, same as on-chain totalSupply).
func (c *Client) TokenSupplyRaw(contract string) (*big.Int, error) {
	q := url.Values{}
	q.Set("module", "stats")
	q.Set("action", "tokensupply")
	q.Set("contractaddress", strings.ToLower(strings.TrimSpace(contract)))

	raw, err := c.get(q)
	if err != nil {
		return nil, err
	}
	env, err := parseEnvelope(raw)
	if err != nil {
		return nil, err
	}
	if env.Status != "1" {
		return nil, apiError(env)
	}
	var s string
	if err := json.Unmarshal(env.Result, &s); err != nil {
		return nil, fmt.Errorf("tokensupply result: %w", err)
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("tokensupply: empty result")
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("tokensupply: parse %q", s)
	}
	return n, nil
}

func (c *Client) decimalsFromFirstTokenTx(contract string) (uint8, error) {
	rows, err := c.tokenTxPage(contract, 1, 1, "asc")
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, fmt.Errorf("no tokentx to infer decimals")
	}
	s := strings.TrimSpace(rows[0].TokenDecimal)
	if s == "" {
		return 0, fmt.Errorf("tokenDecimal missing on tokentx row")
	}
	n, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("parse tokenDecimal %q: %w", s, err)
	}
	return uint8(n), nil
}

type envelope struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

func parseEnvelope(raw []byte) (envelope, error) {
	var e envelope
	if err := json.Unmarshal(raw, &e); err != nil {
		return e, fmt.Errorf("decode envelope: %w", err)
	}
	return e, nil
}

func apiError(e envelope) error {
	var msg string
	_ = json.Unmarshal(e.Result, &msg)
	if msg != "" {
		return fmt.Errorf("api status=%s message=%s detail=%s", e.Status, e.Message, msg)
	}
	return fmt.Errorf("api status=%s message=%s", e.Status, e.Message)
}
