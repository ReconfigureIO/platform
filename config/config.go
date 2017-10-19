package config

import (
	"github.com/caarlos0/env"

	"github.com/ReconfigureIO/platform/service/aws"
	"github.com/ReconfigureIO/platform/service/deployment"
	"github.com/ReconfigureIO/platform/service/events"
	stripe "github.com/stripe/stripe-go"
)

type Config struct {
	ProgramName string     `env:"RECO_NAME"`
	DbUrl       string     `env:"DATABASE_URL"`
	SecretKey   string     `env:"SECRET_KEY_BASE"`
	StripeKey   string     `env:"STRIPE_KEY"`
	Port        string     `env:"PORT"`
	Reco        RecoConfig `env:"RECO"`
	Host        string     `env:"RECO_HOST_NAME"`
}

type RecoConfig struct {
	Env                     string `env:"RECO_ENV"`
	PlatformMigrate         bool   `env:"RECO_PLATFORM_MIGRATE"`
	LogzioToken             string `env:"LOGZIO_TOKEN"`
	FeatureIntercom         bool   `env:"RECO_FEATURE_INTERCOM"`
	FeatureDepQueue         bool   `env:"RECO_FEATURE_DEP_QUEUE"`
	FeatureUseSpotInstances bool   `env:"RECO_FEATURE_USE_SPOT_INSTANCES"`
	AWS                     aws.ServiceConfig
	Deploy                  deployment.ServiceConfig
	Intercom                events.IntercomConfig
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

	stripe.Key = conf.StripeKey

	return &conf, nil
}
