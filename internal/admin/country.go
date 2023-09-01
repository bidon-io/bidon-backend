package admin

import "context"

type Country struct {
	ID int64 `json:"id"`
	CountryAttrs
}

type CountryAttrs struct {
	HumanName  string `json:"human_name"`
	Alpha2Code string `json:"alpha2_code"`
	Alpha3Code string `json:"alpha3_code"`
}

type CountryService = ResourceService[Country, CountryAttrs]

func NewCountryService(store Store) *CountryService {
	return &CountryService{
		repo: store.Countries(),
		policy: &countryPolicy{
			repo: store.Countries(),
		},
	}
}

type CountryRepo = ResourceRepo[Country, CountryAttrs]

type countryPolicy struct {
	repo CountryRepo
}

func (p *countryPolicy) scope(authCtx AuthContext) (resourceScope[Country], error) {
	return &countryScope{
		repo:    p.repo,
		authCtx: authCtx,
	}, nil
}

type countryScope struct {
	repo CountryRepo

	authCtx AuthContext
}

func (s *countryScope) list(ctx context.Context) ([]Country, error) {
	return s.repo.List(ctx)
}

func (s *countryScope) find(ctx context.Context, id int64) (*Country, error) {
	return s.repo.Find(ctx, id)
}
