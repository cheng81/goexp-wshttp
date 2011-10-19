package main

import (
	"./dynamichttp"
	"./wshttp"
	"http"
	"runtime"
	"fmt"
	"flag"
)

var Host = flag.String("host","localhost","Host where the application will be served")
var Port = flag.Int("port",8082,"Port where to serve the application")
var Procs = flag.Int("procs",4,"Number of max Go Processors")

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(*Procs)
//	port := 8082
//	host := "130.226.133.44:8082"
	host := *(Host) + ":" + fmt.Sprint(*(Port))
	mux := dynamichttp.NewServeMux()
	dir := http.Dir("./www")
	mux.Handle("/", http.FileServer(dir))

	wshttp.EnableWsHttp(host,mux)

	fmt.Println("Running on " + host)
	err := dynamichttp.ListenAndServe(host,mux)
	if err != nil {
		fmt.Println("error",err)
	}
	fmt.Println("Bye!")
}