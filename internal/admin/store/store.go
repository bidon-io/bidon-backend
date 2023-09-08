package adminstore

import (
	"github.com/bidon-io/bidon-backend/internal/admin"
	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bwmarrin/snowflake"
)

type Store struct {
	appRepo                  *AppRepo
	appDemandProfileRepo     *AppDemandProfileRepo
	auctionConfigurationRepo *AuctionConfigurationRepo
	countryRepo              *CountryRepo
	demandSourceRepo         *DemandSourceRepo
	demandSourceAccountRepo  *DemandSourceAccountRepo
	lineItemRepo             *LineItemRepo
	segmentRepo              *SegmentRepo
	userRepo                 *UserRepo
}

func New(db *db.DB, snowflakeNode *snowflake.Node) *Store {
	return &Store{
		appRepo:                  NewAppRepo(db, snowflakeNode),
		appDemandProfileRepo:     NewAppDemandProfileRepo(db, snowflakeNode),
		auctionConfigurationRepo: NewAuctionConfigurationRepo(db, snowflakeNode),
		countryRepo:              NewCountryRepo(db, snowflakeNode),
		demandSourceRepo:         NewDemandSourceRepo(db, snowflakeNode),
		demandSourceAccountRepo:  NewDemandSourceAccountRepo(db, snowflakeNode),
		lineItemRepo:             NewLineItemRepo(db, snowflakeNode),
		segmentRepo:              NewSegmentRepo(db, snowflakeNode),
		userRepo:                 NewUserRepo(db, snowflakeNode),
	}
}

func (s *Store) Apps() admin.AppRepo {
	return s.appRepo
}

func (s *Store) AppDemandProfiles() admin.AppDemandProfileRepo {
	return s.appDemandProfileRepo
}

func (s *Store) AuctionConfigurations() admin.AuctionConfigurationRepo {
	return s.auctionConfigurationRepo
}

func (s *Store) Countries() admin.CountryRepo {
	return s.countryRepo
}

func (s *Store) DemandSources() admin.DemandSourceRepo {
	return s.demandSourceRepo
}

func (s *Store) DemandSourceAccounts() admin.DemandSourceAccountRepo {
	return s.demandSourceAccountRepo
}

func (s *Store) LineItems() admin.LineItemRepo {
	return s.lineItemRepo
}

func (s *Store) Segments() admin.SegmentRepo {
	return s.segmentRepo
}

func (s *Store) Users() admin.UserRepo {
	return s.userRepo
}

func platformID(platformID db.PlatformID) admin.PlatformID {
	switch platformID {
	case db.AndroidPlatformID:
		return admin.AndroidPlatformID
	case db.IOSPlatformID:
		return admin.IOSPlatformID
	default:
		return admin.UnknownPlatformID
	}
}

func dbPlatformID(platformID admin.PlatformID) db.PlatformID {
	switch platformID {
	case admin.AndroidPlatformID:
		return db.AndroidPlatformID
	case admin.IOSPlatformID:
		return db.IOSPlatformID
	default:
		return db.UnknownPlatformID
	}
}
