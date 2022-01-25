package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/xareyx/ct-error/emulate"
	"google.golang.org/api/iterator"
)

func main() {
	ctx := context.Background()
	cfg := emulate.DefaultConfig()
	cfg.Database = "testdb"
	cfg.DDL = emulate.ParseDDL(strings.Join(liveSpannerDDL[:], ";\n"))
	spannerEmulator := emulate.New(cfg, emulate.DefaultEmulator)
	err := spannerEmulator.Run(ctx)
	if err != nil {
		fmt.Printf("Can't start spanner emulator: %s", err)
		return
	}
	defer spannerEmulator.Close(ctx)

	dbname := "projects/" + cfg.Project + "/instances/" + cfg.Instance + "/databases/" + cfg.Database
	c, _ := spanner.NewClient(ctx, dbname)
	id := uuid.New().String()
	ti := time.Date(1970, 1, 1, 1, 1, 1, 1, time.UTC)

	_, dbErr1 := c.ReadWriteTransaction(ctx, func(ctx context.Context, readWriteTxn *spanner.ReadWriteTransaction) error {
		//insert
		testItem := &Test{
			ID:             id,
			InstanceName:   "instance-name",
			InstanceIP:     "instance-ip",
			Status:         "available",
			Version:        "version",
			Provisioner:    "provisioner",
			InstanceGroup:  "instance-group",
			Zone:           "zone",
			CreatedOn:      spanner.NullTime{Valid: true, Time: ti},
			FirstHeartbeat: spanner.NullTime{Valid: true, Time: ti},
			LastHeartbeat:  spanner.NullTime{Valid: true, Time: ti},
			EndedOn:        spanner.NullTime{Valid: true, Time: ti},
		}
		mut, mutErr := spanner.InsertStruct("machines", testItem)
		if mutErr != nil {
			return mutErr
		}
		err = readWriteTxn.BufferWrite([]*spanner.Mutation{mut})
		if err != nil {
			fmt.Printf("Failed to insert struct: %s", err)
			return err
		}

		return nil
	})

	if dbErr1 != nil {
		fmt.Printf("failed to insert: %s", dbErr1)
		return
	}

	_, dbErr2 := c.ReadWriteTransaction(ctx, func(ctx context.Context, readWriteTxn *spanner.ReadWriteTransaction) error {
		//get to confirm
		getStmt := spanner.Statement{
			SQL: "SELECT * FROM machines WHERE id = @ID",
			Params: map[string]interface{}{
				"ID": id,
			},
		}

		var iter *spanner.RowIterator
		iter = readWriteTxn.Query(ctx, getStmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err == iterator.Done {
			return fmt.Errorf("Machine not found")
		}
		if err != nil {
			return fmt.Errorf("error occured while getting Machine '%s': %w", id, err)
		}

		m := &Test{}
		if err := row.ToStruct(m); err != nil {
			return fmt.Errorf("Could not parse row into machine struct: %w", err)
		}

		fmt.Printf("get machine:\n%+v", m)

		//update
		args := map[string]interface{}{
			"id":         id,
			"created_on": "PENDING_COMMIT_TIMESTAMP()",
			"ended_on":   "PENDING_COMMIT_TIMESTAMP()",
		}
		query := "UPDATE machines SET created_on=@created_on, ended_on=@ended_on WHERE id = @id"

		updateStmt := spanner.Statement{
			SQL:    query,
			Params: args,
		}
		//fmt.Println(query, args)
		updatedCount, updateErr := readWriteTxn.Update(ctx, updateStmt)
		if updatedCount == 0 {
			fmt.Println("no rows updated")
			return fmt.Errorf("no rows updated")
		}
		if updateErr != nil {
			fmt.Printf("error updating query = %s, args = %s: %s", query, args, updateErr)
			return updateErr
		}

		return nil
	})

	if dbErr2 != nil {
		fmt.Println("!ok: ", dbErr2)
	} else {
		fmt.Println("ok")
	}
}
