package admin

import "context"

type App struct {
	ID int64 `json:"id"`
	AppAttrs
	User User `json:"user"`
}

type AppAttrs struct {
	PlatformID  PlatformID     `json:"platform_id"`
	HumanName   string         `json:"human_name"`
	PackageName string         `json:"package_name"`
	UserID      int64          `json:"user_id"`
	AppKey      string         `json:"app_key"`
	Settings    map[string]any `json:"settings"`
}

type PlatformID string

const (
	UnknownPlatformID PlatformID = ""
	IOSPlatformID     PlatformID = "ios"
	AndroidPlatformID PlatformID = "android"
)

type AppService = ResourceService[App, AppAttrs]

func NewAppService(store Store) *AppService {
	return &AppService{
		repo: store.Apps(),

		policy: &appPolicy{
			repo: store.Apps(),
		},
	}
}

type AppRepo interface {
	ResourceRepo[App, AppAttrs]

	ListOwnedByUser(ctx context.Context, userID int64) ([]App, error)
	FindOwnedByUser(ctx context.Context, userID int64, id int64) (*App, error)
}

type appPolicy struct {
	repo AppRepo
}

func (p *appPolicy) scope(authCtx AuthContext) resourceScope[App] {
	return &ownedResourceScope[App]{
		repo:    p.repo,
		authCtx: authCtx,
	}
}
