package main

import (
	"database/sql"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gofrs/uuid"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	log "github.com/sirupsen/logrus"

	_ "github.com/mattn/go-sqlite3" // sqlite driver
)

type ProcessState string

const PORT = 8080

const (
	RUNNING  ProcessState = "RUNNING"
	FINISHED ProcessState = "FINISHED"
	FAILED   ProcessState = "FAILED"
)

type Process struct {
	Pid    pid
	Status ProcessState
}
type resolver struct{ *store }

func (r *resolver) NewProcess() *Process {
	return &Process{newPID(), RUNNING}
}

func (r *resolver) AllProcesses() []Process {
	return nil
}

//go:embed schema.graphql
var schema string

func main() {
	store := mustNewStore()
	defer store.Close()

	http.Handle("/", playground.Handler("Playground", "/query"))
	http.Handle("/query", &relay.Handler{
		Schema: graphql.MustParseSchema(
			schema,
			&resolver{store},
			graphql.UseFieldResolvers(),
		)})

	log.Info("server on localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != http.ErrServerClosed {
		log.WithError(err).Errorf("server closed due to unexpected error")
	}
}

type store struct {
	db *sql.DB
}

//go:embed migration.sql
var migration string

func mustNewStore() *store {
	if _, err := os.Create("./data.db"); err != nil {
		log.WithError(err).Fatal("failed to create sqlite file")
	}

	conn, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.WithError(err).Fatal("failed to establish connection")
	}

	if _, err := conn.Exec(migration); err != nil {
		log.WithError(err).Fatal("failed to run migration")
	}

	return &store{conn}
}

func (s *store) Close() error { return s.db.Close() }
func (s *store) UpdateProcess(pid pid, status ProcessState) {
	s.db.Exec("INSERT INTO processes(id, status) VALUES()")
}

type pid uuid.UUID

func (p pid) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, uuid.UUID(p).String()), nil
}

func (p *pid) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		uuid, err := uuid.FromString(input)
		if err != nil {
			return err
		}
		*p = pid(uuid)
	default:
		err = fmt.Errorf("wrong type for ID: %T", input)
	}
	return err
}

func (pid) ImplementsGraphQLType(name string) bool {
	return name == "ID"
}

func newPID() pid {
	uuid, _ := uuid.NewV4()
	return pid(uuid)
}
