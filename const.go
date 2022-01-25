package main

import "cloud.google.com/go/spanner"

/*var spannerDDL = `CREATE TABLE test (
	id INT64 NOT NULL,
	t1 TIMESTAMP OPTIONS (allow_commit_timestamp=true),
	t2 TIMESTAMP OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (id);`*/
var liveSpannerDDL = [...]string{
	// stream_keys table and indices
	`CREATE TABLE stream_keys (
		stream_key STRING(MAX) NOT NULL,
		created_on TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
		disabled_on TIMESTAMP OPTIONS (allow_commit_timestamp=true),
		session_url STRING(MAX) NOT NULL,
		updated_on TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
	) PRIMARY KEY (stream_key)`,

	// provisioners table and indices
	`CREATE TABLE provisioners (
		id STRING(MAX) NOT NULL,
		last_heartbeat TIMESTAMP OPTIONS (allow_commit_timestamp=true),
		is_studio_base BOOL NOT NULL,
	) PRIMARY KEY (id)`,
	`CREATE INDEX idx_provisioners_last_heartbeat ON provisioners(last_heartbeat)`,

	// machines table and indices
	`CREATE TABLE machines (
		id STRING(MAX) NOT NULL,
		instance_name STRING(MAX) NOT NULL,
		instance_ip STRING(MAX) NOT NULL,
		status STRING(MAX) NOT NULL,
		version STRING(MAX) NOT NULL,
		created_on TIMESTAMP OPTIONS (allow_commit_timestamp = true),
		last_heartbeat TIMESTAMP OPTIONS (allow_commit_timestamp = true),
		ended_on TIMESTAMP OPTIONS (allow_commit_timestamp = true),
		provisioner STRING(MAX) NOT NULL,
		first_heartbeat TIMESTAMP OPTIONS (allow_commit_timestamp = true),
		instance_group STRING(MAX) NOT NULL,
		zone STRING(MAX) NOT NULL,
	) PRIMARY KEY (id)`,
	`CREATE INDEX idx_machines_provisioner_status_with_created_on ON machines(provisioner,status) STORING (created_on)`,

	// accounts table and indices
	`CREATE TABLE accounts (
		access_key STRING(MAX) NOT NULL,
		secret_key STRING(MAX) NOT NULL,
		permissions STRING(MAX) NOT NULL,
	) PRIMARY KEY (access_key)`,

	// sessions table and indices
	`CREATE TABLE sessions (
		s_id STRING(255) NOT NULL,
		status STRING(255) NOT NULL,
		notify STRING(255),
		stream_type STRING(255) NOT NULL,
		reason STRING(4096),
		account_id STRING(255),
		machine_id STRING(255),
		token STRING(255),
		created_on TIMESTAMP NOT NULL,
		pending_since TIMESTAMP,
		started_on TIMESTAMP,
		ended_on TIMESTAMP,
		ingest_cookie STRING(255),
		stream_key STRING(255) NOT NULL,
		max_height INT64,
		ingest_height INT64,
		ingest_width INT64,
		ingest_error STRING(255),
		archive_url STRING(MAX),
		archive_status STRING(255),
		archive_error STRING(255),
		archive_file_size INT64,
		provisioned_on TIMESTAMP,
		provision_expiration_override TIMESTAMP,
		archive_transcodes STRING(255),
		archive_duration FLOAT64,
		priority INT64 NOT NULL,
		rtmp_mode BOOL,
		rtmp_url STRING(2048) NOT NULL,
		provisioner STRING(255) NOT NULL,
		archive_use_transcoded BOOL,
		archive_vod_transcoding BOOL,
		archive_starlord_container STRING(255),
		metadata STRING(MAX),
		archive_subtitles STRING(255),
		archive_subtitles_language STRING(255),
		audit_machine_id STRING(255),
		auto_cc_duration FLOAT64,
		low_latency BOOL,
	) PRIMARY KEY (s_id)`,
	`CREATE INDEX idx_sessions_stream_key ON sessions(stream_key)`,
	`CREATE INDEX idx_sessions_provisioner_status_priority_with_created_on ON sessions(provisioner,status,priority DESC) STORING (created_on)`,
	`CREATE INDEX idx_sessions_status_machine_id ON sessions(status,machine_id)`,
	`CREATE INDEX idx_sessions_machine_id ON sessions(machine_id)`,

	// simulcast_destinations table and indices
	`CREATE TABLE simulcast_destinations (
		id STRING(MAX) NOT NULL,
		host STRING(MAX) NOT NULL,
		path STRING(MAX) NOT NULL,
		port INT64 NOT NULL,
		service_name STRING(MAX) NOT NULL,
		session_id STRING(255),
		stream_key STRING(MAX) NOT NULL,
		use_ssl BOOL NOT NULL,
	) PRIMARY KEY (id)`,
	`CREATE INDEX idx_simulcast_destinations_session_id ON simulcast_destinations(session_id)`,
}

type Test struct {
	ID             string           `spanner:"id"`
	InstanceName   string           `spanner:"instance_name"`
	InstanceIP     string           `spanner:"instance_ip"`
	Status         string           `spanner:"status"`
	Version        string           `spanner:"version"`
	CreatedOn      spanner.NullTime `spanner:"created_on"`
	LastHeartbeat  spanner.NullTime `spanner:"last_heartbeat"`
	EndedOn        spanner.NullTime `spanner:"ended_on"`
	Provisioner    string           `spanner:"provisioner"`
	FirstHeartbeat spanner.NullTime `spanner:"first_heartbeat"`
	InstanceGroup  string           `spanner:"instance_group"`
	Zone           string           `spanner:"zone"`
}
