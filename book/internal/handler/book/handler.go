package book

import (
	"book/internal/common"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type book struct {
	Name   string `json:"name" db:"name"`
	Author string `json:"author" db:"author"`
	Genre  string `json:"genre" db:"genre"`
}

type Handler struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) GetBooksByUUID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bookUUID := chi.URLParam(r, "bookUid")
	if bookUUID == "" {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.New("empty bookUid"))
		return
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.Select("b.name name", "b.author author", "g.name genre").From("books b").InnerJoin("genres g ON (b.genre_id = g.id)").Where("b.book_uid=?", bookUUID)
	q, args, err := query.ToSql()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	var b book
	if err = h.db.GetContext(ctx, &b, q, args); err == pgx.ErrNoRows {
		common.Respond(ctx, w, http.StatusNotFound)
		return
	} else if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	common.RespondJSON(ctx, w, http.StatusOK, b)
}

func (h *Handler) GetBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bookName := r.URL.Query().Get("name")
	bookAuthor := r.URL.Query().Get("author")

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.Select("b.name name", "b.author author", "g.name genre").From("books b").InnerJoin("genres g ON (b.genre_id = g.id)")
	if bookName != "" {
		query = query.Where("b.name=?", bookName)
	}
	if bookAuthor != "" {
		query = query.Where("b.author=?", bookAuthor)
	}

	q, args, err := query.ToSql()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to convert query to string"))
		return
	}

	var books = make([]book, 0)
	if err = h.db.SelectContext(ctx, &books, q, args...); err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to run query"))
		return
	}

	common.RespondJSON(ctx, w, http.StatusOK, books)
}

func SubQuery(sb sq.SelectBuilder) sq.Sqlizer {
	sql, params, _ := sb.ToSql()
	return sq.Expr("("+sql+")", params)
}

func (h *Handler) CreateBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.Wrap(err, "failed to parse body"))
		return
	}

	var b book
	err = json.Unmarshal(body, &b)
	if err != nil {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.Wrap(err, "failed to unmarshall body"))
		return
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	genreSubQuery := psql.Select("id").From("genres").Where("name=?", b.Genre)
	query := psql.Insert("books").Columns("name", "author", "genre_id").Values(b.Name, b.Author, SubQuery(genreSubQuery))

	q, args, err := query.ToSql()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to convert query to string"))
		return
	}

	if _, err = h.db.ExecContext(ctx, q, args...); err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to run query"))
		return
	}

	common.Respond(ctx, w, http.StatusCreated)
}
