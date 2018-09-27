package migrate

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dnote/actions"
	"github.com/dnote/cli/infra"
	"github.com/dnote/cli/testutils"
	"github.com/dnote/cli/utils"
	"github.com/pkg/errors"
)

func TestExecute_bump_schema(t *testing.T) {
	testCases := []struct {
		schemaKey string
	}{
		{
			schemaKey: infra.SystemSchema,
		},
		{
			schemaKey: infra.SystemRemoteSchema,
		},
	}

	for _, tc := range testCases {
		func() {
			// set up
			ctx := testutils.InitEnv("../tmp", "../testutils/fixtures/schema.sql")
			defer testutils.TeardownEnv(ctx)

			db := ctx.DB
			testutils.MustExec(t, "inserting a schema", db, "INSERT INTO system (key, value) VALUES (?, ?)", tc.schemaKey, 8)

			m1 := migration{
				name: "noop",
				run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
					return nil
				},
			}
			m2 := migration{
				name: "noop",
				run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
					return nil
				},
			}

			// execute
			err := execute(ctx, m1, tc.schemaKey)
			if err != nil {
				t.Fatal(errors.Wrap(err, "failed to execute"))
			}
			err = execute(ctx, m2, tc.schemaKey)
			if err != nil {
				t.Fatal(errors.Wrap(err, "failed to execute"))
			}

			// test
			var schema int
			testutils.MustScan(t, "getting schema", db.QueryRow("SELECT value FROM system WHERE key = ?", tc.schemaKey), &schema)
			testutils.AssertEqual(t, schema, 10, "schema was not incremented properly")
		}()
	}
}

func TestRun_nonfresh(t *testing.T) {
	testCases := []struct {
		mode      int
		schemaKey string
	}{
		{
			mode:      LocalMode,
			schemaKey: infra.SystemSchema,
		},
		{
			mode:      RemoteMode,
			schemaKey: infra.SystemRemoteSchema,
		},
	}

	for _, tc := range testCases {
		func() {
			// set up
			ctx := testutils.InitEnv("../tmp", "../testutils/fixtures/schema.sql")
			defer testutils.TeardownEnv(ctx)

			db := ctx.DB
			testutils.MustExec(t, "inserting a schema", db, "INSERT INTO system (key, value) VALUES (?, ?)", tc.schemaKey, 2)
			testutils.MustExec(t, "creating a temporary table for testing", db,
				"CREATE TABLE migrate_run_test ( name string )")

			sequence := []migration{
				migration{
					name: "v1",
					run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
						testutils.MustExec(t, "marking v1 completed", db, "INSERT INTO migrate_run_test (name) VALUES (?)", "v1")
						return nil
					},
				},
				migration{
					name: "v2",
					run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
						testutils.MustExec(t, "marking v2 completed", db, "INSERT INTO migrate_run_test (name) VALUES (?)", "v2")
						return nil
					},
				},
				migration{
					name: "v3",
					run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
						testutils.MustExec(t, "marking v3 completed", db, "INSERT INTO migrate_run_test (name) VALUES (?)", "v3")
						return nil
					},
				},
				migration{
					name: "v4",
					run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
						testutils.MustExec(t, "marking v4 completed", db, "INSERT INTO migrate_run_test (name) VALUES (?)", "v4")
						return nil
					},
				},
			}

			// execute
			err := Run(ctx, sequence, tc.mode)
			if err != nil {
				t.Fatal(errors.Wrap(err, "failed to run"))
			}

			// test
			var schema int
			testutils.MustScan(t, fmt.Sprintf("getting schema for %s", tc.schemaKey), db.QueryRow("SELECT value FROM system WHERE key = ?", tc.schemaKey), &schema)
			testutils.AssertEqual(t, schema, 4, fmt.Sprintf("schema was not updated for %s", tc.schemaKey))

			var testRunCount int
			testutils.MustScan(t, "counting test runs", db.QueryRow("SELECT count(*) FROM migrate_run_test"), &testRunCount)
			testutils.AssertEqual(t, testRunCount, 2, "test run count mismatch")

			var testRun1, testRun2 string
			testutils.MustScan(t, "finding test run 1", db.QueryRow("SELECT name FROM migrate_run_test WHERE name = ?", "v3"), &testRun1)
			testutils.MustScan(t, "finding test run 2", db.QueryRow("SELECT name FROM migrate_run_test WHERE name = ?", "v4"), &testRun2)
		}()
	}

}

func TestRun_fresh(t *testing.T) {
	testCases := []struct {
		mode      int
		schemaKey string
	}{
		{
			mode:      LocalMode,
			schemaKey: infra.SystemSchema,
		},
		{
			mode:      RemoteMode,
			schemaKey: infra.SystemRemoteSchema,
		},
	}

	for _, tc := range testCases {
		func() {
			// set up
			ctx := testutils.InitEnv("../tmp", "../testutils/fixtures/schema.sql")
			defer testutils.TeardownEnv(ctx)

			db := ctx.DB
			testutils.MustExec(t, "creating a temporary table for testing", db,
				"CREATE TABLE migrate_run_test ( name string )")

			sequence := []migration{
				migration{
					name: "v1",
					run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
						testutils.MustExec(t, "marking v1 completed", db, "INSERT INTO migrate_run_test (name) VALUES (?)", "v1")
						return nil
					},
				},
				migration{
					name: "v2",
					run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
						testutils.MustExec(t, "marking v2 completed", db, "INSERT INTO migrate_run_test (name) VALUES (?)", "v2")
						return nil
					},
				},
				migration{
					name: "v3",
					run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
						testutils.MustExec(t, "marking v3 completed", db, "INSERT INTO migrate_run_test (name) VALUES (?)", "v3")
						return nil
					},
				},
			}

			// execute
			err := Run(ctx, sequence, tc.mode)
			if err != nil {
				t.Fatal(errors.Wrap(err, "failed to run"))
			}

			// test
			var schema int
			testutils.MustScan(t, "getting schema", db.QueryRow("SELECT value FROM system WHERE key = ?", tc.schemaKey), &schema)
			testutils.AssertEqual(t, schema, 3, "schema was not updated")

			var testRunCount int
			testutils.MustScan(t, "counting test runs", db.QueryRow("SELECT count(*) FROM migrate_run_test"), &testRunCount)
			testutils.AssertEqual(t, testRunCount, 3, "test run count mismatch")

			var testRun1, testRun2, testRun3 string
			testutils.MustScan(t, "finding test run 1", db.QueryRow("SELECT name FROM migrate_run_test WHERE name = ?", "v1"), &testRun1)
			testutils.MustScan(t, "finding test run 2", db.QueryRow("SELECT name FROM migrate_run_test WHERE name = ?", "v2"), &testRun2)
			testutils.MustScan(t, "finding test run 2", db.QueryRow("SELECT name FROM migrate_run_test WHERE name = ?", "v3"), &testRun3)
		}()
	}
}

func TestRun_up_to_date(t *testing.T) {
	testCases := []struct {
		mode      int
		schemaKey string
	}{
		{
			mode:      LocalMode,
			schemaKey: infra.SystemSchema,
		},
		{
			mode:      RemoteMode,
			schemaKey: infra.SystemRemoteSchema,
		},
	}

	for _, tc := range testCases {
		func() {
			// set up
			ctx := testutils.InitEnv("../tmp", "../testutils/fixtures/schema.sql")
			defer testutils.TeardownEnv(ctx)

			db := ctx.DB
			testutils.MustExec(t, "creating a temporary table for testing", db,
				"CREATE TABLE migrate_run_test ( name string )")

			testutils.MustExec(t, "inserting a schema", db, "INSERT INTO system (key, value) VALUES (?, ?)", tc.schemaKey, 3)

			sequence := []migration{
				migration{
					name: "v1",
					run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
						testutils.MustExec(t, "marking v1 completed", db, "INSERT INTO migrate_run_test (name) VALUES (?)", "v1")
						return nil
					},
				},
				migration{
					name: "v2",
					run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
						testutils.MustExec(t, "marking v2 completed", db, "INSERT INTO migrate_run_test (name) VALUES (?)", "v2")
						return nil
					},
				},
				migration{
					name: "v3",
					run: func(ctx infra.DnoteCtx, tx *sql.Tx) error {
						testutils.MustExec(t, "marking v3 completed", db, "INSERT INTO migrate_run_test (name) VALUES (?)", "v3")
						return nil
					},
				},
			}

			// execute
			err := Run(ctx, sequence, tc.mode)
			if err != nil {
				t.Fatal(errors.Wrap(err, "failed to run"))
			}

			// test
			var schema int
			testutils.MustScan(t, "getting schema", db.QueryRow("SELECT value FROM system WHERE key = ?", tc.schemaKey), &schema)
			testutils.AssertEqual(t, schema, 3, "schema was not updated")

			var testRunCount int
			testutils.MustScan(t, "counting test runs", db.QueryRow("SELECT count(*) FROM migrate_run_test"), &testRunCount)
			testutils.AssertEqual(t, testRunCount, 0, "test run count mismatch")
		}()
	}
}

func TestLocalMigration1(t *testing.T) {
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

	err = lm1.run(ctx, tx)
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

func TestLocalMigration2(t *testing.T) {
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

	err = lm2.run(ctx, tx)
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

func TestLocalMigration3(t *testing.T) {
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

	err = lm3.run(ctx, tx)
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

func TestLocalMigration4(t *testing.T) {
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

	err = lm4.run(ctx, tx)
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

func TestLocalMigration5(t *testing.T) {
	// set up
	ctx := testutils.InitEnv("../tmp", "../testutils/fixtures/schema.sql")
	defer testutils.TeardownEnv(ctx)

	db := ctx.DB

	b1UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting js book", db, "INSERT INTO books (uuid, label) VALUES (?, ?)", b1UUID, "js")
	b2UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting css book", db, "INSERT INTO books (uuid, label) VALUES (?, ?)", b2UUID, "css")

	data := testutils.MustMarshalJSON(t, actions.AddBookDataV1{BookName: "js"})
	a1UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a1UUID, 1, "add_book", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.AddBookDataV1{BookName: "css"})
	a2UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a2UUID, 1, "add_book", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.AddNoteDataV3{NoteUUID: "note-1-uuid", BookUUID: b1UUID, Content: "note 1", Public: false})
	a3UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a3UUID, 3, "add_note", string(data), 1537829463)

	// Execute
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(errors.Wrap(err, "beginning a transaction"))
	}

	err = lm5.run(ctx, tx)
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

	var a1Data, a2Data actions.AddBookDataV2
	var a3Data actions.AddNoteDataV3
	testutils.MustUnmarshalJSON(t, a1.Data, &a1Data)
	testutils.MustUnmarshalJSON(t, a2.Data, &a2Data)
	testutils.MustUnmarshalJSON(t, a3.Data, &a3Data)

	testutils.AssertEqual(t, a1.Schema, 2, "a1 schema mismatch")
	testutils.AssertEqual(t, a1.Type, "add_book", "a1 type mismatch")
	testutils.AssertEqual(t, a1.Timestamp, int64(1537829463), "a1 timestamp mismatch")
	testutils.AssertEqual(t, a1Data.BookName, "js", "a1 data book_name mismatch")
	testutils.AssertEqual(t, a1Data.BookUUID, b1UUID, "a1 data book_uuid mismatch")

	testutils.AssertEqual(t, a2.Schema, 2, "a2 schema mismatch")
	testutils.AssertEqual(t, a2.Type, "add_book", "a2 type mismatch")
	testutils.AssertEqual(t, a2.Timestamp, int64(1537829463), "a2 timestamp mismatch")
	testutils.AssertEqual(t, a2Data.BookName, "css", "a2 data book_name mismatch")
	testutils.AssertEqual(t, a2Data.BookUUID, b2UUID, "a2 data book_uuid mismatch")

	testutils.AssertEqual(t, a3.Schema, 3, "a3 schema mismatch")
	testutils.AssertEqual(t, a3.Type, "add_note", "a3 type mismatch")
	testutils.AssertEqual(t, a3.Timestamp, int64(1537829463), "a3 timestamp mismatch")
	testutils.AssertEqual(t, a3Data.NoteUUID, "note-1-uuid", "a3 data note_uuid mismatch")
	testutils.AssertEqual(t, a3Data.BookUUID, b1UUID, "a3 data book_uuid mismatch")
	testutils.AssertEqual(t, a3Data.Content, "note 1", "a3 data content mismatch")
	testutils.AssertEqual(t, a3Data.Public, false, "a3 data public mismatch")
}

func TestLocalMigration6(t *testing.T) {
	// set up
	ctx := testutils.InitEnv("../tmp", "../testutils/fixtures/schema.sql")
	defer testutils.TeardownEnv(ctx)

	db := ctx.DB

	b1UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting js book", db, "INSERT INTO books (uuid, label) VALUES (?, ?)", b1UUID, "js")
	b2UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting css book", db, "INSERT INTO books (uuid, label) VALUES (?, ?)", b2UUID, "css")

	data := testutils.MustMarshalJSON(t, actions.AddBookDataV2{BookUUID: b1UUID, BookName: "js"})
	a1UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a1UUID, 2, "add_book", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.RemoveBookDataV1{BookName: "js"})
	a2UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a2UUID, 1, "remove_book", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.RemoveBookDataV1{BookName: "css"})
	a3UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a3UUID, 1, "remove_book", string(data), 1537829463)

	// Execute
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(errors.Wrap(err, "beginning a transaction"))
	}

	err = lm6.run(ctx, tx)
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

	var a1Data actions.AddBookDataV2
	var a3Data, a2Data actions.RemoveBookDataV2
	testutils.MustUnmarshalJSON(t, a1.Data, &a1Data)
	testutils.MustUnmarshalJSON(t, a2.Data, &a2Data)
	testutils.MustUnmarshalJSON(t, a3.Data, &a3Data)

	testutils.AssertEqual(t, a1.Schema, 2, "a1 schema mismatch")
	testutils.AssertEqual(t, a1.Type, "add_book", "a1 type mismatch")
	testutils.AssertEqual(t, a1.Timestamp, int64(1537829463), "a1 timestamp mismatch")
	testutils.AssertEqual(t, a1Data.BookName, "js", "a1 data book_name mismatch")
	testutils.AssertEqual(t, a1Data.BookUUID, b1UUID, "a1 data book_uuid mismatch")

	testutils.AssertEqual(t, a2.Schema, 2, "a2 schema mismatch")
	testutils.AssertEqual(t, a2.Type, "remove_book", "a2 type mismatch")
	testutils.AssertEqual(t, a2.Timestamp, int64(1537829463), "a2 timestamp mismatch")
	testutils.AssertEqual(t, a2Data.BookUUID, b1UUID, "a2 data book_uuid mismatch")

	testutils.AssertEqual(t, a3.Schema, 2, "a3 schema mismatch")
	testutils.AssertEqual(t, a3.Type, "remove_book", "a3 type mismatch")
	testutils.AssertEqual(t, a3.Timestamp, int64(1537829463), "a3 timestamp mismatch")
	testutils.AssertEqual(t, a3Data.BookUUID, b2UUID, "a3 data book_uuid mismatch")
}

func TestRemoteMigration1(t *testing.T) {
	// set up
	ctx := testutils.InitEnv("../tmp", "../testutils/fixtures/schema.sql")
	defer testutils.TeardownEnv(ctx)

	JSBookUUID := utils.GenerateUUID()
	CSSBookUUID := utils.GenerateUUID()
	newJSBookUUID := "new-js-book-uuid"
	newCSSBookUUID := "new-css-book-uuid"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/v1/books" {
			res := []struct {
				UUID  string `json:"uuid"`
				Label string `json:"label"`
			}{
				{
					UUID:  newJSBookUUID,
					Label: "js",
				},
				{
					UUID:  newCSSBookUUID,
					Label: "css",
				},
			}

			if err := json.NewEncoder(w).Encode(res); err != nil {
				t.Fatal(errors.Wrap(err, "encoding response"))
			}
		}
	}))
	defer server.Close()

	ctx.APIEndpoint = server.URL

	confStr := fmt.Sprintf("apikey: mock_api_key")
	testutils.WriteFile(ctx, []byte(confStr), "dnoterc")

	db := ctx.DB

	testutils.MustExec(t, "inserting js book", db, "INSERT INTO books (uuid, label) VALUES (?, ?)", JSBookUUID, "js")
	testutils.MustExec(t, "inserting css book", db, "INSERT INTO books (uuid, label) VALUES (?, ?)", CSSBookUUID, "css")

	// TODO: add_book doesn't need to be migrated. simplify this
	data := testutils.MustMarshalJSON(t, actions.AddBookDataV2{BookName: "js", BookUUID: JSBookUUID})
	a1UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a1UUID, 2, "add_book", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.AddBookDataV2{BookName: "css", BookUUID: CSSBookUUID})
	a2UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a2UUID, 2, "add_book", string(data), 1537829463)

	linuxBookUUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting linux book", db, "INSERT INTO books (uuid, label) VALUES (?, ?)", linuxBookUUID, "linux")
	bashBookUUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting bash book", db, "INSERT INTO books (uuid, label) VALUES (?, ?)", bashBookUUID, "bash")

	data = testutils.MustMarshalJSON(t, actions.AddBookDataV2{BookName: "linux", BookUUID: linuxBookUUID})
	a3UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a3UUID, 2, "add_book", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.AddBookDataV2{BookName: "bash", BookUUID: bashBookUUID})
	a4UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a4UUID, 2, "add_book", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.AddNoteDataV3{NoteUUID: "note-1-uuid", BookUUID: JSBookUUID, Content: "note-1", Public: false})
	a5UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a5UUID, 3, "add_note", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.AddNoteDataV3{NoteUUID: "note-2-uuid", BookUUID: JSBookUUID, Content: "note-2", Public: false})
	a6UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a6UUID, 3, "add_note", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.AddNoteDataV3{NoteUUID: "note-3-uuid", BookUUID: CSSBookUUID, Content: "note-3", Public: false})
	a7UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a7UUID, 3, "add_note", string(data), 1537829463)

	data = testutils.MustMarshalJSON(t, actions.AddNoteDataV3{NoteUUID: "note-4-uuid", BookUUID: linuxBookUUID, Content: "note-4", Public: false})
	a8UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a8UUID, 3, "add_note", string(data), 1537829463)

	c := "note-1-edited"
	data = testutils.MustMarshalJSON(t, actions.EditNoteDataV3{NoteUUID: "note-1-uuid", Content: &c})
	a9UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a9UUID, 3, "edit_note", string(data), 1537829463)

	testutils.MustExec(t, "removing bash book", db, "DELETE FROM books where label = ?", "bash")
	data = testutils.MustMarshalJSON(t, actions.RemoveBookDataV2{BookUUID: bashBookUUID})
	a10UUID := utils.GenerateUUID()
	testutils.MustExec(t, "inserting action", db,
		"INSERT INTO actions (uuid, schema, type, data, timestamp) VALUES (?, ?, ?, ?, ?)", a10UUID, 2, "remove_book", string(data), 1537829463)

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(errors.Wrap(err, "beginning a transaction"))
	}

	err = rm1.run(ctx, tx)
	if err != nil {
		tx.Rollback()
		t.Fatal(errors.Wrap(err, "failed to run"))
	}

	tx.Commit()

	// test
	var postJSBookUUID, postCSSBookUUID, postLinuxBookUUID string
	testutils.MustScan(t, "getting js book uuid", db.QueryRow("SELECT uuid FROM books WHERE label = ?", "js"), &postJSBookUUID)
	testutils.MustScan(t, "getting css book uuid", db.QueryRow("SELECT uuid FROM books WHERE label = ?", "css"), &postCSSBookUUID)
	testutils.MustScan(t, "getting linux book uuid", db.QueryRow("SELECT uuid FROM books WHERE label = ?", "linux"), &postLinuxBookUUID)

	testutils.AssertEqual(t, postJSBookUUID, newJSBookUUID, "js book uuid was not updated correctly")
	testutils.AssertEqual(t, postCSSBookUUID, newCSSBookUUID, "css book uuid was not updated correctly")
	testutils.AssertEqual(t, postLinuxBookUUID, linuxBookUUID, "linux book uuid changed")

	var a1, a2, a3, a4, a5, a6, a7, a8, a9, a10 actions.Action
	testutils.MustScan(t, "getting action 1", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a1UUID),
		&a1.Schema, &a1.Type, &a1.Data, &a1.Timestamp)
	testutils.MustScan(t, "getting action 2", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a2UUID),
		&a2.Schema, &a2.Type, &a2.Data, &a2.Timestamp)
	testutils.MustScan(t, "getting action 3", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a3UUID),
		&a3.Schema, &a3.Type, &a3.Data, &a3.Timestamp)
	testutils.MustScan(t, "getting action 4", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a4UUID),
		&a4.Schema, &a4.Type, &a4.Data, &a4.Timestamp)
	testutils.MustScan(t, "getting action 5", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a5UUID),
		&a5.Schema, &a5.Type, &a5.Data, &a5.Timestamp)
	testutils.MustScan(t, "getting action 6", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a6UUID),
		&a6.Schema, &a6.Type, &a6.Data, &a6.Timestamp)
	testutils.MustScan(t, "getting action 7", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a7UUID),
		&a7.Schema, &a7.Type, &a7.Data, &a7.Timestamp)
	testutils.MustScan(t, "getting action 8", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a8UUID),
		&a8.Schema, &a8.Type, &a8.Data, &a8.Timestamp)
	testutils.MustScan(t, "getting action 9", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a9UUID),
		&a9.Schema, &a9.Type, &a9.Data, &a9.Timestamp)
	testutils.MustScan(t, "getting action 10", db.QueryRow("SELECT schema, type, data, timestamp FROM actions WHERE uuid = ?", a10UUID),
		&a10.Schema, &a10.Type, &a10.Data, &a10.Timestamp)

	var a1Data, a2Data, a3Data, a4Data actions.AddBookDataV2
	var a5Data, a6Data, a7Data, a8Data actions.AddBookDataV2
	var a9Data actions.EditNoteDataV3
	var a10Data actions.RemoveBookDataV2
	testutils.MustUnmarshalJSON(t, a1.Data, &a1Data)
	testutils.MustUnmarshalJSON(t, a2.Data, &a2Data)
	testutils.MustUnmarshalJSON(t, a3.Data, &a3Data)
	testutils.MustUnmarshalJSON(t, a4.Data, &a4Data)
	testutils.MustUnmarshalJSON(t, a5.Data, &a5Data)
	testutils.MustUnmarshalJSON(t, a6.Data, &a6Data)
	testutils.MustUnmarshalJSON(t, a7.Data, &a7Data)
	testutils.MustUnmarshalJSON(t, a8.Data, &a8Data)
	testutils.MustUnmarshalJSON(t, a9.Data, &a9Data)
	testutils.MustUnmarshalJSON(t, a10.Data, &a10Data)

	testutils.AssertEqual(t, a1.Schema, 2, "a1 schema mismatch")
	testutils.AssertEqual(t, a1.Type, "add_book", "a1 type mismatch")
	testutils.AssertEqual(t, a1.Timestamp, int64(1537829463), "a1 timestamp mismatch")
	testutils.AssertEqual(t, a1Data.BookName, "js", "a1 data book_name mismatch")
	testutils.AssertEqual(t, a1Data.BookUUID, newJSBookUUID, "a1 data book_uuid mismatch")

	testutils.AssertEqual(t, a2.Schema, 2, "a2 schema mismatch")
	testutils.AssertEqual(t, a2.Type, "add_book", "a2 type mismatch")
	testutils.AssertEqual(t, a2.Timestamp, int64(1537829463), "a2 timestamp mismatch")
	testutils.AssertEqual(t, a2Data.BookName, "css", "a2 data book_name mismatch")
	testutils.AssertEqual(t, a2Data.BookUUID, newCSSBookUUID, "a2 data book_uuid mismatch")

	testutils.AssertEqual(t, a3.Schema, 2, "a3 schema mismatch")
	testutils.AssertEqual(t, a3.Type, "add_book", "a3 type mismatch")
	testutils.AssertEqual(t, a3.Timestamp, int64(1537829463), "a3 timestamp mismatch")
	testutils.AssertEqual(t, a3Data.BookName, "linux", "a3 data book_name mismatch")
	testutils.AssertEqual(t, a3Data.BookUUID, linuxBookUUID, "a3 data book_uuid mismatch")

	testutils.AssertEqual(t, a4.Schema, 2, "a4 schema mismatch")
	testutils.AssertEqual(t, a4.Type, "add_book", "a4 type mismatch")
	testutils.AssertEqual(t, a4.Timestamp, int64(1537829463), "a4 timestamp mismatch")
	testutils.AssertEqual(t, a4Data.BookName, "bash", "a4 data book_name mismatch")
	testutils.AssertEqual(t, a4Data.BookUUID, bashBookUUID, "a4 data book_uuid mismatch")
}
