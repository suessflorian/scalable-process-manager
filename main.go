package main

import (
	_ "embed"
	"fmt"
	"net/http"
	"strconv"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gofrs/uuid"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	log "github.com/sirupsen/logrus"
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
type resolver struct{}

func (r *resolver) NewProcess() *Process {
	return &Process{newPID(), RUNNING}
}

func (r *resolver) AllProcesses() []Process {
	return nil
}

//go:embed schema.graphql
var schema string

func main() {
	http.Handle("/", playground.Handler("Playground", "/query"))
	http.Handle("/query", &relay.Handler{
		Schema: graphql.MustParseSchema(
			schema,
			&resolver{},
			graphql.UseFieldResolvers(),
		)})

	if err := http.ListenAndServe(":8080", nil); err != http.ErrServerClosed {
		log.WithError(err).Errorf("server closed due to unexpected error")
	}
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
