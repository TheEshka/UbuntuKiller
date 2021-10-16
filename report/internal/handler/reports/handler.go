package reports

import (
	"github.com/jackc/pgx/v4"
	"net/http"
	"report/internal/common"

	"github.com/jmoiron/sqlx"
)

type StatReturns struct {
	UserUid   string `json:"user_uid" db:"user_uid"`
	OnTimeCount int `json:"on_time_count" db:"on_time_count"`
}

type StatGenres struct {
	Genre   string `json:"genre" db:"genre"`
	Count int `json:"count" db:"count"`
}

type Handler struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) ReturnsReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	q := `SELECT user_uid FROM returns WHERE on_time = TRUE GROUP BY user_uid`
	var b []StatReturns
	if err := h.db.GetContext(ctx, &b, q); err == pgx.ErrNoRows {
		common.Respond(ctx, w, http.StatusNotFound)
		return
	} else if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	common.RespondJSON(ctx, w, http.StatusOK, b)
}

func (h *Handler) GenresReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	q := `SELECT genre, COUNT(*) FROM genres GROUP BY genre`
	var b []StatReturns
	if err := h.db.GetContext(ctx, &b, q); err == pgx.ErrNoRows {
		common.Respond(ctx, w, http.StatusNotFound)
		return
	} else if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	common.RespondJSON(ctx, w, http.StatusOK, b)
}