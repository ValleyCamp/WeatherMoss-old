package api

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
)

type ApiHandlers struct {
	db *sql.DB
}

func NewApiHandlers(d *sql.DB) *ApiHandlers {
	return &ApiHandlers{
		db: d,
	}
}

func (a *ApiHandlers) Current(w http.ResponseWriter, r *http.Request) {

}
