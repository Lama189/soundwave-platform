package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/shopspring/decimal"
)

type OpenERAdapter struct {
	httpClient *http.Client
	baseURL    string
}

func NewOpenERAdapter(httpClient *http.Client, baseURL string) *OpenERAdapter {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	if baseURL == "" {
		baseURL = "https://open.er-api.com/v6/latest"
	}

	return &OpenERAdapter{
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

func (a *OpenERAdapter) FetchRates(ctx context.Context, base domain.Currency) (map[domain.Currency]decimal.Decimal, error) {
	endpoint := fmt.Sprintf("%s/%s", a.baseURL, base)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create fx request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fx provider request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fx provider error: %d", resp.StatusCode)
	}

	var apiResp erAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode fx response: %w", err)
	}

	if apiResp.Result != "success" {
		return nil, fmt.Errorf("fx provider error: %s", apiResp.Result)
	}

	result := make(map[domain.Currency]decimal.Decimal)
	result[base] = decimal.NewFromInt(1)

	for curStr, rate := range apiResp.Rates {
		cur := domain.Currency(curStr)
		if cur.IsValid() && rate.IsPositive() {
			result[cur] = rate
		}
	}

	return result, nil
}
