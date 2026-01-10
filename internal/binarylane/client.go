package binarylane

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "https://api.binarylane.com.au/v2"
)

// BinaryLaneClient wraps the generated client with convenience methods
type BinaryLaneClient struct {
	*Client
}

// NewBinaryLaneClient creates a new BinaryLane API client
func NewBinaryLaneClient(token string) (*BinaryLaneClient, error) {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request editor to add authorization header
	authEditor := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}

	client, err := NewClient(defaultBaseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &BinaryLaneClient{
		Client: client,
	}, nil
}
