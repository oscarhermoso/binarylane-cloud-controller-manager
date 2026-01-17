package binarylane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var ErrVpcNotFound = errors.New("VPC not found")

func (c *BinaryLaneClient) GetVpc(ctx context.Context, vpcID int64) (*Vpc, error) {
	resp, err := c.GetVpcsVpcId(ctx, vpcID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VPC: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if resp.StatusCode == 404 {
		return nil, ErrVpcNotFound
	}
	if resp.StatusCode != 200 {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("API error (status %d), failed to read response: %w", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var vpcResp VpcResponse
	if err := json.Unmarshal(body, &vpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &vpcResp.Vpc, nil
}

func (c *BinaryLaneClient) UpdateVpc(ctx context.Context, vpcID int64, req UpdateVpcRequest) (*Vpc, error) {
	resp, err := c.PutVpcsVpcId(ctx, vpcID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update VPC: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if resp.StatusCode == 404 {
		return nil, ErrVpcNotFound
	}
	if resp.StatusCode != 200 {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("API error (status %d), failed to read response: %w", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var vpcResp VpcResponse
	if err := json.Unmarshal(body, &vpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &vpcResp.Vpc, nil
}
