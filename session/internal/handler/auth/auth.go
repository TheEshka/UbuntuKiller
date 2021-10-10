package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/go-chi/jwtauth/v5"
	"github.com/jmoiron/sqlx"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"session/internal/common"
)

type User struct {
	Login    string `db:"login" json:"login"`
	Password string `db:"password" json:",omitempty"`
	Role     string `db:"role" json:"role"`
}

type Handler struct {
	db  *sqlx.DB
	jwt *jwtauth.JWTAuth
}

func New(db *sqlx.DB, jwtAuth *jwtauth.JWTAuth) *Handler {
	return &Handler{
		db: db,
		jwt: jwtAuth,
	}
}

func (h *Handler) Auth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	login, password, ok := r.BasicAuth()
	if !ok {
		common.RespondError(ctx, w, http.StatusForbidden, errors.New("expected login and password for basic auth"))
		return
	}

	var user User
	loginQuery := `
SELECT login, password, role FROM accounts WHERE login = $1 AND password = crypt($2, password)
`

	if err := h.db.GetContext(ctx, &user, loginQuery, login, password); err == sql.ErrNoRows {
		common.Respond(ctx, w, http.StatusUnauthorized)
		return
	} else if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to query user password check"))
		return
	}

	_, tokenString, err := h.jwt.Encode(map[string]interface{}{
		"login": user.Login,
		"role": user.Role,
	})
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to encode jwt token"))
		return
	}

	common.RespondJSON(ctx, w, http.StatusOK, map[string]string{"token": tokenString})
}

func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	_, claims, err := tokenFromContext(ctx)
	if err != nil {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.Wrap(err, "failed to get jwt token"))
		return
	}

	role, ok := claims["role"]
	if !ok {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.New("user role not found"))
		return
	}

	roleString := role.(string)
	if roleString != "admin" {
		common.Respond(ctx, w, http.StatusForbidden)
		return
	}

	getUsersQuery := `
SELECT login, role FROM accounts
`

	var users []User
	if err := h.db.SelectContext(ctx, &users, getUsersQuery); err ==sql.ErrNoRows {
		users = make([]User, 0)
	} else if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to query users"))
		return
	}

	common.RespondJSON(ctx, w, http.StatusOK, users)
}

func tokenFromContext(ctx context.Context) (jwt.Token, map[string]interface{}, error) {
	token, _ := ctx.Value(jwtauth.TokenCtxKey).(jwt.Token)

	var err error
	var claims map[string]interface{}

	if token != nil {
		claims, err = token.AsMap(context.Background())
		if err != nil {
			return token, nil, err
		}
	} else {
		claims = map[string]interface{}{}
	}

	err, _ = ctx.Value(jwtauth.ErrorCtxKey).(error)

	return token, claims, err
}



func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	_, claims, err := tokenFromContext(ctx)
	if err != nil {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.Wrap(err, "failed to get jwt token"))
		return
	}

	role, ok := claims["role"]
	if !ok {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.Wrap(err, "user role not found"))
		return
	}

	roleString := role.(string)
	if roleString != "admin" {
		common.Respond(ctx, w, http.StatusForbidden)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.Wrap(err, "failed to parse body"))
		return
	}

	var user User
	err = json.Unmarshal(body, &user)
	if err != nil {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.Wrap(err, "failed to unmarshall body"))
		return
	}

	createUserQuery := `
INSERT INTO accounts(login, password) VALUES ($1, crypt($2, gen_salt('bf')))
`

	_, err = h.db.ExecContext(ctx, createUserQuery, user.Login, user.Password)
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to query create user"))
		return
	}

	common.Respond(ctx, w, http.StatusCreated)
}

func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	common.Respond(r.Context(), w, http.StatusOK)
}
