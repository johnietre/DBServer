package main

import (
	"fmt"
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	_ "strings"
	"sync"
)

// Collection is a container for data
type Collection struct {
	Data map[string]interface{} `json:"data"`
	sync.RWMutex
}

// Create adds data to the collection if the key doesn't already exist
func (col *Collection) Create(key string, value interface{}) bool {
	l.RLock()
	defer l.RUnlock()
	col.Lock()
	defer col.Unlock()
	if _, ok := col.Data[key]; ok {
		return false
	}
	col.Data[key] = value
	return true
}

// Read retrieves data from the collection
func (col *Collection) Read(path string) interface{} {
	col.RLock()
	defer col.RUnlock()
	// parts := strings.Split(path, "/")
	l.RLock()
	defer l.RUnlock()
	// Loop thru the parts and change the collection while going along
	// var data interface{}
	// for _, part := range parts {
	// 	if part != "" {
	// 		if _, ok := 
	// 	}
	// }
	return col.Data[path]
}

// Update updates or creates a specified field
func (col *Collection) Update(key string, value interface{}) {
	l.RLock()
	defer l.RUnlock()
	col.Lock()
	col.Data[key] = value
	col.Unlock()
}

// Delete deletes data from the collection
func (col *Collection) Delete(key string) {
	l.RLock()
	defer l.RUnlock()
	col.Lock()
	delete(col.Data, key)
	col.Lock()
}

const (
	ip string = "localhost"
	port string = "8000"
)

var (
	collections map[string]*Collection = make(map[string]*Collection)
	// collections map[string]interface{}
	l sync.RWMutex
	stop chan int = make(chan int, 1)
	logger *log.Logger
)

func clean(v interface{}) interface{} {
	switch v.(type) {
	case map[string]interface{}:
		c := &Collection{Data: v.(map[string]interface{})}
		for k, v := range c.Data {
			c.Data[k] = clean(v)
		}
		return c
	default:
		return v
	}
}

func load() {
	logger.Println("loading data")
	f, err := os.Open("data.json")
	if err != nil {
		logger.Panic(err)
	}
	var cols map[string]interface{}
	d := json.NewDecoder(f)
	if err := d.Decode(&cols); err != nil {
		logger.Panic(err)
	}
	
	for k, v := range cols {
		collections[k] = clean(v).(*Collection)
	}
	f.Close()
	logger.Println("load successful")
}

func write() {
	l.Lock()
	logger.Println("writing data")
	f, err := os.OpenFile("data.json", os.O_RDWR, 0755)
	if err != nil {
		logger.Println(err, "\n\nData")
		data, _ := json.Marshal(collections)
		logger.Println(string(data))
		return
	}
	e := json.NewEncoder(f)
	e.SetIndent("", "\t")
	if err := e.Encode(collections); err != nil {
		logger.Println(err, "\n\nData")
		data, _ := json.Marshal(collections)
		logger.Println(string(data))
	}
	logger.Println("write successful")
}

func main() {
	f, err := os.OpenFile("log.log", os.O_RDWR|os.O_APPEND, 0755)
	if err != nil {
		panic(err)
	}
	logger = log.New(f, "", log.LstdFlags)
	load()
	defer write()

	go runSocket()
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	select {
	case <-stop:
	case <-sigc:
	}
	logger.Println("shutting down")
}

func runSocket() {
	ln, err := net.Listen("tcp", ip + ":" + port)
	if err != nil {
		logger.Fatalln(err)
	}
	for {
		if conn, err := ln.Accept(); err != nil {
			logger.Println(err)
		} else {
			handleSock(conn)
		}
	}
}

func handleSock(conn net.Conn) {
	defer conn.Close()
	var bmsg [2048]byte
	l, err := conn.Read(bmsg[:])
	if err != nil {
		if err.Error() != "EOF" {
			logger.Println(err)
			return
		}
	}
	r, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(bmsg[:l])))
	if err == nil {
		handleREST(conn, r)
		return
	}
}

func handleREST(conn net.Conn, r *http.Request) {
	if data, err := json.Marshal(collections); err != nil {
		logger.Println(err)
	} else {
		resp := fmt.Sprintf(
			"HTTP/1.1 200 OK\r\nCONTENT-TYPE: application/json; charset=utf-8\r\nCONTENT-LENGTH: %d\r\n\r\n%s",
			len(data),
			string(data),
		)
		if _, err := conn.Write([]byte(resp)); err != nil {
			logger.Println(err)
		}
	}
}
