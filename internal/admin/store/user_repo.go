package adminstore

import (
	"context"
	"github.com/bidon-io/bidon-backend/internal/admin"
	"github.com/bidon-io/bidon-backend/internal/db"
	"gorm.io/gorm/clause"
)

type UserRepo struct {
	resourceRepo[admin.User, admin.UserAttrs, db.User]
	PasswordGenerator PasswordGenerator
}

//go:generate go run -mod=mod github.com/matryer/moq@latest -out  mocks/mocks.go -pkg mocks . PasswordGenerator

type PasswordGenerator interface {
	Generate(password string) (string, error)
}

type userMapper struct{}

func NewUserRepo(d *db.DB) *UserRepo {
	return &UserRepo{
		resourceRepo[admin.User, admin.UserAttrs, db.User]{
			db:           d,
			mapper:       userMapper{},
			associations: []string{},
		},
		&admin.PasswordService{},
	}
}

func (r *UserRepo) Create(ctx context.Context, attrs *admin.UserAttrs) (*admin.User, error) {
	dbModel := r.mapper.dbModel(attrs, 0)

	passwordHash, err := r.PasswordGenerator.Generate(attrs.Password)
	if err != nil {
		return nil, err
	}
	dbModel.PasswordHash = passwordHash

	if err := r.db.WithContext(ctx).Create(dbModel).Error; err != nil {
		return nil, err
	}

	resource := r.mapper.resource(dbModel)
	return &resource, nil
}

func (r *UserRepo) Update(ctx context.Context, id int64, attrs *admin.UserAttrs) (*admin.User, error) {
	dbModel := r.mapper.dbModel(attrs, id)

	if attrs.Password != "" {
		passwordHash, err := r.PasswordGenerator.Generate(attrs.Password)
		if err != nil {
			return nil, err
		}
		dbModel.PasswordHash = passwordHash
	}

	if err := r.db.WithContext(ctx).Model(dbModel).Select("*").Where("id = ?", id).Clauses(clause.Returning{}).Updates(&dbModel).Error; err != nil {
		return nil, err
	}

	resource := r.mapper.resource(dbModel)
	return &resource, nil
}

//lint:ignore U1000 this method is used by generic struct
func (m userMapper) dbModel(u *admin.UserAttrs, id int64) *db.User {
	return &db.User{
		Model:   db.Model{ID: id},
		Email:   u.Email,
		IsAdmin: u.IsAdmin,
	}
}

//lint:ignore U1000 this method is used by generic struct
func (m userMapper) resource(u *db.User) admin.User {
	return admin.User{
		ID:        u.ID,
		UserAttrs: m.resourceAttrs(u),
	}
}

func (m userMapper) resourceAttrs(u *db.User) admin.UserAttrs {
	return admin.UserAttrs{
		Email:   u.Email,
		IsAdmin: u.IsAdmin,
	}
}
