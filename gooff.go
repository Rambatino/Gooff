package gooff

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dgraph-io/badger"
)

// Transport implements http.RoundTripper. When set as Transport of http.Client,
// it will store online requests for offline usage
type Transport struct {
	Transport      http.RoundTripper
	preferDatabase bool
	db             *badger.DB
}

// GoOffline returns the default transport, including whether to prefer db
// And sets up the database
func GoOffline(dbPath string, preferDatabase bool) *Transport {
	db, err := setupDatabase(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	return &Transport{http.DefaultTransport, preferDatabase, db}
}

func setupDatabase(dbPath string) (*badger.DB, error) {
	if dbPath == "" {
		dbPath = "/tmp"
	}
	os.MkdirAll(dbPath, os.ModePerm)

	opts := badger.DefaultOptions
	opts.Dir = dbPath + "/gooff"
	opts.ValueDir = dbPath + "/gooff"

	return badger.Open(opts)
}

// RoundTrip is the core part of this module and implements http.RoundTripper.
//
// If prefer database is true, always try and return value of request from
// database
//
// If there is data, return it
// If there is no data, then send the request anyway
//
// Store the result, linking it to the request
// If the request errors, then attempt to pull from database, otherwise return
// as is
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	if t.preferDatabase {
		if resp, err = t.fetch(req); err == nil {
			return resp, nil
		}
	}

	resp, err = t.Transport.RoundTrip(req)

	if err != nil {
		if resp, err = t.fetch(req); err == nil {
			return resp, nil
		}
		return nil, err
	}

	if err = t.store(req, resp); err != nil {
		return nil, err
	}
	return resp, err
}

func (t *Transport) fetch(req *http.Request) (*http.Response, error) {
	var valCopy []byte

	err := t.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key(req))

		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			valCopy = append([]byte{}, val...)
			return nil
		})

		return nil
	})

	if err != nil {
		return nil, err
	}
	return http.ReadResponse(bufio.NewReader(bytes.NewReader(valCopy)), req)
}

func (t *Transport) store(req *http.Request, res *http.Response) error {
	return t.db.Update(func(txn *badger.Txn) error {
		l := lol{}
		err := res.Write(&l)
		if err != nil {
			return err
		}
		return txn.Set(key(req), l.b)
	})
}

type lol struct {
	b []byte
}

func (l *lol) Write(p []byte) (int, error) {
	l.b = append([]byte{}, p...)
	return 0, nil
}

func key(req *http.Request) []byte {
	return []byte(fmt.Sprintf("%s:%s", req.Method, req.URL.String()))
}
