package dbpgtx

import (
	"context"
	"database/sql"
)

type DBPGTx struct {
	lc        string
	conn      *sql.DB
	sqlTx     *sql.Tx
	ctx       context.Context
	txOptions sql.TxOptions
}
