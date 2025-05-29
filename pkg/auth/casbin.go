package auth

import (
	"time"

	"github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	mongodbadapter "github.com/casbin/mongodb-adapter/v3"
	mongooptions "go.mongodb.org/mongo-driver/mongo/options" // mongodbadapter uses mongo-driver/v1
)

type CasbinService struct {
	enforcer *casbin.Enforcer
}

func NewService(mongoURI string) (*CasbinService, error) {
	// load auth model
	m, err := model.NewModelFromFile("config/auth_model.conf")
	if err != nil {
		return nil, err
	}

	// create mongo adapter
	serverAPI := mongooptions.ServerAPI(mongooptions.ServerAPIVersion1)
	opts := mongooptions.Client().SetTimeout(5 * time.Second).ApplyURI(mongoURI).SetServerAPIOptions(serverAPI)
	var (
		db   = "casbin"
		coll = "rules"
	)
	adapter, err := mongodbadapter.NewAdapterWithCollectionName(opts, db, coll)
	if err != nil {
		return nil, err
	}

	// create enforcer
	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, err
	}
	enforcer.EnableAutoSave(true)

	return &CasbinService{
		enforcer: enforcer,
	}, nil
}

func (cb *CasbinService) CheckPermission(sub, obj, act string) (bool, error) {
	eft, err := cb.enforcer.Enforce(sub, obj, act)
	if err != nil {
		return false, errors.ErrCasbinEnforcement.WithInternalErr(err)
	}
	return eft, nil
}

func (cb *CasbinService) LoadPolicy() error {
	if err := cb.enforcer.LoadPolicy(); err != nil {
		return errors.ErrCasbinPolicyLoad.WithInternalErr(err)
	}
	return nil
}
