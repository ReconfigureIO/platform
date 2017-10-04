package config

import (
	"github.com/bshuster-repo/logruzio"
	"github.com/sirupsen/logrus"
)

func SetupLogging(version string, conf *Config) error {
	ctx := logrus.Fields{
		"Environment": conf.Reco.Env,
		"Version":     version,
	}
	hook, err := logruzio.New(conf.Reco.LogzioToken, conf.ProgramName, ctx)
	if err != nil {
		return err
	}
	logrus.AddHook(hook)
	return nil
}
