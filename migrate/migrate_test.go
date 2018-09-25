package migrate

import (
	"testing"

	"github.com/dnote/actions"
	"github.com/dnote/cli/testutils"
	"github.com/dnote/cli/utils"
	"github.com/pkg/errors"
)

func TestMigration1(t *testing.T) {
	// set up
	ctx := testutils.InitEnv("../tmp", "../testutils/fixtures/schema.sql")
	defer testutils.TeardownEnv(ctx)

	db := ctx.DB

	data := testutils.MustMarshalJSON(t, actions.AddBookDataV1{BookName: "js"})
	a1UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a1UUID, 1, "add_book", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.EditNoteDataV1{NoteUUID: "note-1-uuid", FromBook: "js", ToBook: "", Content: "note 1"})
	a2UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a2UUID, 1, "edit_note", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.EditNoteDataV1{NoteUUID: "note-2-uuid", FromBook: "js", ToBook: "", Content: "note 2"})
	a3UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a3UUID, 1, "edit_note", string(data), 1537829463)

	// Execute
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(errors.Wrap(err, "beginning a transaction"))
	}

	err = m1.run(tx)
	if err != nil {
		tx.Rollback()
		t.Fatal(errors.Wrap(err, "failed to run"))
	}

	tx.Commit()

	// Test
	var actionCount int
	testutils.MustScan(t, "counting actions", db.QueryRow("SELECT count(*) FROM actions"), &actionCount)
	testutils.AssertEqual(t, actionCount, 3, "action count mismatch")

	var a1, a2, a3 actions.Action
	testutils.MustScan(t, "getting action 1", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a1UUID),
		&a1.Schema, &a1.Type, &a1.Data, &a1.Timestamp)
	testutils.MustScan(t, "getting action 2", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a2UUID),
		&a2.Schema, &a2.Type, &a2.Data, &a2.Timestamp)
	testutils.MustScan(t, "getting action 3", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a3UUID),
		&a3.Schema, &a3.Type, &a3.Data, &a3.Timestamp)

	var a1Data actions.AddBookDataV1
	var a2Data, a3Data actions.EditNoteDataV3
	testutils.MustUnmarshalJSON(t, a1.Data, &a1Data)
	testutils.MustUnmarshalJSON(t, a2.Data, &a2Data)
	testutils.MustUnmarshalJSON(t, a3.Data, &a3Data)

	testutils.AssertEqual(t, a1.Schema, 1, "a1 schema mismatch")
	testutils.AssertEqual(t, a1.Type, "add_book", "a1 type mismatch")
	testutils.AssertEqual(t, a1.Timestamp, int64(1537829463), "a1 timestamp mismatch")
	testutils.AssertEqual(t, a1Data.BookName, "js", "a1 data book_name mismatch")

	testutils.AssertEqual(t, a2.Schema, 3, "a2 schema mismatch")
	testutils.AssertEqual(t, a2.Type, "edit_note", "a2 type mismatch")
	testutils.AssertEqual(t, a2.Timestamp, int64(1537829463), "a2 timestamp mismatch")
	testutils.AssertEqual(t, a2Data.NoteUUID, "note-1-uuid", "a2 data note_uuid mismatch")
	testutils.AssertEqual(t, a2Data.BookUUID, (*string)(nil), "a2 data book_uuid mismatch")
	testutils.AssertEqual(t, *a2Data.Content, "note 1", "a2 data content mismatch")
	testutils.AssertEqual(t, *a2Data.Public, false, "a2 data public mismatch")

	testutils.AssertEqual(t, a3.Schema, 3, "a3 schema mismatch")
	testutils.AssertEqual(t, a3.Type, "edit_note", "a3 type mismatch")
	testutils.AssertEqual(t, a3.Timestamp, int64(1537829463), "a3 timestamp mismatch")
	testutils.AssertEqual(t, a3Data.NoteUUID, "note-2-uuid", "a3 data note_uuid mismatch")
	testutils.AssertEqual(t, a3Data.BookUUID, (*string)(nil), "a3 data book_uuid mismatch")
	testutils.AssertEqual(t, *a3Data.Content, "note 2", "a3 data content mismatch")
	testutils.AssertEqual(t, *a3Data.Public, false, "a3 data public mismatch")
}

func TestMigration2(t *testing.T) {
	// set up
	ctx := testutils.InitEnv("../tmp", "../testutils/fixtures/schema.sql")
	defer testutils.TeardownEnv(ctx)

	db := ctx.DB

	c1 := "note 1 - v1"
	c2 := "note 1 - v2"
	css := "css"

	b1UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting css book", db, "INSERT INTO books (uuid, label) VALUES (?, ?)", b1UUID, "css")

	data := testutils.MustMarshalJSON(t, actions.AddNoteDataV2{NoteUUID: "note-1-uuid", BookName: "js", Content: "note 1", Public: false})
	a1UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a1UUID, 2, "add_note", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.EditNoteDataV2{NoteUUID: "note-1-uuid", FromBook: "js", ToBook: nil, Content: &c1, Public: nil})
	a2UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a2UUID, 2, "edit_note", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.EditNoteDataV2{NoteUUID: "note-1-uuid", FromBook: "js", ToBook: &css, Content: &c2, Public: nil})
	a3UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a3UUID, 2, "edit_note", string(data), 1537829463)

	// Execute
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(errors.Wrap(err, "beginning a transaction"))
	}

	err = m2.run(tx)
	if err != nil {
		tx.Rollback()
		t.Fatal(errors.Wrap(err, "failed to run"))
	}

	tx.Commit()

	// Test
	var actionCount int
	testutils.MustScan(t, "counting actions", db.QueryRow("SELECT count(*) FROM actions"), &actionCount)
	testutils.AssertEqual(t, actionCount, 3, "action count mismatch")

	var a1, a2, a3 actions.Action
	testutils.MustScan(t, "getting action 1", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a1UUID),
		&a1.Schema, &a1.Type, &a1.Data, &a1.Timestamp)
	testutils.MustScan(t, "getting action 2", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a2UUID),
		&a2.Schema, &a2.Type, &a2.Data, &a2.Timestamp)
	testutils.MustScan(t, "getting action 3", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a3UUID),
		&a3.Schema, &a3.Type, &a3.Data, &a3.Timestamp)

	var a1Data actions.AddNoteDataV2
	var a2Data, a3Data actions.EditNoteDataV3
	testutils.MustUnmarshalJSON(t, a1.Data, &a1Data)
	testutils.MustUnmarshalJSON(t, a2.Data, &a2Data)
	testutils.MustUnmarshalJSON(t, a3.Data, &a3Data)

	testutils.AssertEqual(t, a1.Schema, 2, "a1 schema mismatch")
	testutils.AssertEqual(t, a1.Type, "add_note", "a1 type mismatch")
	testutils.AssertEqual(t, a1.Timestamp, int64(1537829463), "a1 timestamp mismatch")
	testutils.AssertEqual(t, a1Data.NoteUUID, "note-1-uuid", "a1 data note_uuid mismatch")
	testutils.AssertEqual(t, a1Data.BookName, "js", "a1 data book_name mismatch")
	testutils.AssertEqual(t, a1Data.Public, false, "a1 data public mismatch")

	testutils.AssertEqual(t, a2.Schema, 3, "a2 schema mismatch")
	testutils.AssertEqual(t, a2.Type, "edit_note", "a2 type mismatch")
	testutils.AssertEqual(t, a2.Timestamp, int64(1537829463), "a2 timestamp mismatch")
	testutils.AssertEqual(t, a2Data.NoteUUID, "note-1-uuid", "a2 data note_uuid mismatch")
	testutils.AssertEqual(t, a2Data.BookUUID, (*string)(nil), "a2 data book_uuid mismatch")
	testutils.AssertEqual(t, *a2Data.Content, c1, "a2 data content mismatch")
	testutils.AssertEqual(t, a2Data.Public, (*bool)(nil), "a2 data public mismatch")

	testutils.AssertEqual(t, a3.Schema, 3, "a3 schema mismatch")
	testutils.AssertEqual(t, a3.Type, "edit_note", "a3 type mismatch")
	testutils.AssertEqual(t, a3.Timestamp, int64(1537829463), "a3 timestamp mismatch")
	testutils.AssertEqual(t, a3Data.NoteUUID, "note-1-uuid", "a3 data note_uuid mismatch")
	testutils.AssertEqual(t, *a3Data.BookUUID, b1UUID, "a3 data book_uuid mismatch")
	testutils.AssertEqual(t, *a3Data.Content, c2, "a3 data content mismatch")
	testutils.AssertEqual(t, a3Data.Public, (*bool)(nil), "a3 data public mismatch")
}

func TestMigration3(t *testing.T) {
	// set up
	ctx := testutils.InitEnv("../tmp", "../testutils/fixtures/schema.sql")
	defer testutils.TeardownEnv(ctx)

	db := ctx.DB

	c1 := "note 2 - v1"

	b1UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting js book", db, "INSERT INTO books (uuid, label) VALUES (?, ?)", b1UUID, "js")
	b2UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting css book", db, "INSERT INTO books (uuid, label) VALUES (?, ?)", b2UUID, "css")

	data := testutils.MustMarshalJSON(t, actions.AddNoteDataV2{NoteUUID: "note-1-uuid", BookName: "js", Content: "note 1", Public: false})
	a1UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a1UUID, 2, "add_note", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.AddNoteDataV2{NoteUUID: "note-2-uuid", BookName: "js", Content: "note 2", Public: false})
	a2UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a2UUID, 2, "add_note", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.EditNoteDataV3{NoteUUID: "note-1-uuid", BookUUID: &b2UUID, Content: &c1, Public: nil})
	a3UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a3UUID, 3, "edit_note", string(data), 1537829463)

	// Execute
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(errors.Wrap(err, "beginning a transaction"))
	}

	err = m3.run(tx)
	if err != nil {
		tx.Rollback()
		t.Fatal(errors.Wrap(err, "failed to run"))
	}

	tx.Commit()

	// Test
	var actionCount int
	testutils.MustScan(t, "counting actions", db.QueryRow("SELECT count(*) FROM actions"), &actionCount)
	testutils.AssertEqual(t, actionCount, 3, "action count mismatch")

	var a1, a2, a3 actions.Action
	testutils.MustScan(t, "getting action 1", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a1UUID),
		&a1.Schema, &a1.Type, &a1.Data, &a1.Timestamp)
	testutils.MustScan(t, "getting action 2", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a2UUID),
		&a2.Schema, &a2.Type, &a2.Data, &a2.Timestamp)
	testutils.MustScan(t, "getting action 3", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a3UUID),
		&a3.Schema, &a3.Type, &a3.Data, &a3.Timestamp)

	var a1Data, a2Data actions.AddNoteDataV3
	var a3Data actions.EditNoteDataV3
	testutils.MustUnmarshalJSON(t, a1.Data, &a1Data)
	testutils.MustUnmarshalJSON(t, a2.Data, &a2Data)
	testutils.MustUnmarshalJSON(t, a3.Data, &a3Data)

	testutils.AssertEqual(t, a1.Schema, 3, "a1 schema mismatch")
	testutils.AssertEqual(t, a1.Type, "add_note", "a1 type mismatch")
	testutils.AssertEqual(t, a1.Timestamp, int64(1537829463), "a1 timestamp mismatch")
	testutils.AssertEqual(t, a1Data.NoteUUID, "note-1-uuid", "a1 data note_uuid mismatch")
	testutils.AssertEqual(t, a1Data.BookUUID, b1UUID, "a1 data book_uuid mismatch")
	testutils.AssertEqual(t, a1Data.Content, "note 1", "a1 data content mismatch")
	testutils.AssertEqual(t, a1Data.Public, false, "a1 data public mismatch")

	testutils.AssertEqual(t, a2.Schema, 3, "a2 schema mismatch")
	testutils.AssertEqual(t, a2.Type, "add_note", "a2 type mismatch")
	testutils.AssertEqual(t, a2.Timestamp, int64(1537829463), "a2 timestamp mismatch")
	testutils.AssertEqual(t, a2Data.NoteUUID, "note-2-uuid", "a2 data note_uuid mismatch")
	testutils.AssertEqual(t, a2Data.BookUUID, b1UUID, "a2 data book_uuid mismatch")
	testutils.AssertEqual(t, a2Data.Content, "note 2", "a2 data content mismatch")
	testutils.AssertEqual(t, a2Data.Public, false, "a2 data public mismatch")

	testutils.AssertEqual(t, a3.Schema, 3, "a3 schema mismatch")
	testutils.AssertEqual(t, a3.Type, "edit_note", "a3 type mismatch")
	testutils.AssertEqual(t, a3.Timestamp, int64(1537829463), "a3 timestamp mismatch")
	testutils.AssertEqual(t, a3Data.NoteUUID, "note-1-uuid", "a3 data note_uuid mismatch")
	testutils.AssertEqual(t, *a3Data.BookUUID, b2UUID, "a3 data book_uuid mismatch")
	testutils.AssertEqual(t, *a3Data.Content, c1, "a3 data content mismatch")
	testutils.AssertEqual(t, a3Data.Public, (*bool)(nil), "a3 data public mismatch")
}

func TestMigration4(t *testing.T) {
	// set up
	ctx := testutils.InitEnv("../tmp", "../testutils/fixtures/schema.sql")
	defer testutils.TeardownEnv(ctx)

	db := ctx.DB

	data := testutils.MustMarshalJSON(t, actions.AddNoteDataV3{NoteUUID: "note-1-uuid", BookUUID: "book-1-uuid", Content: "note 1", Public: false})
	a1UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a1UUID, 3, "add_note", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.RemoveNoteDataV1{NoteUUID: "note-1-uuid", BookName: "js"})
	a2UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a2UUID, 1, "remove_note", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.RemoveNoteDataV1{NoteUUID: "note-2-uuid", BookName: "js"})
	a3UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a3UUID, 1, "remove_note", string(data), 1537829463)

	// Execute
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(errors.Wrap(err, "beginning a transaction"))
	}

	err = m4.run(tx)
	if err != nil {
		tx.Rollback()
		t.Fatal(errors.Wrap(err, "failed to run"))
	}

	tx.Commit()

	// Test
	var actionCount int
	testutils.MustScan(t, "counting actions", db.QueryRow("SELECT count(*) FROM actions"), &actionCount)
	testutils.AssertEqual(t, actionCount, 3, "action count mismatch")

	var a1, a2, a3 actions.Action
	testutils.MustScan(t, "getting action 1", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a1UUID),
		&a1.Schema, &a1.Type, &a1.Data, &a1.Timestamp)
	testutils.MustScan(t, "getting action 2", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a2UUID),
		&a2.Schema, &a2.Type, &a2.Data, &a2.Timestamp)
	testutils.MustScan(t, "getting action 3", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a3UUID),
		&a3.Schema, &a3.Type, &a3.Data, &a3.Timestamp)

	var a1Data actions.AddNoteDataV3
	var a2Data, a3Data actions.RemoveNoteDataV2
	testutils.MustUnmarshalJSON(t, a1.Data, &a1Data)
	testutils.MustUnmarshalJSON(t, a2.Data, &a2Data)
	testutils.MustUnmarshalJSON(t, a3.Data, &a3Data)

	testutils.AssertEqual(t, a1.Schema, 3, "a1 schema mismatch")
	testutils.AssertEqual(t, a1.Type, "add_note", "a1 type mismatch")
	testutils.AssertEqual(t, a1.Timestamp, int64(1537829463), "a1 timestamp mismatch")
	testutils.AssertEqual(t, a1Data.NoteUUID, "note-1-uuid", "a1 data note_uuid mismatch")
	testutils.AssertEqual(t, a1Data.BookUUID, "book-1-uuid", "a1 data book_uuid mismatch")
	testutils.AssertEqual(t, a1Data.Content, "note 1", "a1 data content mismatch")
	testutils.AssertEqual(t, a1Data.Public, false, "a1 data public mismatch")

	testutils.AssertEqual(t, a2.Schema, 2, "a2 schema mismatch")
	testutils.AssertEqual(t, a2.Type, "remove_note", "a2 type mismatch")
	testutils.AssertEqual(t, a2.Timestamp, int64(1537829463), "a2 timestamp mismatch")
	testutils.AssertEqual(t, a2Data.NoteUUID, "note-1-uuid", "a2 data note_uuid mismatch")

	testutils.AssertEqual(t, a3.Schema, 2, "a3 schema mismatch")
	testutils.AssertEqual(t, a3.Type, "remove_note", "a3 type mismatch")
	testutils.AssertEqual(t, a3.Timestamp, int64(1537829463), "a3 timestamp mismatch")
	testutils.AssertEqual(t, a3Data.NoteUUID, "note-2-uuid", "a3 data note_uuid mismatch")
}
