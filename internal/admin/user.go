package admin

import (
	"context"
	"fmt"
)

type User struct {
	ID int64 `json:"id"`
	UserAttrs
}

type UserAttrs struct {
	Email string `json:"email"`
}

type UserService = ResourceService[User, UserAttrs]

func NewUserService(store Store) *UserService {
	return &UserService{
		repo: store.Users(),
		policy: &userPolicy{
			repo: store.Users(),
		},
	}
}

type UserRepo = ResourceRepo[User, UserAttrs]

type userPolicy struct {
	repo UserRepo
}

func (p *userPolicy) scope(authCtx AuthContext) (resourceScope[User], error) {
	if !authCtx.IsAdmin() {
		return nil, fmt.Errorf("access denied")
	}

	return &userScope{
		repo:    p.repo,
		authCtx: authCtx,
	}, nil
}

type userScope struct {
	repo UserRepo

	authCtx AuthContext
}

func (s *userScope) list(ctx context.Context) ([]User, error) {
	return s.repo.List(ctx)
}

func (s *userScope) find(ctx context.Context, id int64) (*User, error) {
	return s.repo.Find(ctx, id)
}
