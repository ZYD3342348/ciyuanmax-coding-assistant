package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientDeviceAndAuthRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer cli_test_token" {
			t.Error("missing bearer token")
		}
		if request.URL.Path == "/api/cli/status" {
			_ = json.NewEncoder(writer).Encode(Status{Email: "user@example.com", BalanceCNY: "12.30"})
			return
		}
		if request.URL.Path == "/api/cli/device/start" {
			_ = json.NewEncoder(writer).Encode(DeviceStart{DeviceCode: "cli_dev_1", UserCode: "ABCD-EF12", Interval: 2})
			return
		}
		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "cli_test_token")
	status, err := client.Status(context.Background())
	if err != nil || status.BalanceCNY != "12.30" {
		t.Fatalf("status = %#v, err = %v", status, err)
	}
	device, err := client.StartDevice(context.Background())
	if err != nil || device.UserCode != "ABCD-EF12" {
		t.Fatalf("device = %#v, err = %v", device, err)
	}
}

func TestClientUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(server.URL, "expired")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := client.Status(ctx)
	if err != ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestListCurrentKeysUsesDeviceEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/cli/keys/current" || request.URL.Query().Get("device_name") != "Dev Mac" {
			t.Fatalf("unexpected URL %s", request.URL.String())
		}
		_, _ = writer.Write([]byte("[]"))
	}))
	defer server.Close()
	client := NewClient(server.URL, "token")
	if _, err := client.ListKeys(context.Background(), "Dev Mac"); err != nil {
		t.Fatal(err)
	}
}
