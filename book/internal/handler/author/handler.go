package author

import (
	"book/internal/common"
	sq "github.com/Masterminds/squirrel"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"net/http"
)

type Book struct {
	BookUID string `json:"bookUid,omitempty" db:"book_uid"`
	Name    string `json:"name" db:"name"`
}

type Author struct {
	Name        string  `json:"name" db:"name"`
	Surname     *string `json:"surname" db:"surname"`
	Description *string `json:"description" db:"description"`
	Books       []Book  `json:"books"`
}

type Handler struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) GetAuthor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	authorUid := chi.URLParam(r, "authorUid")
	if authorUid == "" {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.New("empty authorUid"))
		return
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	q, args, err := psql.Select("a.name, a.surname, a.description").From("authors a").Where(sq.Eq{"id": authorUid}).ToSql()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to convert query to string"))
		return
	}

	var author Author
	if err = h.db.GetContext(ctx, &author, q, args...); err == pgx.ErrNoRows {
		common.Respond(ctx, w, http.StatusNotFound)
		return
	} else if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	common.RespondJSON(ctx, w, http.StatusOK, author)
}

func (h *Handler) GetAuthorBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	authorUid := chi.URLParam(r, "authorUid")
	if authorUid == "" {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.New("empty authorUid"))
		return
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	q, args, err := psql.Select("a.name, a.surname, a.description").From("authors a").Where(sq.Eq{"id": authorUid}).ToSql()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to convert query to string"))
		return
	}

	tx, err := h.db.Beginx()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to begin transaction"))
		return
	}

	var author Author
	if err = tx.GetContext(ctx, &author, q, args...); err == pgx.ErrNoRows {
		common.Respond(ctx, w, http.StatusNotFound)
		return
	} else if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	q, args, err = psql.Select("b.book_uid, b.name").From("books b").Where(sq.Eq{"b.author_id": authorUid}).ToSql()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to convert query to string"))
		return
	}

	var books []Book
	if err = tx.SelectContext(ctx, &books, q, args...); err == pgx.ErrNoRows {
		books = []Book{}
	} else if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to commit transaction"))
		return
	}

	author.Books = books

	common.RespondJSON(ctx, w, http.StatusOK, author)
}
