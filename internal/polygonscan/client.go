package polygonscan

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	apiBaseURL         = "https://api.etherscan.io/v2/api"
	polygonChainID     = 137
	minRequestInterval = 400 * time.Millisecond
)

type Client struct {
	c             *http.Client
	apiKey        string
	mu            sync.Mutex
	lastRequestAt time.Time
}

func NewClinet(apiKey string) *Client {
	if apiKey == "" {
		panic("polygonscan api key is empty")
	}
	return &Client{
		c:       http.DefaultClient,
		apiKey:  apiKey,
		mu:      sync.Mutex{},
		lastRequestAt: time.Time{},
	}
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
}

// FetchAllTokenTx paginates tokentx until a page returns fewer than offset rows or maxPages reached (0 = unlimited).
func (c *Client) FetchAllTokenTx(contract string, offset int, pause time.Duration) ([]TokenTransfer, error) {
	if offset <= 0 {
		offset = 1000
	}
	sort := "asc"
	var all []TokenTransfer
	page := 1
	for {
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
		page++
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

func (c *Client) throttle() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.lastRequestAt.IsZero() {
		if elapsed := time.Since(c.lastRequestAt); elapsed < minRequestInterval {
			time.Sleep(minRequestInterval - elapsed)
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
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, errors.New("api key is empty")
	}
	q.Set("apikey", strings.TrimSpace(c.apiKey))
	if q.Get("chainid") == "" {
		q.Set("chainid", strconv.Itoa(polygonChainID))
	}
	u, err := url.Parse(apiBaseURL)
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
		resp, err := c.c.Do(req)
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


