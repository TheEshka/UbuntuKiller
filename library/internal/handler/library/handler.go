package library

import (
	"encoding/json"
	sq "github.com/Masterminds/squirrel"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"io/ioutil"
	"library/internal/common"
	"net/http"
)

type bookLibrary struct {
	Location   string `json:"location" db:"location"`
	AvailableCount   string `json:"available_count" db:"available_count"`
}

type bookReq struct {
	UserUid   string `json:"user_uid"`
	Status	  string `json:"status"`
}
var bookStatuses = [4]string{"new", "used", "bad_condition", "lost"}
func isValidStatus(status string) bool {
	for _, st := range bookStatuses {
		if st == status { return true }
	}
	return false
}

type Handler struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) GetLibraryBookUIDS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	libraryUUID := chi.URLParam(r, "libraryUid")
	if libraryUUID == "" {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.New("empty libraryUid"))
		return
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.Select("lb.book_uid").From("libraryBooks lb").InnerJoin("library l ON (lb.library_id = l.id)").Where("l.library_uid=?", libraryUUID)
	q, args, err := query.ToSql()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	var b = make([]string, 0)
	if err = h.db.SelectContext(ctx, &b, q, args...); err == pgx.ErrNoRows {
		common.Respond(ctx, w, http.StatusNotFound)
		return
	} else if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	common.RespondJSON(ctx, w, http.StatusOK, b)
}

func (h *Handler) ReturnBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	libraryUUID := chi.URLParam(r, "libraryUid")
	bookUUID := chi.URLParam(r, "bookUid")
	if libraryUUID == "" || bookUUID == "" {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.New("empty libraryUid/bookUUID"))
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.Wrap(err, "failed to parse body"))
		return
	}

	var b bookReq
	err = json.Unmarshal(body, &b)
	if err != nil {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.Wrap(err, "bad request body"))
		return
	}

	tx, err := h.db.Beginx()

	delTookQuery := `
DELETE FROM takenBooks
WHERE ctid IN (
    SELECT ctid
    FROM takenBooks
    WHERE library_id = (SELECT id FROM library WHERE library_uid = $1) AND user_uid = $2 AND book_uid = $3
    LIMIT 1
)
`
	if result, err := tx.ExecContext(ctx, delTookQuery, libraryUUID, b.UserUid, bookUUID); err != nil {
		_ = tx.Rollback()
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to run query"))
		return
	} else {
		affected, err := result.RowsAffected()
		if err != nil {
			_ = tx.Rollback()
			common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to get affected rows"))
			return
		}

		if affected != 1 {
			_ = tx.Rollback()
			common.Respond(ctx, w, http.StatusNotFound)
			return
		}
	}

	// Увеличить счетчик книг
	incQuery := `
	UPDATE libraryBooks lb
	SET available_count = available_count + 1
	FROM library l
	WHERE l.id = lb.library_id AND l.library_uid = $1 AND lb.book_uid = $2;
	`
	if result, err := tx.ExecContext(ctx, incQuery, libraryUUID, bookUUID); err != nil {
		_ = tx.Rollback()
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to run query"))
		return
	} else {
		affected, err := result.RowsAffected()
		if err != nil {
			_ = tx.Rollback()
			common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to get affected rows"))
			return
		}

		if affected != 1 {
			_ = tx.Rollback()
			common.Respond(ctx, w, http.StatusNotFound)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to commit transaction"))
		return
	}
	common.Respond(ctx, w, http.StatusNoContent)
}

func (h *Handler) TakeBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	libraryUUID := chi.URLParam(r, "libraryUid")
	bookUUID := chi.URLParam(r, "bookUid")
	if libraryUUID == "" || bookUUID == "" {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.New("empty libraryUid/bookUUID/userUid"))
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.Wrap(err, "failed to parse body"))
		return
	}

	var b bookReq
	err = json.Unmarshal(body, &b)
	if err != nil {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.Wrap(err, "bad request body"))
		return
	}
	if !isValidStatus(b.Status) {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.New("incorrect status for taking book"))
		return
	}

	// проверка что есть книга
	countGetQuery := `
SELECT available_count 
FROM libraryBooks lb
INNER JOIN library l ON l.id = lb.library_id
WHERE l.library_uid = $1 AND lb.book_uid = $2
`
	var counts []int
	if err := h.db.SelectContext(ctx, &counts, countGetQuery, libraryUUID, bookUUID); err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to run query"))
		return
	}
	if len(counts) != 1 || counts[0] == 0 {
		common.RespondError(ctx, w, http.StatusNotAcceptable, errors.New("no available queried book"))
		return
	}

	tx, err := h.db.Beginx()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "fail on starting db transaction"))
	}

	// уменьшение количества книг
	decQuery := `
UPDATE libraryBooks lb
SET available_count = available_count - 1
FROM library l
WHERE l.id = lb.library_id AND l.library_uid = $1 AND lb.book_uid = $2;
`
	if result, err := tx.ExecContext(ctx, decQuery, libraryUUID, bookUUID); err != nil {
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

	// добавляем запись о взятой книге
	tookQuery := `INSERT INTO takenBooks(book_uid, user_uid, library_id, status)
VALUES ($1, $2, (SELECT id FROM library WHERE library_uid = $3), $4)
`
	if result, err := tx.ExecContext(ctx, tookQuery, bookUUID, b.UserUid, libraryUUID, b.Status); err != nil {
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

	err = tx.Commit()
	if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to commit transaction"))
		return
	}
	common.Respond(ctx, w, http.StatusNoContent)
}

func (h *Handler) AddBookToLibrary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	libraryUUID := chi.URLParam(r, "libraryUid")
	bookUUID := chi.URLParam(r, "bookUid")

	if libraryUUID == "" || bookUUID == "" {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.New("empty libraryUid"))
		return
	}

	query := `
INSERT INTO libraryBooks(library_id, book_uid, available_count) VALUES
((SELECT id FROM library WHERE library_uid = $1), $2, 1)
`

	if result, err := h.db.ExecContext(ctx, query, libraryUUID, bookUUID); err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to run query"))
		return
	} else {
		affected, err := result.RowsAffected()
		if err != nil {
			common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to get affected rows"))
			return
		}

		if affected != 1 {
			common.RespondError(ctx, w, http.StatusNotAcceptable, errors.Wrap(err, "book already exist in library"))
			return
		}
	}

	common.Respond(ctx, w, http.StatusCreated)
}

func (h *Handler) DeleteBookFromLibrary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	libraryUUID := chi.URLParam(r, "libraryUid")
	bookUUID := chi.URLParam(r, "bookUid")

	if libraryUUID == "" || bookUUID == "" {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.New("empty libraryUid"))
		return
	}

	query := `
DELETE FROM libraryBooks
WHERE library_id = (SELECT id FROM library WHERE library_uid = $1) AND book_uid = $2
`

	if result, err := h.db.ExecContext(ctx, query, libraryUUID, bookUUID); err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to run query"))
		return
	} else {
		affected, err := result.RowsAffected()
		if err != nil {
			common.RespondError(ctx, w, http.StatusInternalServerError, errors.Wrap(err, "failed to get affected rows"))
			return
		}

		if affected != 1 {
			common.RespondError(ctx, w, http.StatusNotAcceptable, errors.Wrap(err, "book already exist in library"))
			return
		}
	}

	common.Respond(ctx, w, http.StatusNoContent)
}

func (h *Handler) FindBook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bookUid := chi.URLParam(r, "bookUid")
	if bookUid == "" {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.New("empty bookUid"))
		return
	}

	query := `
SELECT l.location, lb.available_count
FROM library l 
INNER JOIN libraryBooks lb ON lb.library_id = l.id
WHERE lb.book_uid = $1
`

	var b = make([]bookLibrary, 0)
	if err := h.db.SelectContext(ctx, &b, query, bookUid); err == pgx.ErrNoRows {
		common.Respond(ctx, w, http.StatusNotFound)
		return
	} else if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	common.RespondJSON(ctx, w, http.StatusOK, b)
}

func (h *Handler) TookBooksList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userUid := chi.URLParam(r, "userUid")
	if userUid == "" {
		common.RespondError(ctx, w, http.StatusBadRequest, errors.New("empty userUid"))
		return
	}

	query := `
SELECT book_uid
FROM takenBooks tb
WHERE tb.user_uid = $1
`

	var b = make([]string, 0)
	if err := h.db.SelectContext(ctx, &b, query, userUid); err == pgx.ErrNoRows {
		common.Respond(ctx, w, http.StatusNotFound)
		return
	} else if err != nil {
		common.RespondError(ctx, w, http.StatusInternalServerError, err)
		return
	}

	common.RespondJSON(ctx, w, http.StatusOK, b)
}
