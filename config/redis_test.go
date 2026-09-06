package config_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redismock/v9"

	"github.com/bidon-io/bidon-backend/config"
)

func TestNewRedisClient_RedisURLNotSet(t *testing.T) {
	t.Setenv("REDIS_URL", "")

	client, err := config.NewRedisClient(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if client != nil {
		t.Fatalf("expected nil client, got %v", client)
	}
	if want := "REDIS_URL is not set"; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
}

func TestNewRedisClient_UnparseableURL(t *testing.T) {
	t.Setenv("REDIS_URL", "http://example.com")

	client, err := config.NewRedisClient(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if client != nil {
		t.Fatalf("expected nil client, got %v", client)
	}
	if want := "redis.ParseURL"; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
}

func TestNewRedisClient_Unreachable(t *testing.T) {
	// Bind and immediately close a listener to obtain a port nothing listens on,
	// guaranteeing a fast "connection refused" instead of a slow timeout.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen(): %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("ln.Close(): %v", err)
	}

	t.Setenv("REDIS_URL", "redis://"+addr)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := config.NewRedisClient(ctx, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if client != nil {
		t.Fatalf("expected nil client, got %v", client)
	}
	if want := "redis ping"; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
}

func TestNewRedisClient_Success(t *testing.T) {
	mr := miniredis.RunT(t)

	t.Setenv("REDIS_URL", "redis://"+mr.Addr()+"/0")

	client, err := config.NewRedisClient(context.Background(), 1)
	if err != nil {
		t.Fatalf("NewRedisClient(): %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Errorf("client.Ping(): %v", err)
	}
}

func TestRedisPinger_Ping(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		pinger := config.NewRedisPinger(nil)
		if err := pinger.Ping(context.Background()); err != nil {
			t.Errorf("Ping() = %v, want nil", err)
		}
	})

	t.Run("healthy", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		mock.ExpectPing().SetVal("PONG")

		pinger := config.NewRedisPinger(client)
		if err := pinger.Ping(context.Background()); err != nil {
			t.Errorf("Ping() = %v, want nil", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("ExpectationsWereMet(): %v", err)
		}
	})

	t.Run("unhealthy", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		wantErr := errors.New("connection refused")
		mock.ExpectPing().SetErr(wantErr)

		pinger := config.NewRedisPinger(client)
		err := pinger.Ping(context.Background())
		if !errors.Is(err, wantErr) {
			t.Errorf("Ping() = %v, want %v", err, wantErr)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("ExpectationsWereMet(): %v", err)
		}
	})
}

