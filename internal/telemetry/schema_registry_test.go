package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestSchemaRegistryRegisterKeepsIDWhenCompatibilityFails(t *testing.T) {
	const subject = "telemetry-events-org.bidon.telemetry.v1.DspRequestSent"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.schemaregistry.v1+json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/subjects/"+subject+"/versions":
			_, _ = w.Write([]byte(`{"id":7}`))
		case r.Method == http.MethodGet && r.URL.Path == "/schemas/ids/7/versions":
			_, _ = w.Write([]byte(`[{"subject":"` + subject + `","version":1}]`))
		case r.Method == http.MethodGet && r.URL.Path == "/subjects/"+subject+"/versions/1":
			_, _ = w.Write([]byte(`{"subject":"` + subject + `","version":1,"id":7,"schemaType":"PROTOBUF","schema":"syntax = \"proto3\";"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/config/"+subject:
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error_code":40301,"message":"forbidden"}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	core, logs := observer.New(zap.WarnLevel)
	reg, err := NewSchemaRegistry(server.URL, zap.New(core))
	if err != nil {
		t.Fatal(err)
	}

	id, err := reg.Register(context.Background(), subject, `syntax = "proto3";`)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if id != 7 {
		t.Errorf("id = %d, want 7", id)
	}

	entries := logs.FilterMessage("schema registry set compatibility").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 compatibility warning, got %d", len(entries))
	}
	fields := entries[0].ContextMap()
	assertField(t, fields, "subject", subject)
	assertField(t, fields, "level", "BACKWARD")
}
