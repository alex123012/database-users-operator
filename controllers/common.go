package controllers

import (
	"context"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/controllers/database"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type databaseCreator func(ctx context.Context, s v1alpha1.DatabaseSpec, kClient client.Client, logger logr.Logger) (database.Interface, error)
