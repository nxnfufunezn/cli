package migrate

import (
	"testing"

	"github.com/dnote/cli/testutils"
	"github.com/pkg/errors"
)

func TestMigration1(t *testing.T) {
	// set up
	ctx := testutils.InitEnv("../tmp", "../testutils/fixtures/schema.sql")
	defer testutils.TeardownEnv(ctx)

	db := ctx.DB

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
}
