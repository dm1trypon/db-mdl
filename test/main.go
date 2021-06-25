package main

import (
	"time"

	"github.com/dm1trypon/db-mdl/dbpgconnector"
	logger "github.com/dm1trypon/easy-logger"
)

// LC - logging's category
const LC = "MAIN"

func main() {
	logCfg := logger.Cfg{
		AppName: "DB_MDL",
		LogPath: "",
		Level:   0,
	}

	logger.SetConfig(logCfg)

	logger.InfoJ(LC, "STARTING SERVICE")

	dbPgConnInst := new(dbpgconnector.DBPGConnector).Create()

	cfg := dbpgconnector.Config{
		Username:             "postgres",
		Password:             "mpassword",
		Host:                 "localhost",
		Port:                 5432,
		DbName:               "db_game",
		SSLMode:              0,
		ConnectTimeout:       10,
		PingInterval:         2 * time.Second,
		ReconnectionInterval: 2 * time.Second,
		Certs:                dbpgconnector.Certs{},
	}
	dbPgConnInst.SetConfig(cfg)

	go dbPgConnInst.Run()
	<-dbPgConnInst.GetChConnected()

	// Configuring the access level and transaction isolation level.
	settings := map[uint8]bool{
		0: false,
	}
	dbPgConnInst.SetDBPGToolsList(settings)

	// Getting a tool with the necessary isolation and access level to work with the database.
	dbPgToolsInst := dbPgConnInst.GetDBPGTools(0, false)
	if dbPgToolsInst == nil {
		return
	}

	for {
		if _, ok := dbPgToolsInst.Exec("INSERT INTO persons VALUES (1, 'testuser');"); !ok {
			time.Sleep(2 * time.Second)
			continue
		} else {
			break
		}
	}

	<-dbPgConnInst.GetChDisconnected()
}
