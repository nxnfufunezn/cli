package migrate

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/dnote/actions"
	"github.com/dnote/cli/core"
	"github.com/dnote/cli/infra"
	"github.com/pkg/errors"
)

type migration struct {
	name string
	run  func(ctx infra.DnoteCtx, tx *sql.Tx) error
}

var lm1 = migration{
	name: "upgrade-edit-note-from-v1-to-v3",
	run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
		rows, err := tx.Query("SELECT uuid, data FROM actions WHERE type = ? AND schema = ?", "edit_note", 1)
		if err != nil {
			return errors.Wrap(err, "querying rows")
		}
		defer rows.Close()

		f := false

		for rows.Next() {
			var uuid, dat string

			err = rows.Scan(&uuid, &dat)
			if err != nil {
				return errors.Wrap(err, "scanning a row")
			}

			var oldData actions.EditNoteDataV1
			err = json.Unmarshal([]byte(dat), &oldData)
			if err != nil {
				return errors.Wrap(err, "unmarshalling existing data")
			}

			newData := actions.EditNoteDataV3{
				NoteUUID: oldData.NoteUUID,
				Content:  &oldData.Content,
				// With edit_note v1, CLI did not support changing books or public
				BookUUID: nil,
				Public:   &f,
			}

			b, err := json.Marshal(newData)
			if err != nil {
				return errors.Wrap(err, "marshalling new data")
			}

			_, err = tx.Exec("UPDATE actions SET data = ?, schema = ? WHERE uuid = ?", string(b), 3, uuid)
			if err != nil {
				return errors.Wrap(err, "updating a row")
			}
		}

		return nil
	},
}

var lm2 = migration{
	name: "upgrade-edit-note-from-v2-to-v3",
	run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
		rows, err := tx.Query("SELECT uuid, data FROM actions WHERE type = ? AND schema = ?", "edit_note", 2)
		if err != nil {
			return errors.Wrap(err, "querying rows")
		}
		defer rows.Close()

		for rows.Next() {
			var uuid, dat string

			err = rows.Scan(&uuid, &dat)
			if err != nil {
				return errors.Wrap(err, "scanning a row")
			}

			var oldData actions.EditNoteDataV2
			err = json.Unmarshal([]byte(dat), &oldData)
			if err != nil {
				return errors.Wrap(err, "unmarshalling existing data")
			}

			var bookUUID *string
			if oldData.ToBook != nil {
				var dst string
				err = tx.QueryRow("SELECT uuid FROM books WHERE label = ?", *oldData.ToBook).Scan(&dst)
				if err != nil {
					return errors.Wrap(err, "scanning book uuid")
				}

				bookUUID = &dst
			}

			newData := actions.EditNoteDataV3{
				NoteUUID: oldData.NoteUUID,
				BookUUID: bookUUID,
				Content:  oldData.Content,
				Public:   oldData.Public,
			}

			b, err := json.Marshal(newData)
			if err != nil {
				return errors.Wrap(err, "marshalling new data")
			}

			_, err = tx.Exec("UPDATE actions SET data = ?, schema = ? WHERE uuid = ?", string(b), 3, uuid)
			if err != nil {
				return errors.Wrap(err, "updating a row")
			}
		}

		return nil
	},
}

var lm3 = migration{
	name: "upgrade-add-note-from-v2-to-v3",
	run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
		rows, err := tx.Query("SELECT uuid, data FROM actions WHERE type = ? AND schema = ?", "add_note", 2)
		if err != nil {
			return errors.Wrap(err, "querying rows")
		}
		defer rows.Close()

		for rows.Next() {
			var uuid, dat string

			err = rows.Scan(&uuid, &dat)
			if err != nil {
				return errors.Wrap(err, "scanning a row")
			}

			var oldData actions.AddNoteDataV2
			err = json.Unmarshal([]byte(dat), &oldData)
			if err != nil {
				return errors.Wrap(err, "unmarshalling existing data")
			}

			var bookUUID string
			err = tx.QueryRow("SELECT uuid FROM books WHERE label = ?", oldData.BookName).Scan(&bookUUID)
			if err != nil {
				return errors.Wrap(err, "scanning book uuid")
			}

			newData := actions.AddNoteDataV3{
				NoteUUID: oldData.NoteUUID,
				BookUUID: bookUUID,
				Content:  oldData.Content,
				Public:   oldData.Public,
			}

			b, err := json.Marshal(newData)
			if err != nil {
				return errors.Wrap(err, "marshalling new data")
			}

			_, err = tx.Exec("UPDATE actions SET data = ?, schema = ? WHERE uuid = ?", string(b), 3, uuid)
			if err != nil {
				return errors.Wrap(err, "updating a row")
			}
		}

		return nil
	},
}

var lm4 = migration{
	name: "upgrade-remove-note-from-v1-to-v2",
	run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
		rows, err := tx.Query("SELECT uuid, data FROM actions WHERE type = ? AND schema = ?", "remove_note", 1)
		if err != nil {
			return errors.Wrap(err, "querying rows")
		}
		defer rows.Close()

		for rows.Next() {
			var uuid, dat string

			err = rows.Scan(&uuid, &dat)
			if err != nil {
				return errors.Wrap(err, "scanning a row")
			}

			var oldData actions.RemoveNoteDataV1
			err = json.Unmarshal([]byte(dat), &oldData)
			if err != nil {
				return errors.Wrap(err, "unmarshalling existing data")
			}

			newData := actions.RemoveNoteDataV2{
				NoteUUID: oldData.NoteUUID,
			}

			b, err := json.Marshal(newData)
			if err != nil {
				return errors.Wrap(err, "marshalling new data")
			}

			_, err = tx.Exec("UPDATE actions SET data = ?, schema = ? WHERE uuid = ?", string(b), 2, uuid)
			if err != nil {
				return errors.Wrap(err, "updating a row")
			}
		}

		return nil
	},
}

var lm5 = migration{
	name: "upgrade-add-book-from-v1-to-v2",
	run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
		rows, err := tx.Query("SELECT uuid, data FROM actions WHERE type = ? AND schema = ?", "add_book", 1)
		if err != nil {
			return errors.Wrap(err, "querying rows")
		}
		defer rows.Close()

		for rows.Next() {
			var uuid, dat string

			err = rows.Scan(&uuid, &dat)
			if err != nil {
				return errors.Wrap(err, "scanning a row")
			}

			var oldData actions.AddBookDataV1
			err = json.Unmarshal([]byte(dat), &oldData)
			if err != nil {
				return errors.Wrap(err, "unmarshalling existing data")
			}

			var bookUUID string
			err = tx.QueryRow("SELECT uuid FROM books WHERE label = ?", oldData.BookName).Scan(&bookUUID)
			if err != nil {
				return errors.Wrap(err, "scanning book uuid")
			}

			newData := actions.AddBookDataV2{
				BookName: oldData.BookName,
				BookUUID: bookUUID,
			}

			b, err := json.Marshal(newData)
			if err != nil {
				return errors.Wrap(err, "marshalling new data")
			}

			_, err = tx.Exec("UPDATE actions SET data = ?, schema = ? WHERE uuid = ?", string(b), 2, uuid)
			if err != nil {
				return errors.Wrap(err, "updating a row")
			}
		}

		return nil
	},
}

var lm6 = migration{
	name: "upgrade-remove-book-from-v1-to-v2",
	run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
		rows, err := tx.Query("SELECT uuid, data FROM actions WHERE type = ? AND schema = ?", "remove_book", 1)
		if err != nil {
			return errors.Wrap(err, "querying rows")
		}
		defer rows.Close()

		for rows.Next() {
			var uuid, dat string

			err = rows.Scan(&uuid, &dat)
			if err != nil {
				return errors.Wrap(err, "scanning a row")
			}

			var oldData actions.RemoveBookDataV1
			err = json.Unmarshal([]byte(dat), &oldData)
			if err != nil {
				return errors.Wrap(err, "unmarshalling existing data")
			}

			var bookUUID string
			err = tx.QueryRow("SELECT uuid FROM books WHERE label = ?", oldData.BookName).Scan(&bookUUID)
			if err != nil {
				return errors.Wrap(err, "scanning book uuid")
			}

			newData := actions.RemoveBookDataV2{
				BookUUID: bookUUID,
			}

			b, err := json.Marshal(newData)
			if err != nil {
				return errors.Wrap(err, "marshalling new data")
			}

			_, err = tx.Exec("UPDATE actions SET data = ?, schema = ? WHERE uuid = ?", string(b), 2, uuid)
			if err != nil {
				return errors.Wrap(err, "updating a row")
			}
		}

		return nil
	},
}

func getbookLabelFromUUID(tx *sql.Tx, uuid string) (string, error) {
	var ret string

	err := tx.QueryRow("SELECT label FROM books WHERE uuid = ?", uuid).Scan(&ret)
	if err != nil {
		return ret, errors.Wrap(err, "finding book label")
	}

	return ret, nil
}

func rm1UpdateAddNoteAction(tx *sql.Tx, actionUUID, actionData string, schema int, uuidMap map[string]string) error {
	if schema != 3 {
		return errors.Errorf("unsupported schema '%d' for add_note.", schema)
	}

	var data actions.AddNoteDataV3
	err := json.Unmarshal([]byte(actionData), &data)
	if err != nil {
		return errors.Wrap(err, "unmarshalling action data")
	}

	var bookLabel string
	err = tx.QueryRow("SELECT label FROM books WHERE uuid = ?", actionUUID).Scan(&bookLabel)
	if err != nil {
		return errors.Wrap(err, "finding book label")
	}

	data.BookUUID = uuidMap[bookLabel]

	b, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "marshalling action data")
	}

	_, err = tx.Exec("UPDATE actions SET data = ? WHERE uuid = ?", string(b), actionUUID)
	if err != nil {
		return errors.Wrap(err, "updating action")
	}

	return nil
}

func rm1UpdateEditNoteAction(tx *sql.Tx, actionUUID, actionData string, schema int, uuidMap map[string]string) error {
	if schema != 3 {
		return errors.Errorf("unsupported schema '%d' for edit_note.", schema)
	}

	var data actions.EditNoteDataV3
	err := json.Unmarshal([]byte(actionData), &data)
	if err != nil {
		return errors.Wrap(err, "unmarshalling action data")
	}

	if data.BookUUID == nil {
		var bookLabel string
		err := tx.QueryRow("SELECT label FROM books WHERE uuid = ?", actionUUID).Scan(&bookLabel)
		if err != nil {
			return errors.Wrap(err, "finding book label")
		}
		if err != nil {
			return errors.Wrap(err, "finding book label")
		}

		bookUUID := uuidMap[bookLabel]
		data.BookUUID = &bookUUID

		b, err := json.Marshal(data)
		if err != nil {
			return errors.Wrap(err, "marshalling action data")
		}

		_, err = tx.Exec("UPDATE actions SET data = ? WHERE uuid = ?", string(b), actionUUID)
		if err != nil {
			return errors.Wrap(err, "updating action")
		}
	}

	return nil
}

func rm1UpdateRemoveBookAction(tx *sql.Tx, actionUUID, actionData string, schema int, uuidMap map[string]string) error {
	if schema != 2 {
		return errors.Errorf("unsupported schema '%d' for remove_book", schema)
	}

	var data actions.RemoveBookDataV2
	err := json.Unmarshal([]byte(actionData), &data)
	if err != nil {
		return errors.Wrap(err, "unmarshalling action data")
	}

	var bookLabel string
	err = tx.QueryRow("SELECT label FROM books WHERE uuid = ?", actionUUID).Scan(&bookLabel)
	if err != nil {
		return errors.Wrap(err, "finding book label")
	}
	bookUUID := uuidMap[bookLabel]
	data.BookUUID = bookUUID

	b, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "marshalling action data")
	}

	_, err = tx.Exec("UPDATE actions SET data = ? WHERE uuid = ?", string(b), actionUUID)
	if err != nil {
		return errors.Wrap(err, "updating action")
	}

	return nil
}

var rm1 = migration{
	name: "sync-book-uuids-from-server",
	run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
		endpoint := fmt.Sprintf("%s/v1/books", ctx.APIEndpoint)
		req, err := http.NewRequest("GET", endpoint, strings.NewReader(""))
		if err != nil {
			return errors.Wrap(err, "constructing http request")
		}

		config, err := core.ReadConfig(ctx)
		if err != nil {
			return errors.Wrap(err, "reading the config")
		}
		if config.APIKey == "" {
			return errors.New("login required")
		}

		req.Header.Set("Authorization", config.APIKey)
		req.Header.Set("CLI-Version", ctx.Version)

		hc := http.Client{}
		res, err := hc.Do(req)
		if err != nil {
			return errors.Wrap(err, "making http request")
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return errors.Wrap(err, "reading the response body")
		}

		resData := []struct {
			UUID  string `json:"uuid"`
			Label string `json:"label"`
		}{}
		if err = json.Unmarshal(body, &resData); err != nil {
			return errors.Wrap(err, "unmarshalling the payload")
		}

		UUIDMap := map[string]string{}

		for _, book := range resData {
			// 1. Build a map from uuid to label
			UUIDMap[book.Label] = book.UUID

			// 2. update uuid in the books table
			_, err := tx.Exec("UPDATE books SET uuid = ? WHERE label = ?", book.UUID, book.Label)
			if err != nil {
				return errors.Wrap(err, "updating book")
			}
		}

		rows, err := tx.Query("SELECT uuid, schema, type, data FROM actions")
		if err != nil {
			return errors.Wrap(err, "querying actions")
		}
		defer rows.Close()

		// 3. transform actions
		for rows.Next() {
			var schema int
			var actionUUID, actionType, actionData string

			err = rows.Scan(&actionUUID, &schema, &actionType, &actionData)
			if err != nil {
				return errors.Wrap(err, "scanning a row")
			}

			switch actionType {
			case actions.ActionAddNote:
				err = rm1UpdateAddNoteAction(tx, actionUUID, actionData, schema, UUIDMap)
			case actions.ActionEditNote:
				err = rm1UpdateEditNoteAction(tx, actionUUID, actionData, schema, UUIDMap)
			case actions.ActionRemoveBook:
				err = rm1UpdateRemoveBookAction(tx, actionUUID, actionData, schema, UUIDMap)
			}

			if err != nil {
				return errors.Wrapf(err, "updatng action %s uuid %s", actionType, actionUUID)
			}
		}

		return nil
	},
}
