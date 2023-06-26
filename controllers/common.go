package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/controllers/database"
)

type databaseCreator func(ctx context.Context, s v1alpha1.DatabaseSpec, client client.Client, logger logr.Logger) (database.Interface, error)
