package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type (
	APIEntry struct {
		ID        int       `json:"id"`
		StartTime time.Time `json:"start_time"`
		EndTime   NullTime  `json:"end_time"`
		Note      string    `json:"note"`
	}

	NullTime struct {
		Time  time.Time `json:"Time"`
		Valid bool      `json:"Valid"`
	}

	Response struct {
		Success bool   `json:"success"`
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
		Data    any    `json:"data,omitempty"`
	}

	EntriesResponse struct {
		Total   int        `json:"total"`
		Entries []APIEntry `json:"entries"`
	}

	MarkImportedResponse struct {
		ImportedCount  int `json:"imported_count"`
		RemainingCount int `json:"remaining_count"`
	}
)

type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// fetches all unimported entries from the API
func (c *APIClient) GetUnimportedEntries() ([]APIEntry, error) {
	url := fmt.Sprintf("%s/api/entries", c.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		var errRes Response
		if err := json.NewDecoder(res.Body).Decode(&errRes); err != nil {
			return nil, fmt.Errorf("error decoding error response: %w", err)
		}
		return nil, fmt.Errorf("%s", errRes.Message)
	}

	var apiRes Response
	if err := json.NewDecoder(res.Body).Decode(&apiRes); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if !apiRes.Success {
		return nil, fmt.Errorf("%s", apiRes.Message)
	}

	// Convert the Data field to EntriesResponse
	dataJSON, err := json.Marshal(apiRes.Data)
	if err != nil {
		return nil, fmt.Errorf("error re-encoding data: %w", err)
	}

	var entriesResp EntriesResponse
	// here breaks
	if err := json.Unmarshal(dataJSON, &entriesResp); err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("error decoding entries data: %w", err)
	}

	return entriesResp.Entries, nil
}

// send requests to mark specified entries as imported
func (c *APIClient) MarkEntriesAsImported(entryIDs []int64) (string, error) {
	url := fmt.Sprintf("%s/api/entries/mark", c.baseURL)
	reqData := struct {
		EntryIDs []int64 `json:"entry_ids"`
	}{EntryIDs: entryIDs}

	reqBody, err := json.Marshal(reqData)
	if err != nil {
		return "", fmt.Errorf("error encoding request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errRes Response
		if err := json.NewDecoder(resp.Body).Decode(&errRes); err != nil {
			return "", fmt.Errorf("error decoding error response: %w", err)
		}
		return "", fmt.Errorf("%s", errRes.Message)
	}

	var apiRes Response
	if err := json.NewDecoder(resp.Body).Decode(&apiRes); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	if !apiRes.Success {
		return "", fmt.Errorf(apiRes.Message)
	}

	dataJSON, err := json.Marshal(apiRes.Data)
	if err != nil {
		return "", fmt.Errorf("error re-encoding data: %w", err)
	}

	var res MarkImportedResponse
	if err := json.Unmarshal(dataJSON, &res); err != nil {
		return "", fmt.Errorf("error decoding mark response data: %w", err)
	}
	return fmt.Sprintf("Successfully imported %d of %d entries.", res.ImportedCount, res.RemainingCount), nil
}
