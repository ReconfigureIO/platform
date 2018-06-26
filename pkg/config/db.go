package config

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	log "github.com/sirupsen/logrus"
)

type GormLogger struct{}

func (*GormLogger) Print(v ...interface{}) {
	if v[0] == "sql" {
		log.WithFields(log.Fields{"module": "gorm", "type": "sql"}).Print(v[3])
	}
	if v[0] == "log" {
		log.WithFields(log.Fields{"module": "gorm", "type": "log"}).Print(v[2])
	}
}

func SetupDB(conf *Config) *gorm.DB {
	db, err := gorm.Open("postgres", conf.DbUrl)
	db.SetLogger(&GormLogger{})

	if conf.Reco.Env != "production" {
		db.LogMode(true)
	}

	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	return db
}
