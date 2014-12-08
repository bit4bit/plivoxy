package main

import (
	"flag"
	"fmt"
	"github.com/antonholmquist/jason"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const PlivoAPIVersion = "v0.1"

var configFile = flag.String("config", "", "config file")
var listenServer = flag.String("listen", "localhost:8090", "listen on address:port")

var plivoPool = NewPlivoPool()
var proxyRequest = NewProxyRequest()

type PlivoHandler struct{}

//Send request to plivo, an limit if have HangupUrl
func (handler PlivoHandler) proxify(action string, w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	r.ParseForm()

	conn := <-plivoPool.Get()

	plivoRequest, respHTTP, respErr := conn.Request(action, r.Form)
	if respErr != nil {
		http.Error(w, respErr.Error(), 403)
	}

	//register for tracking
	if plivoRequest != nil {
		proxyRequest.Set(plivoRequest.UUID, plivoRequest)
	}

	writeResponse(w, respHTTP)
}

func (handler PlivoHandler) Call(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	handler.proxify("Call", w, r, params)
}

func (handler PlivoHandler) BulkCall(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	handler.proxify("BulkCall", w, r, params)
}

func (handler PlivoHandler) GroupCall(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	handler.proxify("GroupCall", w, r, params)
}

func (handler PlivoHandler) Hangup(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	r.ParseForm()

	requestUUID := r.Form.Get("ALegRequestUUID")
	if requestUUID != "" {
		http.NotFound(w, r)
	}

	plivoRequest := proxyRequest.Get(requestUUID)
	if plivoRequest == nil {
		panic("failed find plivo request")
	}
	plivoRequest.Conn.Release()

	resp, err := http.PostForm(plivoRequest.HangupURLProxy, r.Form)
	if err != nil {
		http.Error(w, err.Error(), 403)
	}
	proxyRequest.Del(requestUUID)
	writeResponse(w, resp)
}

func main() {
	flag.Parse()

	loadPlivoPool()

	plivoHandler := &PlivoHandler{}
	router := httprouter.New()

	router.POST("/hangup", plivoHandler.Hangup)
	router.POST("/"+PlivoAPIVersion+"/Call/", plivoHandler.Call)
	router.POST("/"+PlivoAPIVersion+"/BulkCall/", plivoHandler.BulkCall)
	router.POST("/"+PlivoAPIVersion+"/GroupCall/", plivoHandler.GroupCall)

	log.Printf("listent at %s", *listenServer)
	if err := http.ListenAndServe(*listenServer, router); err != nil {
		log.Fatal(err)
	}
}

func loadPlivoPool() {

	if *configFile == "" {
		fmt.Println("need config file")
		os.Exit(1)
	}

	data, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalf(err.Error())
	}

	config, jerr := jason.NewValueFromBytes(data)
	if jerr != nil {
		log.Fatalf(jerr.Error())
	}

	plivos, err := config.AsArray()
	if err != nil {
		panic(err)
	}

	for _, item := range plivos {
		plivoConf, err := item.AsObject()
		if err != nil {
			panic(err)
		}
		url, urlErr := plivoConf.GetString("url")
		if urlErr != nil {
			log.Printf("omiting plivo error %s", urlErr.Error())
			continue
		}
		sid, sidErr := plivoConf.GetString("sid")
		if sidErr != nil {
			log.Printf("omiting plivo err %s", sidErr.Error())
			continue
		}

		authToken, authTokenErr := plivoConf.GetString("auth-token")
		if authTokenErr != nil {
			log.Printf("omiting plivo error %s", authTokenErr.Error())
			continue
		}

		limit, _ := plivoConf.GetNumber("limit")
		if limit == 0 {
			limit = 1
		}
		log.Printf("Adding plivo %s", url)
		plivoPool.Add(url, sid, authToken, uint64(limit))
	}
}
