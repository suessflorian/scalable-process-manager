package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/georgysavva/scany/sqlscan"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	log "github.com/sirupsen/logrus"

	_ "github.com/mattn/go-sqlite3" // sqlite driver
)

type ProcessState string

const (
	RUNNING ProcessState = "RUNNING"
)

type Process struct {
	Id     pid
	Status ProcessState
}
type resolver struct{ *store }

func (r *resolver) Process() *resolver                         { return r }
func (r *resolver) All(ctx context.Context) ([]Process, error) { return r.store.List(ctx) }
func (r *resolver) New(ctx context.Context) (Process, error)   { return r.store.New(ctx) }

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

	log.Info("server on localhost:8080...")
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
func (s *store) New(ctx context.Context) (Process, error) {
	var process Process
	return process, sqlscan.Get(ctx, s.db, &process, "INSERT INTO processes(status) VALUES($1) RETURNING *", RUNNING)
}

func (s *store) List(ctx context.Context) ([]Process, error) {
	var processes []Process
	return processes, sqlscan.Select(ctx, s.db, &processes, "SELECT id, status FROM processes")
}

type pid int

func (p pid) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, fmt.Sprintf("%d", p)), nil
}

func (p *pid) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		uuid, err := strconv.Atoi(input)
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
