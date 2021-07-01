package main

import (
	"fmt"
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

	res, ok := dbPgToolsInst.Query("SELECT * FROM users;")
	if !ok {
		return
	}

	var username string
	var id int64

	val, ok := res[0]["username"]
	if !ok {
		return
	}

	username, ok = val.(string)
	if !ok {
		return
	}

	val, ok = res[0]["id"]
	if !ok {
		return
	}

	id, ok = val.(int64)
	if !ok {
		return
	}

	logger.InfoJ(LC, fmt.Sprint("username: ", username, " | id: ", id))

	<-dbPgConnInst.GetChDisconnected()
}
