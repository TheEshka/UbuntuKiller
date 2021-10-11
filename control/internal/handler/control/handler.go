package control

import (
	"control/internal/common"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"net/http"
)

type Handler struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) IncUserCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userUid := chi.URLParam(r, "userUid")

	q := "SELECT limit_count - current_count FROM control WHERE user_uid = $1"
	var l int
	if err := h.db.GetContext(ctx, &l, q, userUid); err == pgx.ErrNoRows {
		common.Respond(ctx, w, http.StatusNotFound)
		return
	} else if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	if l <= 0 {
		common.RespondError(ctx, w, http.StatusNotAcceptable, errors.New("user reached the limit"))
		return
	}

	// Увеличить счетчик книг
	incQuery := "UPDATE control SET current_count = current_count + 1 WHERE user_uid = $1;"
	if result, err := h.db.ExecContext(ctx, incQuery, userUid); err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to run query"))
		return
	} else {
		affected, err := result.RowsAffected()
		if err != nil {
			common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to get affected rows"))
			return
		}

		if affected != 1 {
			common.Respond(ctx, w, http.StatusNotFound)
			return
		}
	}

	common.Respond(ctx, w, http.StatusNoContent)
}

func (h *Handler) DecUserCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userUid := chi.URLParam(r, "userUid")

	// Увеличить счетчик книг
	incQuery := "UPDATE control SET current_count = current_count - 1 WHERE user_uid = $1;"
	if result, err := h.db.ExecContext(ctx, incQuery, userUid); err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to run query"))
		return
	} else {
		affected, err := result.RowsAffected()
		if err != nil {
			common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to get affected rows"))
			return
		}

		if affected != 1 {
			common.Respond(ctx, w, http.StatusNotFound)
			return
		}
	}

	common.Respond(ctx, w, http.StatusNoContent)
}
