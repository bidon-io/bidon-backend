package admin

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

func (p *demandSourcePolicy) scope(_ AuthContext) resourceScope[DemandSource] {
	return &publicResourceScope[DemandSource]{
		repo: p.repo,
	}
}
