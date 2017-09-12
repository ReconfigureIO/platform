package config

import (
	"github.com/caarlos0/env"

	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/ReconfigureIO/platform/service/intercom"
)

type Config struct {
	DbUrl     string     `env:"DATABASE_URL"`
	SecretKey string     `env:"SECRET_KEY_BASE"`
	StripeKey string     `env:"STRIPE_KEY"`
	Port      string     `env:"PORT"`
	Reco      RecoConfig `env:"RECO"`
}

type RecoConfig struct {
	Env             string `env:"RECO_ENV"`
	PlatformMigrate bool   `env:"RECO_PLATFORM_MIGRATE"`
	FeatureDeploy   bool   `env:"RECO_FEATURE_DEPLOY"`
	AWS             aws.ServiceConfig
	Deploy          deployment.ServiceConfig
	Intercom        intercom.ServiceConfig
}

func ParseEnvConfig() (*Config, error) {
	conf := Config{}

	err := env.Parse(&conf)
	if err != nil {
		return nil, err
	}

	err = env.Parse(&conf.Reco)
	if err != nil {
		return nil, err
	}

	err = env.Parse(&conf.Reco.AWS)
	if err != nil {
		return nil, err
	}

	err = env.Parse(&conf.Reco.Deploy)
	if err != nil {
		return nil, err
	}

	err = env.Parse(&conf.Reco.Intercom)
	if err != nil {
		return nil, err
	}

	return &conf, nil
}
