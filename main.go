package main

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/xareyx/ct-error/emulate"
)

type test struct {
	ID int64            `spanner:"id"`
	T1 spanner.NullTime `spanner:"last_heartbeat"`
	T2 spanner.NullTime `spanner:"ended_on"`
}

var spannerDDL = `CREATE TABLE test (
					id INT64 NOT NULL,
					t1 TIMESTAMP OPTIONS (allow_commit_timestamp=true),
					t2 TIMESTAMP OPTIONS (allow_commit_timestamp=true),
				) PRIMARY KEY (id);`

func main() {
	ctx := context.Background()
	cfg := emulate.DefaultConfig()
	cfg.Database = "testdb"
	cfg.DDL = emulate.ParseDDL(spannerDDL)
	spannerEmulator := emulate.New(cfg, emulate.DefaultEmulator)
	err := spannerEmulator.Run(ctx)
	if err != nil {
		fmt.Printf("Can't start spanner emulator: %s", err)
		return
	}
	defer spannerEmulator.Close(ctx)

	dbname := "projects/" + cfg.Project + "/instances/" + cfg.Instance + "/databases/" + cfg.Database
	c, _ := spanner.NewClient(ctx, dbname)

	_, _ = c.ReadWriteTransaction(ctx, func(ctx context.Context, readWriteTxn *spanner.ReadWriteTransaction) error {
		//insert
		testItem := test{
			ID: 0,
			T1: spanner.NullTime{
				Valid: true,
				Time:  spanner.CommitTimestamp,
			},
			T2: spanner.NullTime{
				Valid: true,
				Time:  spanner.CommitTimestamp,
			},
		}
		mut, _ := spanner.InsertStruct("test", testItem)
		err = readWriteTxn.BufferWrite([]*spanner.Mutation{mut})
		if err != nil {
			fmt.Printf("Failed to insert struct: %s", err)
			return err
		}

		//update
		args := map[string]interface{}{
			"id": 0,
			"t1": spanner.CommitTimestamp,
			"t2": spanner.CommitTimestamp,
		}
		query := "UPDATE test SET t1=@t1, t2=@t2 WHERE id = @id"

		stmt := spanner.Statement{
			SQL:    query,
			Params: args,
		}
		_, updateErr := readWriteTxn.Update(ctx, stmt)
		if updateErr != nil {
			fmt.Printf("error updating query = %s, args = %s: %s", query, args, updateErr)
			return updateErr
		}

		return nil
	})

	fmt.Printf("ok")
}
