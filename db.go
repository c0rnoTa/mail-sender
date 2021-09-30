package main

import (
	"database/sql"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

func (App *MyApp) ConncetDB() error {

	var err error

	log.Debug("Initializing Database connector")
	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8", App.config.DB.User, App.config.DB.Password, App.config.DB.Host, App.config.DB.Port, App.config.DB.Database)
	App.db, err = sql.Open("mysql", dataSourceName)
	if err != nil {
		log.Error("Database initialization failed: ", err)
		return err
	}

	// Connect to servers
	log.Debug("Connecting to database server ", App.config.DB.User)
	App.db.SetConnMaxLifetime(time.Second)
	err = App.db.Ping()
	if err != nil {
		log.Error("Database connection failed: ", err)
		return err
	} else {
		log.Info("Connected to Database")
	}
	return nil

}
