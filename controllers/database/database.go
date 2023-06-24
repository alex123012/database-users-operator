package database

import (
	"context"
	"fmt"
	"io"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
)

type Interface interface {
	io.Closer
	CreateUser(ctx context.Context, username, password string) (map[string]string, error)
	DeleteUser(ctx context.Context, username string) error
	ApplyPrivileges(ctx context.Context, username string, privileges []v1alpha1.PrivilegeSpec) error
	RevokePrivileges(ctx context.Context, username string, privileges []v1alpha1.PrivilegeSpec) error
}

func NewDatabase(s v1alpha1.DatabaseSpec) (Interface, error) {
	switch s.Type {
	case v1alpha1.PostgreSQL:
		// s.PostgreSQL
		return nil, nil
	}
	return nil, fmt.Errorf("can't find supported DB type '%s'", s.Type)
}
