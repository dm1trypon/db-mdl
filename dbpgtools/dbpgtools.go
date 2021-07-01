package dbpgtools

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/dm1trypon/db-mdl/dbpgtx"
	logger "github.com/dm1trypon/easy-logger"
)

func (d *DBPGTools) Create(conn *sql.DB, ctx context.Context, txOptions sql.TxOptions) *DBPGTools {
	d = &DBPGTools{
		lc:        "DB_PG_TOOLS",
		conn:      conn,
		ctx:       ctx,
		txOptions: txOptions,
	}

	return d
}

func (d *DBPGTools) CustomTransaction() *dbpgtx.DBPGTx {
	return new(dbpgtx.DBPGTx).Create(d.conn, d.ctx, d.txOptions)
}

func (d *DBPGTools) Transaction(queries []string) (int, bool) {
	if d.conn == nil {
		return -1, false
	}

	sqlTx, err := d.conn.BeginTx(d.ctx, &d.txOptions)
	if err != nil {
		logger.ErrorJ(d.lc, fmt.Sprint("Failed start transaction: ", err.Error()))
		return -1, false
	}

	for step, query := range queries {
		_, err := sqlTx.ExecContext(d.ctx, query)
		if err != nil {
			logger.ErrorJ(d.lc, fmt.Sprint("Failed to exec query: ", err.Error()))
			if err := sqlTx.Rollback(); err != nil {
				logger.ErrorJ(d.lc, fmt.Sprint("Failed to rollback transcation: ", err.Error()))
			}

			return step, false
		}
	}

	if err := sqlTx.Commit(); err != nil {
		logger.ErrorJ(d.lc, fmt.Sprint("Failed to commit transcation: ", err.Error()))
		return len(queries) - 1, false
	}

	return len(queries) - 1, true
}

func (d *DBPGTools) Query(query string) ([]map[string]interface{}, bool) {
	logger.DebugJ(d.lc, fmt.Sprint("Query: ", query))

	if d.conn == nil {
		logger.ErrorJ(d.lc, "Connection is down")
		return nil, false
	}

	sqlRows, err := d.conn.QueryContext(d.ctx, query)
	if err != nil {
		logger.ErrorJ(d.lc, fmt.Sprint("Failed to execute query: ", err.Error()))
		return nil, false
	}

	return d.makeResult(sqlRows)
}

func (d *DBPGTools) Exec(query string) (sql.Result, bool) {
	logger.DebugJ(d.lc, fmt.Sprint("Executing query: ", query))

	if d.conn == nil {
		logger.ErrorJ(d.lc, "Connection is down")
		return nil, false
	}

	sqlResult, err := d.conn.ExecContext(d.ctx, query)
	if err != nil {
		logger.ErrorJ(d.lc, fmt.Sprint("Failed to execute query: ", err.Error()))
		return nil, false
	}

	return sqlResult, true
}

func (d *DBPGTools) makeResult(sqlRows *sql.Rows) ([]map[string]interface{}, bool) {
	result := []map[string]interface{}{}

	columns, err := sqlRows.Columns()
	if err != nil {
		logger.ErrorJ(d.lc, fmt.Sprint("Failed to get columns from SQL result: ", err.Error()))
		return nil, false
	}

	count := len(columns)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)

	for sqlRows.Next() {
		for index := range columns {
			valuePtrs[index] = &values[index]
		}

		if err := sqlRows.Scan(valuePtrs...); err != nil {
			logger.WarningJ(d.lc, fmt.Sprint("Failed to scan values from SQL result's row: ", err.Error()))
			continue
		}

		rowRes := map[string]interface{}{}

		for i, col := range columns {
			val := values[i]

			b, ok := val.([]byte)
			var v interface{}
			if ok {
				v = string(b)
			} else {
				v = val
			}

			rowRes[col] = v
		}

		result = append(result, rowRes)
	}

	return result, true
}
