package admin

import "context"

type DemandSource struct {
	ID int64 `json:"id"`
	DemandSourceAttrs
}

type DemandSourceAttrs struct {
	HumanName string `json:"human_name"`
	ApiKey    string `json:"api_key"`
}

type DemandSourceService = ResourceService[DemandSource, DemandSourceAttrs]

func NewDemandSourceService(store Store) *DemandSourceService {
	return &DemandSourceService{
		repo: store.DemandSources(),
		policy: &demandSourcePolicy{
			repo: store.DemandSources(),
		},
	}
}

type DemandSourceRepo = ResourceRepo[DemandSource, DemandSourceAttrs]

type demandSourcePolicy struct {
	repo DemandSourceRepo
}

func (p *demandSourcePolicy) scope(authCtx AuthContext) (resourceScope[DemandSource], error) {
	return &demandSourceScope{
		repo:    p.repo,
		authCtx: authCtx,
	}, nil
}

type demandSourceScope struct {
	repo DemandSourceRepo

	authCtx AuthContext
}

func (s *demandSourceScope) list(ctx context.Context) ([]DemandSource, error) {
	return s.repo.List(ctx)
}

func (s *demandSourceScope) find(ctx context.Context, id int64) (*DemandSource, error) {
	return s.repo.Find(ctx, id)
}
