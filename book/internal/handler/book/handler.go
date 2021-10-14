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

type Book struct {
	Name   string `json:"name" db:"name"`
	Author string `json:"author" db:"author"`
	Genre  string `json:"books_genre" db:"books_genre"`
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

	query := psql.Select("b.name", "concat_ws(' ', a.name, a.surname) author", "g.name books_genre").From("books b").InnerJoin("genres g ON (b.genre_id = g.id)").InnerJoin("authors a ON (b.author_id = a.id)").Where(sq.Eq{"b.book_uid": bookUUID}, bookUUID)
	q, args, err := query.ToSql()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	var b Book
	if err = h.db.GetContext(ctx, &b, q, args...); err == pgx.ErrNoRows {
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

	query := psql.Select("b.name name", "concat_ws(' ', a.name, a.surname) author", "g.name books_genre").From("books b").InnerJoin("genres g ON (b.genre_id = g.id)").InnerJoin("authors a ON (b.author_id = a.id)")
	if bookName != "" {
		query = query.Where(sq.Eq{"b.name": bookName}, bookName)
	}
	if bookAuthor != "" {
		query = query.Where(sq.Eq{"concat_ws(' ', a.name, a.surname)": bookAuthor}, bookAuthor)
	}

	q, args, err := query.ToSql()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to convert query to string"))
		return
	}

	var books = make([]Book, 0)
	if err = h.db.SelectContext(ctx, &books, q, args...); err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to run query"))
		return
	}

	common.RespondJSON(ctx, w, http.StatusOK, books)
}

func SubQuery(sb sq.SelectBuilder) sq.Sqlizer {
	sql, params, _ := sb.ToSql()
	return sq.Expr("("+sql+")", params...)
}

func (h *Handler) CreateBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.Wrap(err, "failed to parse body"))
		return
	}

	var b Book
	err = json.Unmarshal(body, &b)
	if err != nil {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.Wrap(err, "failed to unmarshall body"))
		return
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	genreSubQuery := sq.Select("id").From("genres").Where(sq.Eq{"name": b.Genre})
	authorSubQuery := sq.Select("id").From("authors").Where(sq.Eq{"concat(name, surname)": b.Author})
	query := psql.Insert("books").Columns("name", "author_id", "genre_id").Values(b.Name, SubQuery(authorSubQuery), SubQuery(genreSubQuery)).Suffix("RETURNING \"book_uid\"")

	q, args, err := query.ToSql()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to convert query to string"))
		return
	}

	var bookUUID string
	if err = h.db.GetContext(ctx, &bookUUID, q, args...); err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to run query"))
		return
	}

	common.RespondJSON(ctx, w, http.StatusCreated, map[string]string{"bookUid": bookUUID})
}

func (h *Handler) DeleteBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bookUUID := chi.URLParam(r, "bookUid")
	if bookUUID == "" {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.New("empty bookUid"))
		return
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	q, args, err := psql.Delete("books").Where("book_uid=?", bookUUID).ToSql()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to convert query to string"))
		return
	}

	if result, err := h.db.ExecContext(ctx, q, args...); err != nil {
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

	common.Respond(ctx, w, http.StatusOK)
}
