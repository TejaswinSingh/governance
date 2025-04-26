package auth

import (
	"github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
)

type CasbinService struct {
	enforcer *casbin.Enforcer
}

func NewService() (*CasbinService, error) {
	m, err := model.NewModelFromFile("config/model.conf")
	if err != nil {
		return nil, err
	}
	adapter := fileadapter.NewAdapter("config/policy.csv")
	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, err
	}
	return &CasbinService{
		enforcer: enforcer,
	}, nil
}

func (csb *CasbinService) CheckPermission(sub, obj, act string) (bool, error) {
	if err := csb.enforcer.LoadPolicy(); err != nil {
		return false, errors.ErrCasbinPolicyLoad.SetInternalErr(err)
	}
	eft, err := csb.enforcer.Enforce(sub, obj, act)
	if err != nil {
		return false, errors.ErrCasbinEnforcement.SetInternalErr(err)
	}
	return eft, nil
}
