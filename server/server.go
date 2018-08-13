package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Zac-Garby/db/db"
	"github.com/gorilla/mux"
)

// A Server listens for query requests over HTTP and manages a database instance.
type Server struct {
	Addr     string
	Database *db.DB
}

// NewServer makes a new server, initialising a database from the schema string.
func NewServer(addr, schema string) (*Server, error) {
	sch := &db.Schema{}
	if err := db.SchemaParser.ParseString(schema, sch); err != nil {
		return nil, err
	}

	d, err := db.MakeDB(sch)
	if err != nil {
		return nil, err
	}

	return &Server{
		Addr:     addr,
		Database: d,
	}, nil
}

// Listen starts listening on the given address.
func (s *Server) Listen() error {
	r := mux.NewRouter()
	r.HandleFunc("/json", s.handleJSON)
	r.HandleFunc("/set", s.handleSet)

	return http.ListenAndServe(s.Addr, r)
}

func (s *Server) handleJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/json")

	if r.Method != "GET" {
		errorMessage(w, "only GET is supported for /json")
		return
	}

	if err := r.ParseForm(); err != nil {
		errorMessage(w, err.Error())
		return
	}

	if len(r.Form["selector"]) != 1 {
		errorMessage(w, "only one form value expected for the selector")
		return
	}

	res, err := s.Database.QueryString(r.Form["selector"][0])
	if err != nil {
		errorMessage(w, err.Error())
		return
	}

	fmt.Fprint(w, res.JSON())
}

func (s *Server) handleSet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/json")

	if r.Method != "POST" {
		errorMessage(w, "only POST is supported for /set")
		return
	}

	if err := r.ParseForm(); err != nil {
		errorMessage(w, err.Error())
		return
	}

	if len(r.Form["selector"]) != 1 {
		errorMessage(w, "only one form value expected for the selector")
		return
	}

	if r.Body == nil {
		errorMessage(w, "expected a request body")
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorMessage(w, "could not read request body")
		return
	}

	item, err := s.Database.QueryString(r.Form["selector"][0])
	if err != nil {
		errorMessage(w, err.Error())
		return
	}

	var val interface{}
	if err := json.Unmarshal(body, &val); err != nil {
		errorMessage(w, err.Error())
		return
	}

	if err = item.Set(val); err != nil {
		errorMessage(w, err.Error())
		return
	}

	fmt.Fprint(w, item.JSON())
}

func errorMessage(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusInternalServerError)

	bytes, err := json.Marshal(map[string]string{
		"err": msg,
	})

	if err != nil {
		fmt.Fprint(w, `{"err": "couldn't convert error message to JSON"}`)
		return
	}

	w.Write(bytes)
}
