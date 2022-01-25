package emulate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	adminInst "cloud.google.com/go/spanner/admin/instance/apiv1"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"google.golang.org/api/iterator"
	adminpb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
	instancepb "google.golang.org/genproto/googleapis/spanner/admin/instance/v1"
	"google.golang.org/grpc/codes"
)

const (
	DefaultProject  = "test-project"
	DefaultInstance = "test-instance"
	DefaultDatabase = "test-database"
)

// DefaultConfig uses default values
// but is missing DDL.
func DefaultConfig() Config {
	return Config{
		Project:  DefaultProject,
		Instance: DefaultInstance,
		Database: DefaultDatabase,
	}
}

type Config struct {
	Project  string
	Instance string
	Database string
	DDL      []string
}

func (c Config) DB() string {
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s", c.Project, c.Instance, c.Database)
}

// Spanner will be used to create/destroy spanner emulator details components
// e.g. Instances, Databases
type Spanner struct {
	cfg      Config
	instance *adminInst.InstanceAdminClient
	admin    *database.DatabaseAdminClient

	emulator Emulator
}

func New(cfg Config, emulator Emulator) *Spanner {
	return &Spanner{cfg: cfg, emulator: emulator}
}

// use grpc over rest
func (s *Spanner) host(e Emulator) (string, error) {
	grpc, rest := e.Hosts()
	if grpc != "" {
		return grpc, nil
	}
	if rest != "" {
		return rest, nil
	}
	return "", nil
}

// Run the emulator and create Instance and DB
func (s *Spanner) Run(ctx context.Context) error {
	host, err := s.host(s.emulator)
	if err != nil {
		return err
	}
	os.Setenv("SPANNER_EMULATOR_HOST", host)

	if !Running(host) {
		err = s.emulator.Run()
		if err != nil {
			return err
		}
	}
	s.instance, err = instance.NewInstanceAdminClient(ctx)
	if err != nil {
		return err
	}
	s.admin, err = database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return err
	}

	instanceExists, instanceErr := ExistsInstance(ctx, s.instance, s.cfg.Project, s.cfg.Instance)
	if instanceErr != nil {
		return instanceErr
	}
	if !instanceExists {
		err = CreateInstance(ctx, s.instance, s.cfg.Project, s.cfg.Instance)
		if err != nil {
			return err
		}
	}

	dbExists, err := ExistsDB(ctx, s.admin, s.cfg.Project, s.cfg.Instance, s.cfg.Database)
	if err != nil {
		return err
	}
	if !dbExists {
		return CreateDB(ctx, s.admin, s.cfg.Project, s.cfg.Instance, s.cfg.Database, s.cfg.DDL)
	}
	return nil
}

// Close will run cleanup and shutdown emulator
func (s *Spanner) Close(ctx context.Context) error {
	return s.emulator.Close()
}

const (
	emulatorTimeout = 2 * time.Second
)

// CreateInstance will create a spanner instance for a given project and instance
func CreateInstance(ctx context.Context, client *adminInst.InstanceAdminClient, projectID, instance string) error {
	req := &instancepb.CreateInstanceRequest{
		Parent:     fmt.Sprintf("projects/%s", projectID),
		InstanceId: instance,
		Instance: &instancepb.Instance{
			Config:      "emulator-test-config",
			NodeCount:   1,
			DisplayName: "Test Instance",
		},
	}
	op, err := client.CreateInstance(ctx, req)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), emulatorTimeout)
	defer cancel()

	c := make(chan error, 1)
	go func() {
		_, err := op.Wait(ctx)
		c <- err
	}()

	select {
	case err = <-c:
	case <-ctx.Done():
		err = ctx.Err()
	}

	return err
}

func ExistsInstance(ctx context.Context, client *adminInst.InstanceAdminClient, projectID, instance string) (bool, error) {
	req := &instancepb.ListInstancesRequest{
		Parent:   fmt.Sprintf("projects/%s", projectID),
		PageSize: 1,
		Filter:   fmt.Sprintf("name:%s", instance),
	}
	iter := client.ListInstances(ctx, req)
	_, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateDB will instantiate a new Spanner database for a given instance and project
func CreateDB(ctx context.Context, adminClient *database.DatabaseAdminClient, projectID, instance, database string, ddl []string) error {
	req := adminpb.CreateDatabaseRequest{
		Parent:          fmt.Sprintf("projects/%s/instances/%s", projectID, instance),
		CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", database),
		ExtraStatements: ddl,
	}

	op, err := adminClient.CreateDatabase(ctx, &req)
	if err != nil {
		return fmt.Errorf("Error creating database: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), emulatorTimeout)
	defer cancel()

	c := make(chan error, 1)
	go func() {
		_, err := op.Wait(ctx)
		c <- err
	}()

	select {
	case err = <-c:
		if err != nil {
			err = fmt.Errorf("Error running extra database statements: %s", err)
		}
	case <-ctx.Done():
		err = ctx.Err()
	}
	return nil
}

// ExistsDB will check if a Spanner database exists for a given instance and project
func ExistsDB(ctx context.Context, adminClient *database.DatabaseAdminClient, projectID string, instance string, database string) (bool, error) {
	req := adminpb.GetDatabaseRequest{
		Name: fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instance, database),
	}

	_, err := adminClient.GetDatabase(ctx, &req)
	if spanner.ErrCode(err) == codes.NotFound {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

// ParseDDL takes ddl string  and splits it at a ';'.
func ParseDDL(data string) []string {
	statements := strings.Split(string(data), ";")
	var ddl []string
	for _, s := range statements {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			ddl = append(ddl, trimmed)
		}
	}
	return ddl
}

// LoadDML is used to initialize data into the db
func LoadDML(ctx context.Context, client *spanner.Client, dml string) error {
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: dml,
		}
		_, err := txn.Update(ctx, stmt)
		return err
	})
	return err
}
