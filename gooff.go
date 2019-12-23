package gooff

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
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
	only200        bool
	db             *badger.DB
}

func init() {
	GoOffline("", true, true)
}

// GoOffline returns the default transport, including whether to prefer db
// And sets up the database
func GoOffline(dbPath string, preferDatabase, only200 bool) {
	db, err := setupDatabase(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	http.DefaultTransport = &Transport{http.DefaultTransport, preferDatabase, only200, db}
}

func setupDatabase(dbPath string) (*badger.DB, error) {
	if dbPath == "" {
		dbPath = "/tmp"
	}
	os.MkdirAll(dbPath, os.ModePerm)

	return badger.Open(badger.DefaultOptions(dbPath + "/gooff"))
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

	if !t.only200 || (t.only200 && resp.StatusCode == 200) {
		if err = t.store(req, resp); err != nil {
			return nil, err
		}
	}
	return resp, err
}

func (t *Transport) fetch(req *http.Request) (*http.Response, error) {
	var valCopy []byte

	err := t.db.View(func(txn *badger.Txn) error {
		bytesKey, err := key(req)
		if err != nil {
			return err
		}
		item, err := txn.Get(bytesKey)

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

		// capture the bytes
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		// must readd the bytes
		res.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		var b bytes.Buffer
		writer := bufio.NewWriter(&b)
		if err := res.Write(writer); err != nil {
			return err
		}
		writer.Flush()

		// must readd again
		res.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		log.Println("Caching endpoint:", req.URL.String())

		bytesKey, err := key(req)
		if err != nil {
			return err
		}
		return txn.Set(bytesKey, b.Bytes())
	})
}

func key(req *http.Request) ([]byte, error) {
	key := fmt.Sprintf("%s:%s", req.Method, req.URL.String())

	if req.Body != nil {
		bodyBytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return []byte{}, err
		}
		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		key += ":" + string(bodyBytes)
	}
	return []byte(key), nil
}
