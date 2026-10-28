package adminecho

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/bidon-io/bidon-backend/internal/adapter"
)

func TestGetNetworks_ReturnsRegistryCatalog(t *testing.T) {
	t.Parallel()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/networks", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("authCtx", echoAuthContext{})

	s := &Server{}
	if err := s.GetNetworks(c); err != nil {
		t.Fatalf("GetNetworks() error = %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var got []networkCatalogItem
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}

	want := adapter.Networks()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}

	byKey := make(map[string]networkCatalogItem, len(got))
	for _, item := range got {
		byKey[item.Key] = item
	}

	moloco, ok := byKey[string(adapter.MolocoKey)]
	if !ok {
		t.Fatal("missing moloco")
	}
	if moloco.Label != "Moloco" || moloco.AccountType != "DemandSourceAccount::Moloco" {
		t.Fatalf("moloco = %+v", moloco)
	}
	if !moloco.SupportsBidding || moloco.SupportsWaterfall {
		t.Fatalf("moloco flags = %+v", moloco)
	}

	admob, ok := byKey[string(adapter.AdmobKey)]
	if !ok {
		t.Fatal("missing admob")
	}
	if admob.SupportsBidding || !admob.SupportsWaterfall {
		t.Fatalf("admob flags = %+v", admob)
	}
}

// networkCatalogItem is hand-written; keep it aligned with network.schema.json.
func TestNetworkCatalogItem_MatchesOpenAPISchema(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../openapi/schemas/network.schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Required   []string                  `json:"required"`
		Properties map[string]map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}

	encoded, err := json.Marshal(networkCatalogItem{})
	if err != nil {
		t.Fatalf("encode item: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("decode item: %v", err)
	}

	for name := range fields {
		if _, ok := schema.Properties[name]; !ok {
			t.Errorf("response field %q not in schema", name)
		}
	}
	for name := range schema.Properties {
		if _, ok := fields[name]; !ok {
			t.Errorf("schema property %q not in response", name)
		}
	}
	for _, name := range schema.Required {
		if _, ok := fields[name]; !ok {
			t.Errorf("required schema property %q not in response", name)
		}
	}
}

func TestNetworkCatalog_KeysAreValidAdapterKeys(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../openapi/schemas/adapter-key.schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Enum []string `json:"enum"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	allowed := map[string]bool{}
	for _, k := range schema.Enum {
		allowed[k] = true
	}
	for _, n := range adapter.Networks() {
		if !allowed[string(n.Key)] {
			t.Errorf("%s missing from adapter-key.schema.json enum", n.Key)
		}
	}
}
