package config

import (
	"github.com/caarlos0/env"

	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/ReconfigureIO/platform/service/events"
)

type Config struct {
	ProgramName string     `env:"RECO_NAME"`
	DbUrl       string     `env:"DATABASE_URL"`
	SecretKey   string     `env:"SECRET_KEY_BASE"`
	StripeKey   string     `env:"STRIPE_KEY"`
	Port        string     `env:"PORT"`
	Reco        RecoConfig `env:"RECO"`
}

type RecoConfig struct {
	Env             string `env:"RECO_ENV"`
	PlatformMigrate bool   `env:"RECO_PLATFORM_MIGRATE"`
	FeatureDeploy   bool   `env:"RECO_FEATURE_DEPLOY"`
	LogzioToken     string `env:"LOGZIO_TOKEN"`
	FeatureIntercom bool   `env:"RECO_FEATURE_INTERCOM"`
	AWS             aws.ServiceConfig
	Deploy          deployment.ServiceConfig
	Intercom        events.IntercomConfig
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
