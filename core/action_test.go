package core

import (
	"encoding/json"
	"testing"

	"github.com/dnote/actions"
	"github.com/dnote/cli/testutils"
	"github.com/dnote/cli/utils"
	"github.com/pkg/errors"
)

func TestLogActionEditNote(t *testing.T) {
	// Setup
	ctx := testutils.InitEnv("../tmp", "../testutils/fixtures/schema.sql")
	defer testutils.TeardownEnv(ctx)

	db := ctx.DB

	b1UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting css book", db, "INSERT INTO books (uuid, label) VALUES (?, ?)", b1UUID, "js")

	// Execute
	tx, err := db.Begin()
	if err != nil {
		panic(errors.Wrap(err, "beginning a transaction"))
	}

	if err := LogActionEditNote(tx, "f0d0fbb7-31ff-45ae-9f0f-4e429c0c797f", "updated content", 1536168581); err != nil {
		t.Fatalf("Failed to perform %s", err.Error())
	}

	tx.Commit()

	// Test
	var actionCount int
	testutils.MustScan(t, "counting actions", db.QueryRow("SELECT count(*) FROM actions"), &actionCount)

	var action actions.Action
	testutils.MustScan(t, "finding action", db.QueryRow("SELECT uuid, schema, type, timestamp, data FROM actions"),
		&action.UUID, &action.Schema, &action.Type, &action.Timestamp, &action.Data)

	var actionData actions.EditNoteDataV3
	if err := json.Unmarshal(action.Data, &actionData); err != nil {
		panic(errors.Wrap(err, "unmarshalling action data"))
	}

	testutils.AssertEqualf(t, actionCount, 1, "action count mismatch")
	testutils.AssertNotEqual(t, action.UUID, "", "action uuid mismatch")
	testutils.AssertEqual(t, action.Schema, 3, "action schema mismatch")
	testutils.AssertEqual(t, action.Type, actions.ActionEditNote, "action type mismatch")
	testutils.AssertNotEqual(t, action.Timestamp, 0, "action timestamp mismatch")
	testutils.AssertEqual(t, actionData.NoteUUID, "f0d0fbb7-31ff-45ae-9f0f-4e429c0c797f", "action data note_uuid mismatch")
	testutils.AssertEqual(t, actionData.BookUUID, (*string)(nil), "action data book_uuid mismatch")
	testutils.AssertEqual(t, actionData.Public, (*bool)(nil), "action data public mismatch")
	testutils.AssertEqual(t, *actionData.Content, "updated content", "action data content mismatch")
}
