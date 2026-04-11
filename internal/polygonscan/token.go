package polygonscan

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
)

func (c *Client) GetTotalSupply(addr string) (*big.Int, error) {
	q := url.Values{}
	q.Set("module", "stats")
	q.Set("action", "tokensupply")
	q.Set("contractaddress", addr)

	raw, err := c.get(q)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  string `json:"result"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if resp.Status != "1" {
		return nil, fmt.Errorf("api status=%s, message=%s, detail=%s", resp.Status, resp.Message, resp.Result)
	}
	n := big.NewInt(0)
	if _, ok := n.SetString(resp.Result, 10); !ok {
		return nil, fmt.Errorf("parse supply %s", resp.Result)
	}
	return n, nil
}
