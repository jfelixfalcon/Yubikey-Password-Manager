package db

import (
	"log"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	Host     string `json:"host"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"dbname"`
	Port     string `json:"port"`
	Ssl      string `json:"ssl"`
	sync.Mutex
}

func Connect(config *Config) (*gorm.DB, error) {

	dbConf := "host=" + config.Host +
		" user=" + config.User +
		" password=" + config.Password +
		" dbname=" + config.Name +
		" port=" + config.Port +
		" sslmode=" + config.Ssl

	db, err := gorm.Open(postgres.Open(dbConf), &gorm.Config{})

	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	return db, nil

}
