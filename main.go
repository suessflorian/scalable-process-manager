package main

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	log "github.com/sirupsen/logrus"

	_ "github.com/mattn/go-sqlite3" // sqlite driver
)

type ProcessState string

const (
	RUNNING  ProcessState = "RUNNING"
	FINISHED ProcessState = "FINISHED"
	CANCELED ProcessState = "CANCELED"
)

type Process struct {
	Id     pid
	Status ProcessState
}

type resolver struct{ *manager }

func (r *resolver) Process() *resolver                         { return r }
func (r *resolver) All(ctx context.Context) ([]Process, error) { return r.List(ctx) }
func (r *resolver) New(ctx context.Context) (Process, error)   { return r.Spawn(ctx) }

func (r *resolver) Stop(ctx context.Context, args struct{ Pid pid }) (Process, error) {
	return r.Interupt(ctx, args.Pid)
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
			&resolver{&manager{store, make(map[pid]context.CancelFunc)}},
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
func (s *store) New(ctx context.Context) (process Process, err error) {
	return process, s.db.QueryRowContext(ctx, "INSERT INTO processes(status) VALUES($1) RETURNING id, status", RUNNING).Scan(&process.Id, &process.Status)
}
func (s *store) Update(ctx context.Context, update Process) (process Process, err error) {
	return process, s.db.QueryRowContext(ctx, "UPDATE processes SET status=$1 WHERE id=$2 RETURNING id, status", update.Status, update.Id).Scan(&process.Id, &process.Status)
}

func (s *store) List(ctx context.Context) ([]Process, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, status FROM processes")
	if err != nil {
		return nil, err
	}

	var processes []Process
	for rows.Next() {
		var process Process
		if err := rows.Scan(&process.Id, &process.Status); err != nil {
			return nil, err
		}
		processes = append(processes, process)
	}

	return processes, nil
}

type manager struct {
	*store
	managing map[pid]context.CancelFunc
}

func (m *manager) Spawn(ctx context.Context) (p Process, err error) {
	p, err = m.store.New(ctx)
	if err != nil {
		return
	}

	ctx, m.managing[p.Id] = context.WithCancel(context.Background())

	go func(ctx context.Context) {
		time.Sleep(time.Second * 5)
		_, err := m.Update(ctx, Process{Id: p.Id, Status: FINISHED})
		if err != nil && err != context.Canceled {
			log.WithError(err).Fatal("couldn't update process state")
		}
	}(ctx)

	return p, nil
}

func (m *manager) Interupt(ctx context.Context, pid pid) (process Process, err error) {
	if cancel, ok := m.managing[pid]; !ok {
		return process, errors.New("process not managed by this instance")
	} else {
		cancel()
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()

	return m.Update(ctx, Process{Id: pid, Status: CANCELED})
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
	case int32:
		*p = pid(input)
	default:
		err = fmt.Errorf("wrong type for ID: %T", input)
	}
	return err
}

func (pid) ImplementsGraphQLType(name string) bool {
	return name == "ID"
}
