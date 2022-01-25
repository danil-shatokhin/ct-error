package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/xareyx/ct-error/emulate"
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

	_, dbErr := c.ReadWriteTransaction(ctx, func(ctx context.Context, readWriteTxn *spanner.ReadWriteTransaction) error {
		//insert
		id := uuid.New().String()
		ti := time.Date(1970, 1, 1, 1, 1, 1, 1, time.UTC)
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
		mut, _ := spanner.InsertStruct("machines", testItem)
		err = readWriteTxn.BufferWrite([]*spanner.Mutation{mut})
		if err != nil {
			fmt.Printf("Failed to insert struct: %s", err)
			return err
		}

		//update
		args := map[string]interface{}{
			"id":         id,
			"created_on": spanner.CommitTimestamp,
			"ended_on":   spanner.CommitTimestamp,
		}
		query := "UPDATE machines SET created_on=@created_on, ended_on=@ended_on WHERE id = @id"

		stmt := spanner.Statement{
			SQL:    query,
			Params: args,
		}
		fmt.Println(query, args)
		updatedCount, updateErr := readWriteTxn.Update(ctx, stmt)
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

	if dbErr == nil {
		fmt.Println("ok")
	}
}
