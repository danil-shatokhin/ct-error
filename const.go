package main

import "cloud.google.com/go/spanner"

var spannerDDL = `CREATE TABLE test (
	id STRING(MAX) NOT NULL,
	t1 TIMESTAMP OPTIONS (allow_commit_timestamp=true),
	t2 TIMESTAMP OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (id);`

type Test struct {
	ID string           `spanner:"id"`
	T1 spanner.NullTime `spanner:"t1"`
	T2 spanner.NullTime `spanner:"t2"`
}
