package internal

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

func NewService(log *logrus.Logger, ssm ssmAgent) *Service {
	return &Service{
		log: log,
		ssm: &ssmClient{
			ssm: ssm,
		},
	}
}

type Service struct {
	log *logrus.Logger
	ssm *ssmClient
}

func (s *Service) CreateDBWithOwner(ctx context.Context, resourceProperties map[string]interface{}) (string, error) {
	cfgParams, err := decodeConfig(resourceProperties)
	id := fmt.Sprintf("%s/%s", cfgParams.Connection.Host, cfgParams.DatabaseName)
	if err != nil {
		s.log.WithError(err).WithField("resource_properties", resourceProperties).Error("decoding config failed")
		return id, fmt.Errorf("loading config from resource properties: %w", err)
	}
	cfg, err := s.ssm.loadConfigParameters(ctx, cfgParams)
	if err != nil {
		return id, fmt.Errorf("loading config from resource properties: %w", err)
	}
	db, err := open(cfg.Connection)
	if err != nil {
		return id, fmt.Errorf("connection to postgres: %w", err)
	}
	defer db.Close()

	s.log.WithField("user", cfg.User.Name).Info("Creating user")

	err = db.CreateUser(ctx, cfg.User.Name, cfg.User.Password)
	if err != nil {
		return id, fmt.Errorf("creating user failed: %w", err)
	}

	s.log.WithField("db_name", cfg.DatabaseName).Info("Creating database")

	err = db.CreateDBWithOwner(ctx, cfg.DatabaseName, cfg.User.Name)
	if err != nil {
		return id, fmt.Errorf("creating db failed: %w", err)
	}

	s.log.WithFields(map[string]interface{}{
		"db_name": cfg.DatabaseName,
		"user":    cfg.User.Name,
	}).Info("Successfully created user and database")

	if len(cfg.Extensions) > 0 {
		dbConn, err := openDatabase(cfg.Connection, cfg.DatabaseName)
		if err != nil {
			return id, fmt.Errorf("conn to created database to install extensions failed: %w", err)
		}
		err = dbConn.CreateExtensions(ctx, cfg.Extensions...)
		if err != nil {
			return id, err
		}
	}

	return id, nil
}
