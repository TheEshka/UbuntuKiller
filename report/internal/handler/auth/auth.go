package auth

import (
	"github.com/go-chi/jwtauth/v5"
	"github.com/pkg/errors"
	"net/http"
	"report/internal/common"
)

type Handler struct {
	jwt *jwtauth.JWTAuth
}

func New(jwtAuth *jwtauth.JWTAuth) *Handler {
	return &Handler{
		jwt: jwtAuth,
	}
}

func (h *Handler) Auth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, _, ok := r.BasicAuth()
	if !ok {
		common.RespondError(ctx, w, http.StatusForbidden, errors.New("expected login and password for basic auth"))
		return
	}

	_, tokenString, err := h.jwt.Encode(map[string]interface{}{
		"login": user,
	})
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to encode jwt token"))
		return
	}

	common.RespondJSON(ctx, w, http.StatusOK, map[string]string{"token": tokenString})
}
