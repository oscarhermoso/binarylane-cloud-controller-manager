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

type BinaryLaneClient struct {
	*Client
}

func NewBinaryLaneClient(token string) (*BinaryLaneClient, error) {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	client, err := NewClient(
		defaultBaseURL,
		WithHTTPClient(httpClient),
		WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("User-Agent", "binarylane-cloud-controller-manager/v0") // TODO: set version dynamically
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &BinaryLaneClient{
		Client: client,
	}, nil
}
