package api

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestProductionDeviceContract(t *testing.T) {
	if os.Getenv("CIYUANMAX_INTEGRATION") != "1" {
		t.Skip("set CIYUANMAX_INTEGRATION=1 to test the production device contract")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client := NewClient(DefaultBaseURL, "")
	device, err := client.StartDevice(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if device.DeviceCode == "" || device.UserCode == "" || device.VerificationURIComplete == "" {
		t.Fatalf("incomplete device response: %#v", device)
	}
	poll, err := client.PollDevice(ctx, device.DeviceCode)
	if err != nil {
		t.Fatal(err)
	}
	if poll.Status != "pending" {
		t.Fatalf("poll status = %q, want pending", poll.Status)
	}
}
