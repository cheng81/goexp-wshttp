package wshttp

import(
	"./dynamichttp"
	"websocket"
	// "io"
	"log"
	"http"
	"strings"
	// "url"
)

func EnableWsHttp(host string,mux *dynamichttp.ServeMux) {
	core := &WsHttpCore{mux,MakeDefaultConnections(),host}
	core.Init()
	mux.Handle("/meta/wsconn",makeWsConnHandler(core))
	mux.Handle("/meta/wshttp",makeWsHttpHandler(core))
}

func makeWsConnHandler(core *WsHttpCore) http.Handler {
	return websocket.Handler(func(c *websocket.Conn) {
		id := core.RegisterConn(c)
		log.Println("send connid",id)
		c.Write([]byte(id))
		// chanWriter := core.MakeChanWriter(id)
		jsChan := core.MakeJsonChan(id)
		for {
			var data WsHttpMsg
			err := websocket.JSON.Receive(c,&data)
			if err != nil {
				log.Println("Error reading, socket closed")
				break
			}
			log.Println("recvd.",id,data)

			jsChan <- &data
		}
		// _,err := io.Copy(chanWriter,c)
		// if err != nil {
		// 	log.Println("wshttp.conn.error",id,err)
		// }
		// chanWriter.Close()
		core.RemoveConn(id)
	})
}

func makeWsHttpHandler(core *WsHttpCore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("/meta/wshttp got request",r.URL,r.Method)
		//we accept only post,delete
		if !(r.Method == "POST" || r.Method == "DELETE") { w.WriteHeader(405); log.Println("Wrong method",r.Method); return }

		connid := r.FormValue("connid")
		url := r.FormValue("url")
		methods := strings.Split(r.FormValue("methods"),":")
		log.Println("Request info",connid,url,methods)
		if connid=="" || url=="" { w.WriteHeader(400); log.Println("Empty connid or url"); return }
		switch r.Method {
			case "POST":
				log.Println("Registering",url,methods)
				//connid,url and methods are mandatory
				if r.FormValue("methods") == "" { w.WriteHeader(400); log.Println("Empty methods"); return }
				err := core.RegisterWsHttpHandler(connid,url,methods)
				if err != nil {
					log.Println("core.RegisterWsHttpHandler/error",connid,url,methods,err)
					w.WriteHeader(500)
					return
				}
			case "DELETE":
				log.Println("Removing",url)
				err := core.RemoveWsHttpHandler(connid,url,methods)
				if err != nil {
					log.Println("core.RemoveWsHttpHandler/error",connid,url,methods,err)
					w.WriteHeader(500)
					return
				}
		}
		log.Println("/meta/wshttp done")
		w.Write([]byte("true"))
	})
}
