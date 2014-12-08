package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

func splitResponse(src *http.Response, req *http.Request) (*http.Response, *http.Response) {
	var buff bytes.Buffer

	buff.ReadFrom(src.Body)

	pipeR, pipeW := io.Pipe()
	go func() {
		buf := bytes.NewBuffer(buff.Bytes())
		io.Copy(pipeW, buf)
		pipeR.Close()
	}()

	pipe2R, pipe2W := io.Pipe()
	go func() {
		buf := bytes.NewBuffer(buff.Bytes())
		io.Copy(pipe2W, buf)
		pipe2R.Close()
	}()

	resp1 := &http.Response{}
	*resp1 = *src
	resp1.Body = pipeR

	resp2 := &http.Response{}
	*resp2 = *src
	resp2.Body = pipe2R

	return resp1, resp2
}

func writeResponse(w http.ResponseWriter, r *http.Response) {
	//send response to client
	for k, v := range r.Header {
		for _, item := range v {
			w.Header().Add(k, item)
		}
	}

	w.WriteHeader(r.StatusCode)
	body, err := ioutil.ReadAll(r.Body)
	if err != nil && err != io.ErrClosedPipe {
		panic(err)
	}
	w.Write(body)
}
