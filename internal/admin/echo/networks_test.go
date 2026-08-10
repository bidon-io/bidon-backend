package adminecho

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
