package binarylane

import (
"testing"
)

func TestNewBinaryLaneClient(t *testing.T) {
client, err := NewBinaryLaneClient("test-token")
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if client == nil {
t.Fatal("expected client to be created")
}
if client.Client == nil {
t.Fatal("expected generated client to be set")
}
}

func TestNewBinaryLaneClientEmpty(t *testing.T) {
// Empty token should still create a client
client, err := NewBinaryLaneClient("")
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if client == nil {
t.Fatal("expected client to be created")
}
}
