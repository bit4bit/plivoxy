package main

import (
	"github.com/antonholmquist/jason"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

//A connection exists when we receive a RequestUUID
type PlivoConnection struct {
	URL       string
	SID       string
	AuthToken string

	Limit uint64
	//count connections on this plivo
	lock  sync.Mutex
	count uint64
}

func (conn *PlivoConnection) ReachedLimit() bool {
	return conn.count >= conn.Limit
}

func (conn *PlivoConnection) Add() {
	conn.lock.Lock()
	log.Printf("Add connection at %s count %d", conn.URL, conn.count)
	conn.count += 1
	conn.lock.Unlock()
}

func (conn *PlivoConnection) Release() {
	conn.lock.Lock()
	log.Printf("Release connection at %s count %d", conn.URL, conn.count)

	conn.count -= 1
	log.Printf("Totals connection at %s count %d", conn.URL, conn.count)

	conn.lock.Unlock()
}

func (conn *PlivoConnection) Request(method string, params url.Values) (*PlivoRequest, *http.Response, error) {
	hangupUrlProxy := ""
	urlAction := conn.URL + "/" + PlivoAPIVersion + "/" + method + "/"

	if params.Get("HangupUrl") != "" {
		hangupUrlProxy = params.Get("HangupUrl")
		params.Set("HangupUrl", "http://"+*listenServer+"/hangup")
	}

	req, err := http.NewRequest("POST", urlAction, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(conn.SID, conn.AuthToken)
	client := &http.Client{Transport: http.DefaultTransport}
	resp, respErr := client.Do(req)
	if respErr != nil {
		return nil, nil, respErr
	}

	if resp.StatusCode != 200 {
		return nil, resp, nil
	}

	resp1, resp2 := splitResponse(resp, req)

	bodyData, bodyDataErr := jason.NewObjectFromReader(resp2.Body)

	if bodyDataErr != nil {
		return nil, nil, bodyDataErr
	}

	success, _ := bodyData.GetBoolean("Success")
	requestUUID, _ := bodyData.GetString("RequestUUID")

	if success {
		if hangupUrlProxy != "" {
			conn.Add()
		}

		return &PlivoRequest{Conn: conn, UUID: requestUUID, Params: params, HangupURLProxy: hangupUrlProxy},
			resp1, nil
	} else {
		return nil, resp1, nil
	}

}
