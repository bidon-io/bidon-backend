package adminecho

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/bidon-io/bidon-backend/internal/admin"
	"github.com/bidon-io/bidon-backend/internal/admin/resource"
)

type lineItemCreateServiceStub struct {
	create func(ctx context.Context, authCtx admin.AuthContext, attrs *admin.LineItemAttrs) (*admin.LineItemResource, error)
}

func (s lineItemCreateServiceStub) List(context.Context, admin.AuthContext, map[string][]string) (*resource.Collection[admin.LineItemResource], error) {
	panic("not implemented")
}

func (s lineItemCreateServiceStub) Find(context.Context, admin.AuthContext, int64) (*admin.LineItemResource, error) {
	panic("not implemented")
}

func (s lineItemCreateServiceStub) Create(ctx context.Context, authCtx admin.AuthContext, attrs *admin.LineItemAttrs) (*admin.LineItemResource, error) {
	return s.create(ctx, authCtx, attrs)
}

func (s lineItemCreateServiceStub) Update(context.Context, admin.AuthContext, int64, *admin.LineItemAttrs) (*admin.LineItemResource, error) {
	panic("not implemented")
}

func (s lineItemCreateServiceStub) Delete(context.Context, admin.AuthContext, int64) error {
	panic("not implemented")
}

type echoAuthContext struct{}

func (echoAuthContext) UserID() int64 { return 1 }
func (echoAuthContext) IsAdmin() bool { return true }

func TestServer_CreateLineItem_statusCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		alreadyExists  bool
		wantStatusCode int
	}{
		{
			name:           "new line item returns 201",
			alreadyExists:  false,
			wantStatusCode: http.StatusCreated,
		},
		{
			name:           "existing line item returns 200",
			alreadyExists:  true,
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := &Server{
				LineItemHandler: &lineItemServiceHandler{
					service: lineItemCreateServiceStub{
						create: func(_ context.Context, _ admin.AuthContext, _ *admin.LineItemAttrs) (*admin.LineItemResource, error) {
							return &admin.LineItemResource{
								LineItem: &admin.LineItem{
									ID:            42,
									AlreadyExists: tt.alreadyExists,
								},
								Permissions: admin.ResourceInstancePermissions{
									Update: true,
									Delete: true,
								},
							}, nil
						},
					},
				},
			}

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/line_items", strings.NewReader(`{"human_name":"test","app_id":1,"account_id":1,"account_type":"DemandSourceAccount::admob","ad_type":"banner","format":"banner","extra":{"ad_unit_id":"x"}}`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("authCtx", echoAuthContext{})

			if err := server.CreateLineItem(c); err != nil {
				t.Fatalf("CreateLineItem() error = %v", err)
			}
			if rec.Code != tt.wantStatusCode {
				t.Fatalf("status code = %d, want %d", rec.Code, tt.wantStatusCode)
			}

			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			permissions, ok := body["_permissions"].(map[string]any)
			if !ok {
				t.Fatalf("response missing _permissions: %#v", body)
			}
			if permissions["update"] != true || permissions["delete"] != true {
				t.Fatalf("_permissions = %#v, want update/delete true", permissions)
			}
		})
	}
}
