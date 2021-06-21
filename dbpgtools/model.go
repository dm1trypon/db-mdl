package dbpgtools

import (
	"context"
	"database/sql"
)

type DBPGTools struct {
	lc        string
	conn      *sql.DB
	ctx       context.Context
	txOptions sql.TxOptions
}
