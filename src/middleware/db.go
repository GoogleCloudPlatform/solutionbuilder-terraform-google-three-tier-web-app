// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"cloud.google.com/go/cloudsqlconn"
	"cloud.google.com/go/cloudsqlconn/postgres/pgxv4"
)

// SQLStorage is a wrapper for database operations
type SQLStorage struct {
	db *sql.DB
}

func (s *SQLStorage) log(msg string) {
	log.Printf("sql       : %s\n", msg)
}

// Init kicks off the database connector
func (s *SQLStorage) Init(user, password, host, name, conn string) error {
	var err error
	instanceConnectionName := conn

	s.log("Opening connection")

	if password == "" {
		s.log("method is service account")
		trimmedUser := strings.ReplaceAll(user, ".gserviceaccount.com", "")
		if s.db, err = connectWithConnector(trimmedUser, password, name, instanceConnectionName); err != nil {
			return fmt.Errorf("could not open connection using Service Account: %s", err)
		}
	} else {
		s.log("method is database user")
		if s.db, err = connectDirect(user, password, name, host); err != nil {
			return fmt.Errorf("could open connection using username/password: %s", err)
		}
	}

	s.log("Connection opened")

	s.log("Pinging")
	if err := s.db.Ping(); err != nil {
		return fmt.Errorf("could not ping database: %s", err)
	}
	s.log("Pinging complete")

	populated, err := s.SchemaExists()
	if err != nil {
		return fmt.Errorf("schema exists failure: %s", err)
	}

	if !populated {
		s.log("populating schema")
		if err := s.SchemaInit(); err != nil {
			return fmt.Errorf("cannot populate schema: %s", err)
		}
	}

	s.log("Schema populated")

	return nil
}

func connectWithConnector(user, pass, name, connection string) (*sql.DB, error) {
	cleanup, err := pgxv4.RegisterDriver(
		"cloudsql-postgres",
		cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPrivateIP()),
		cloudsqlconn.WithIAMAuthN(),
	)
	if err != nil {
		log.Fatalf("uncaught error occured: %s", err)
	}
	defer cleanup()

	connectString := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable", connection, user, name)

	return sql.Open(
		"cloudsql-postgres",
		connectString,
	)
}

func connectDirect(user, pass, name, connection string) (*sql.DB, error) {
	connectString := fmt.Sprintf("host=%s user=%s password=%s port=%s dbname=%s sslmode=disable", connection, user, pass, "5432", name)

	return sql.Open("pgx", connectString)
}

// Close ends the database connection
func (s *SQLStorage) Close() error {
	s.log("close called on database")
	return s.db.Close()
}

// SchemaExists checks to see if the schema has been prepopulated
func (s SQLStorage) SchemaExists() (bool, error) {
	s.log("Checking schema exists")
	var result string
	err := s.db.QueryRow(`SELECT
    table_schema || '.' || table_name
FROM
    information_schema.tables
WHERE
    table_type = 'BASE TABLE'
AND
    table_schema NOT IN ('pg_catalog', 'information_schema');`).Scan(&result)
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("schema check failed: %s", err)
	}
	s.log("Schema check complete")
	s.log(fmt.Sprintf("Schema err: %s", err))
	return len(result) > 0, nil
}

// SchemaInit will initialize the schema
func (s *SQLStorage) SchemaInit() error {
	sl := make([]string, 0)

	sl = append(sl, `CREATE TABLE todo (
		id SERIAL PRIMARY KEY,
		title varchar(512) DEFAULT NULL,
		updated timestamp DEFAULT NULL,
		completed timestamp DEFAULT NULL)`)
	sl = append(sl, `INSERT INTO todo (id, title, updated, completed)
					VALUES
	  				(1,'Install and configure todo app','2021-10-28 12:00:00','2021-10-28 12:00:00'),
					(2,'Add your own todo','2021-10-28 12:00:00',NULL),
					(3,'Mark task 1 done','2021-10-27 14:26:00',NULL)`)

	sl = append(sl, `SELECT setval('todo_id_seq', (SELECT MAX(id) FROM todo)+1)`)

	// Get new Transaction. See http://golang.org/pkg/database/sql/#DB.Begin
	txn, err := s.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		// Rollback the transaction after the function returns.
		// If the transaction was already commited, this will do nothing.
		_ = txn.Rollback()
	}()

	for _, q := range sl {
		// Execute the query in the transaction.
		s.log(fmt.Sprintf("Executing sql: %s", q))
		_, err := txn.Exec(q)
		if err != nil {
			return err
		}
	}

	// Commit the transaction.
	return txn.Commit()
}

// List returns a list of all todos
func (s SQLStorage) List() (Todos, error) {
	ts := Todos{}
	results, err := s.db.Query("SELECT * FROM todo ORDER BY updated DESC")
	if err != nil {
		return ts, fmt.Errorf("list error: on query: %s", err)
	}

	for results.Next() {
		t, err := resultToTodo(results)
		if err != nil {
			return ts, fmt.Errorf("list error: on resultToTodo: %s", err)
		}

		ts = append(ts, t)
	}
	return ts, nil
}

// Create records a new todo in the database.
func (s SQLStorage) Create(t Todo) (Todo, error) {
	sql := `
		INSERT INTO todo(title, updated) 
		VALUES($1, NOW() )	
		RETURNING id	
	`

	if t.Complete {
		sql = `
		INSERT INTO todo(title, updated, completed) 
		VALUES($1,NOW(),NOW())
		RETURNING id	
	`
	}

	var id int

	if err := s.db.QueryRow(sql, t.Title).Scan(&id); err != nil {
		return t, fmt.Errorf("create error: on exec: %s", err)
	}

	t.ID = int(id)

	return t, nil
}

func resultToTodo(results *sql.Rows) (Todo, error) {
	t := Todo{}
	if err := results.Scan(&t.ID, &t.Title, &t.Updated, &t.completedNull); err != nil {
		return t, fmt.Errorf("resultToTodo error: on scan: %s", err)
	}

	if t.completedNull.Valid {
		t.Completed = t.completedNull.Time
		t.Complete = true
	}

	return t, nil
}

// Read returns a single todo from the database
func (s SQLStorage) Read(id string) (Todo, error) {
	t := Todo{}
	results, err := s.db.Query("SELECT * FROM todo WHERE id =$1;", id)
	if err != nil {
		s.log(fmt.Sprintf("could not read item: %s", err))
		return t, fmt.Errorf("read error: Query: %s", err)
	}

	results.Next()
	t, err = resultToTodo(results)
	if err != nil {
		return t, fmt.Errorf("read error: resultToTodo: %s", err)
	}

	return t, nil
}

// Update changes one todo in the database.
func (s SQLStorage) Update(t Todo) error {
	orig, err := s.Read(strconv.Itoa(t.ID))
	if err != nil {
		s.log(fmt.Sprintf("could not read item to update it: %s", err))
		return err
	}

	sql := `
		UPDATE todo
		SET title = $1, updated = NOW() 
		WHERE id = $2
	`

	if t.Complete && !orig.Complete {
		sql = `
		UPDATE todo
		SET title = $1, updated = NOW(), completed = NOW() 
		WHERE id = $2
	`
	}

	if orig.Complete && !t.Complete {
		sql = `
		UPDATE todo
		SET title = $1, updated = NOW(), completed = NULL 
		WHERE id = $2
	`
	}

	op, err := s.db.Prepare(sql)
	if err != nil {
		s.log(fmt.Sprintf("could not prepare item to update: %s", err))
		return fmt.Errorf("update error: on prepare: %s", err)
	}

	_, err = op.Exec(t.Title, t.ID)

	if err != nil {
		s.log(fmt.Sprintf("could not exec update: %s", err))
		return fmt.Errorf("update error: on exec: %s", err)
	}

	return nil
}

// Delete removes one todo from the database.
func (s SQLStorage) Delete(id string) error {
	op, err := s.db.Prepare("DELETE FROM todo WHERE id =$1")
	if err != nil {
		s.log(fmt.Sprintf("could not prepare item to delete: %s", err))
		return fmt.Errorf("delete error: on prepare: %s", err)
	}

	if _, err = op.Exec(id); err != nil {
		s.log(fmt.Sprintf("could not exec delete: %s", err))
		return fmt.Errorf("delete error: on exec: %s", err)
	}

	return nil
}
