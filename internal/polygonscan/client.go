package polygonscan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultBaseURL is Etherscan API v2 (multichain); use chainid=137 for Polygon PoS.
const DefaultBaseURL = "https://api.etherscan.io/v2/api"

// PolygonChainID is Polygon PoS mainnet for Etherscan API v2.
const PolygonChainID = 137

// defaultMinRequestInterval spaces calls for free-tier caps (often ~3 calls/sec).
const defaultMinRequestInterval = 400 * time.Millisecond

// Client calls the Etherscan-compatible explorer API (v2: base + chainid + unified API key).
type Client struct {
	HTTP    *http.Client
	BaseURL string
	APIKey  string
	// ChainID selects the chain for v2 (default PolygonChainID when 0).
	ChainID int
	// MinRequestInterval is the minimum time between completed requests (0 = defaultMinRequestInterval).
	MinRequestInterval time.Duration

	mu            sync.Mutex
	lastRequestAt time.Time
}

func (c *Client) base() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return DefaultBaseURL
}

func (c *Client) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) chainID() int {
	if c.ChainID != 0 {
		return c.ChainID
	}
	return PolygonChainID
}

// TokenTransfer is one row from module=account&action=tokentx.
type TokenTransfer struct {
	BlockNumber     string `json:"blockNumber"`
	TimeStamp       string `json:"timeStamp"`
	Hash            string `json:"hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           string `json:"value"`
	TokenDecimal    string `json:"tokenDecimal"`
	ContractAddress string `json:"contractAddress"`
	TxreceiptStatus string `json:"txreceipt_status"`
}

// FetchAllTokenTx paginates tokentx until a page returns fewer than offset rows or maxPages reached (0 = unlimited).
func (c *Client) FetchAllTokenTx(contract string, sort string, offset, maxPages int, pause time.Duration) ([]TokenTransfer, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("polygonscan api key is empty")
	}
	if offset <= 0 {
		offset = 1000
	}
	if sort == "" {
		sort = "asc"
	}
	var all []TokenTransfer
	for page := 1; maxPages == 0 || page <= maxPages; page++ {
		batch, err := c.tokenTxPage(contract, page, offset, sort)
		if err != nil {
			return all, err
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		if len(batch) < offset {
			break
		}
		if pause > 0 {
			time.Sleep(pause)
		}
	}
	return all, nil
}

func (c *Client) tokenTxPage(contract string, page, offset int, sort string) ([]TokenTransfer, error) {
	q := url.Values{}
	q.Set("module", "account")
	q.Set("action", "tokentx")
	q.Set("contractaddress", contract)
	q.Set("page", strconv.Itoa(page))
	q.Set("offset", strconv.Itoa(offset))
	q.Set("sort", sort)

	raw, err := c.get(q)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Status  string          `json:"status"`
		Message string          `json:"message"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode envelope: %w", err)
	}
	if envelope.Status != "1" {
		// Some errors return result as string
		var msg string
		_ = json.Unmarshal(envelope.Result, &msg)
		if msg != "" {
			return nil, fmt.Errorf("polygonscan api: status=%s message=%s result=%s", envelope.Status, envelope.Message, msg)
		}
		return nil, fmt.Errorf("polygonscan api: status=%s message=%s", envelope.Status, envelope.Message)
	}

	var rows []TokenTransfer
	if err := json.Unmarshal(envelope.Result, &rows); err != nil {
		return nil, fmt.Errorf("decode tokentx rows: %w", err)
	}
	return rows, nil
}

func (c *Client) minInterval() time.Duration {
	if c.MinRequestInterval > 0 {
		return c.MinRequestInterval
	}
	return defaultMinRequestInterval
}

func (c *Client) throttle() {
	min := c.minInterval()
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.lastRequestAt.IsZero() {
		if elapsed := time.Since(c.lastRequestAt); elapsed < min {
			time.Sleep(min - elapsed)
		}
	}
}

func (c *Client) markRequestDone() {
	c.mu.Lock()
	c.lastRequestAt = time.Now()
	c.mu.Unlock()
}

func responseLooksRateLimited(body []byte) bool {
	return bytes.Contains(bytes.ToLower(body), []byte("rate limit"))
}

func (c *Client) get(q url.Values) ([]byte, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("api key is empty")
	}
	q.Set("apikey", strings.TrimSpace(c.APIKey))
	if q.Get("chainid") == "" {
		q.Set("chainid", strconv.Itoa(c.chainID()))
	}
	u, err := url.Parse(c.base())
	if err != nil {
		return nil, err
	}
	u.RawQuery = q.Encode()
	urlStr := u.String()

	var lastBody []byte
	for attempt := 0; attempt < 8; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * 350 * time.Millisecond
			if backoff > 4*time.Second {
				backoff = 4 * time.Second
			}
			time.Sleep(backoff)
		}

		c.throttle()

		req, err := http.NewRequest(http.MethodGet, urlStr, nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.client().Do(req)
		if err != nil {
			c.markRequestDone()
			return nil, err
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		resp.Body.Close()
		c.markRequestDone()
		if err != nil {
			return nil, err
		}
		lastBody = body

		if resp.StatusCode != http.StatusOK {
			if responseLooksRateLimited(body) && attempt < 7 {
				continue
			}
			return nil, fmt.Errorf("polygonscan http %d: %s", resp.StatusCode, truncate(string(body), 500))
		}

		if responseLooksRateLimited(body) && attempt < 7 {
			continue
		}
		return body, nil
	}
	return nil, fmt.Errorf("polygonscan: rate limit persists after retries: %s", truncate(string(lastBody), 300))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ReplayBalances rebuilds address balances from transfer rows (wei/raw token units).
func ReplayBalances(rows []TokenTransfer) (map[string]*big.Int, error) {
	bal := make(map[string]*big.Int)
	adjust := func(addr string, delta *big.Int) {
		if addr == "" {
			return
		}
		a := strings.ToLower(addr)
		cur := bal[a]
		if cur == nil {
			cur = big.NewInt(0)
		}
		cur = new(big.Int).Add(cur, delta)
		bal[a] = cur
	}

	for _, r := range rows {
		if r.TxreceiptStatus == "0" {
			continue
		}
		v, ok := new(big.Int).SetString(r.Value, 10)
		if !ok {
			return nil, fmt.Errorf("parse value %q", r.Value)
		}
		neg := new(big.Int).Neg(v)
		from := strings.ToLower(r.From)
		to := strings.ToLower(r.To)
		zero := "0x0000000000000000000000000000000000000000"
		if from != zero {
			adjust(from, neg)
		}
		if to != zero {
			adjust(to, v)
		}
	}
	return bal, nil
}
