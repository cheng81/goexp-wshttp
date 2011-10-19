package main

import (
	"./dynamichttp"
	"./wshttp"
	"http"
	"runtime"
	"fmt"
)

func main() {
	runtime.GOMAXPROCS(4)
//	port := 8082
	host := "130.226.133.44:8082"
	mux := dynamichttp.NewServeMux()
	dir := http.Dir("./www")
	mux.Handle("/", http.FileServer(dir))

	wshttp.EnableWsHttp(host,mux)

	fmt.Println("Running on port 8082")
	err := dynamichttp.ListenAndServe("130.226.133.44:8082",mux)
	if err != nil {
		fmt.Println("error",err)
	}
	fmt.Println("Bye!")
}