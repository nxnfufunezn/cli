package core

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/dnote/actions"
	"github.com/pkg/errors"
)

// LogActionAddNote logs an action for adding a note
func LogActionAddNote(tx *sql.Tx, noteUUID, bookUUID, content string, timestamp int64) error {
	b, err := json.Marshal(actions.AddNoteDataV3{
		NoteUUID: noteUUID,
		BookUUID: bookUUID,
		Content:  content,
		// TODO: support adding a public note
		Public: false,
	})
	if err != nil {
		return errors.Wrap(err, "marshalling data into JSON")
	}

	if err := LogAction(tx, 3, actions.ActionAddNote, string(b), timestamp); err != nil {
		return errors.Wrapf(err, "logging action")
	}

	return nil
}

// LogActionRemoveNote logs an action for removing a book
func LogActionRemoveNote(tx *sql.Tx, noteUUID string) error {
	b, err := json.Marshal(actions.RemoveNoteDataV2{
		NoteUUID: noteUUID,
	})
	if err != nil {
		return errors.Wrap(err, "marshalling data into JSON")
	}

	ts := time.Now().Unix()
	if err := LogAction(tx, 2, actions.ActionRemoveNote, string(b), ts); err != nil {
		return errors.Wrapf(err, "logging action")
	}

	return nil
}

// LogActionEditNote logs an action for editing a note
func LogActionEditNote(tx *sql.Tx, noteUUID, content string, ts int64) error {
	b, err := json.Marshal(actions.EditNoteDataV3{
		NoteUUID: noteUUID,
		Content:  &content,
	})
	if err != nil {
		return errors.Wrap(err, "marshalling data into JSON")
	}

	if err := LogAction(tx, 3, actions.ActionEditNote, string(b), ts); err != nil {
		return errors.Wrapf(err, "logging action")
	}

	return nil
}

// LogActionAddBook logs an action for adding a book
func LogActionAddBook(tx *sql.Tx, name, uuid string) error {
	b, err := json.Marshal(actions.AddBookDataV2{
		BookName: name,
		BookUUID: uuid,
	})
	if err != nil {
		return errors.Wrap(err, "marshalling data into JSON")
	}

	ts := time.Now().Unix()
	if err := LogAction(tx, 2, actions.ActionAddBook, string(b), ts); err != nil {
		return errors.Wrapf(err, "logging action")
	}

	return nil
}

// LogActionRemoveBook logs an action for removing book
func LogActionRemoveBook(tx *sql.Tx, uuid string) error {
	b, err := json.Marshal(actions.RemoveBookDataV2{BookUUID: uuid})
	if err != nil {
		return errors.Wrap(err, "marshalling data into JSON")
	}

	ts := time.Now().Unix()
	if err := LogAction(tx, 2, actions.ActionRemoveBook, string(b), ts); err != nil {
		return errors.Wrapf(err, "logging action")
	}

	return nil
}
