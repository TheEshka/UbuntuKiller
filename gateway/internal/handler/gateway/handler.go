package gateway

import (
	//"github.com/go-chi/chi/v5"
	//"github.com/jackc/pgx/v4"
	//"github.com/pkg/errors"
	//"net/http"
	//"gateway/internal/common"
	"net/http"
)

type Handler struct {

}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) GetUserRating(w http.ResponseWriter, r *http.Request) {
	//ctx := r.Context()
	//
	//userUid := chi.URLParam(r, "userUid")
	//
	//q := "SELECT rate FROM ratings WHERE user_uid = $1"
	//var b int
	//if err := h.db.GetContext(ctx, &b, q, userUid); err == pgx.ErrNoRows {
	//	common.Respond(ctx, w, http.StatusNotFound)
	//	return
	//} else if err != nil {
	//	common.RespondError(ctx, w, http.StatusInternalServerError, err)
	//	return
	//}
	//
	//common.RespondJSON(ctx, w, http.StatusOK, b)
}
//
//func (h *Handler) IncUserRate(w http.ResponseWriter, r *http.Request) {
//	ctx := r.Context()
//
//	userUid := chi.URLParam(r, "userUid")
//
//	// Увеличить счетчик книг
//	incQuery := "UPDATE ratings SET rate = rate + 1 WHERE user_uid = $1;"
//	if result, err := h.db.ExecContext(ctx, incQuery, userUid); err != nil {
//		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to run query"))
//		return
//	} else {
//		affected, err := result.RowsAffected()
//		if err != nil {
//			common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to get affected rows"))
//			return
//		}
//
//		if affected != 1 {
//			common.Respond(ctx, w, http.StatusNotFound)
//			return
//		}
//	}
//
//	common.Respond(ctx, w, http.StatusNoContent)
//}
//
//func (h *Handler) DecUserRate(w http.ResponseWriter, r *http.Request) {
//	ctx := r.Context()
//
//	userUid := chi.URLParam(r, "userUid")
//
//	// Увеличить счетчик книг
//	incQuery := "UPDATE ratings SET rate = rate - 1 WHERE user_uid = $1;"
//	if result, err := h.db.ExecContext(ctx, incQuery, userUid); err != nil {
//		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to run query"))
//		return
//	} else {
//		affected, err := result.RowsAffected()
//		if err != nil {
//			common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to get affected rows"))
//			return
//		}
//
//		if affected != 1 {
//			common.Respond(ctx, w, http.StatusNotFound)
//			return
//		}
//	}
//
//	common.Respond(ctx, w, http.StatusNoContent)
//}
