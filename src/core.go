package wshttp

import(
	"./dynamichttp"
	"http"
	"websocket"
	"os"
	"log"
//	"io"
	"json"
	"strings"
)

type WsHttpCore struct {
	mux *dynamichttp.ServeMux
	connections Connections
	Host string
}
func (c *WsHttpCore) Init() {
	
}
func connidJson(conns []string) string {
	var out []map[string]string
	out = make([]map[string]string,len(conns))
	for id,conn := range(conns) {
		out[id] = make(map[string]string)
		out[id]["connid"] = conn
	}
	bytes,err := json.Marshal(out)
	if err != nil {log.Println("e che sfiga 2")}
	return string(bytes)
}
func (c *WsHttpCore) AddConnectionListener(ocw *ConnWrapper,txId string) {
	log.Println("WsHttpCore.AddConnectionListener",ocw.Id)
	fnc := func(t ConnEventType,cw *ConnWrapper){
		if ocw.Id == cw.Id {
			return
		}
		log.Println("ConnectionListener called")
		meth := ""; surl := "/meta/wsconnections"
		switch t {
			case Connected: meth = "POST"
			case Disconnected: meth = "DELETE"
		}
		req,err := convert(c.Host,&fakeHttpRequest{
			Method: meth,
			URL: surl,
			Body: connidJson([]string{cw.Id}),//"[{'connid':'"+cw.Id+"'}]",
		})
		if err != nil {
			log.Println("Error in connectionListener notification - convert fakeHttpRequest",err)
			return
		}
		reqBytes,err := json.Marshal(req)
		if err != nil {
			log.Println("Error in connectionListener notification - json.Marshal",err)
			return
		}
		ocw.Channel.ch <- WsHttpMsg{txId,false,string(reqBytes)}
		log.Println("ConnectionListener Done.")
	}
	curr := c.connections.AddListener(fnc)
	currStr := connidJson(curr)
	req,err := convert(c.Host,&fakeHttpRequest{
		Method: "POST", URL: "/meta/wsconnections", Body: string(currStr),
	})
	if err != nil {log.Println("e che sfiga")}
	bytes,err := json.Marshal(req)
	ocw.Channel.ch <- WsHttpMsg{txId,false,string(bytes)}
	c.onDisconnect(ocw.Id,func(){
		c.connections.RemoveListener(fnc)
	})



}
func (c *WsHttpCore) RegisterConn(w *websocket.Conn) string {
	connWrap := c.connections.RegisterConn(w)
	return connWrap.Id
}
func (c *WsHttpCore) RemoveConn(id string) {
	wrapper := c.connections.Remove(id)
	for _,lst := range(wrapper.DisconnectListeners) {
		lst()
	}
}
func (c *WsHttpCore) MakeJsonChan(id string) chan *WsHttpMsg {
	out := make(chan *WsHttpMsg)
	go func() {
		for msg := range(out) {
//			go func() {
				switch msg.IsReq {
					case true: c.DispatchHTTP(msg,id)
					case false: c.DispatchResponse([]byte(msg.Payload),nil,id,msg.Id)
				}
//			}()
		}
	}()
	return out
}
// func (c *WsHttpCore) MakeChanWriter(id string) *ChanWriter {
// 	out := MakeChanWriter(c.lookupConn(id))
// 	go func() {
// 		for msg := range(out.ch) {
// 			go func() {
// 				req := &WsHttpMsg{}
// 				err := json.Unmarshal(msg,req)
// 				if err != nil {
// 					log.Println("json.Unmarshal WsHttpReq error",err,string(msg))
// 					return
// 				}
// 				//yawg, it could be the fucking response :D
// 				go func(){
// 					switch req.IsReq {
// 						case true: c.DispatchHTTP(req,id)
// 						case false: c.DispatchResponse([]byte(req.Payload),nil,id,req.Id)
// 					}
// 				}()
// //				c.DispatchHTTP(req,id)
// 			}()
// 		}
// 	}()
// 	return &out
// }

func (c *WsHttpCore) RemoveWsHttpHandler(id string,pattern string,methods []string) os.Error {
	if c.lookupConn(id) == nil {
		return os.NewError("connection not found")
	}
	// if strings.HasPrefix(pattern,"/meta/wsconnections") {
	// 	log.Println("Request to get async notifications about connections")
	// 	c.RemoveConnectionListener(wc,r.Id)
	// 	return nil
	// }

	handler := getHandlerFor(pattern,c.mux)
	handler.RemoveAll(id,methods)
	return nil
}
func (c *WsHttpCore) RegisterWsHttpHandler(id string,pattern string,methods []string) os.Error {
	if c.lookupConn(id) == nil {
		log.Println("Connection not found!",id)
		return os.NewError("connection not found")
	}
	if strings.HasPrefix(pattern,"/meta/wsconnections") {
		log.Println("Request to get async notifications about connections")
		c.AddConnectionListener(c.lookupConn(id),id)
		//c.DispatchResponse([]byte("true"),nil,connid,r.Id)
		// }
		return nil
	}

	handler := getHandlerFor(pattern,c.mux)
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					rsp,err := c.SendReceive(id,r)
					if err != nil {
						w.WriteHeader(500)
						log.Println(pattern,"handling error",err)
						return
					}
					w.Write(rsp)
					return
				})
	for _,meth := range(methods) {
		handler.Add(meth,id,fn)
	}
	c.onDisconnect(id,func(){
		handler.RemoveAll(id,[]string{})
	})
	log.Println("RegisterWsHttpHandler done")
	return nil
}

func (c *WsHttpCore) SendReceive(id string,r *http.Request) ([]byte,os.Error) {
	conn := c.lookupConn(id)
	if conn == nil { //..this should never happen
		log.Println("Connection leaked, Handler present but connection gone",id)
		return nil,os.NewError("NoConnection")
	}
	return conn.HTTP(r)
}

func (c *WsHttpCore) DispatchHTTP(r *WsHttpMsg,connid string) {
	//create response channel
	wc := c.lookupConn(connid)
	if wc == nil {
		log.Println("Uops, connection dropped before ServeHTTP!")
		return
	}
	resch := make(chan wsResp)
	r.Id = c.connections.ids.Next()
	wc.Channel.ResChan(r.Id,resch)
	freq := fakeHttpRequest{}
	err := json.Unmarshal([]byte(r.Payload),&freq)
	go func(){
		resp := <- resch
		if resp.Err != nil {
			log.Println("Uops, response error:",r.Id,resp.Err)
		}
		if wc.Channel.ch == nil {
			log.Println("httpChannel closed")
			return
		}
		wc.Channel.ch <- WsHttpMsg{r.Id,false,string(resp.Body)}
	}()
	if err != nil {
		log.Println("http.Request marshal error",err,string(r.Payload))
		//send err through chan
		c.DispatchResponse(nil,err,connid,r.Id)
		return
	}
	req,err := convert(c.Host,&freq)
	if err != nil {
		log.Println("Cannot convert req to http.Request",err)
		c.DispatchResponse(nil,err,connid,r.Id)
		return
	}
	if req.Host==c.Host {
		log.Println("Local call")
		//dispatch locally
		// if strings.HasPrefix(req.URL.Path,"/meta/wsconnections") {
		// 	log.Println("Request to get async notifications about connections")
		// 	c.AddConnectionListener(wc,r.Id)
		// 	c.DispatchResponse([]byte("true"),nil,connid,r.Id)
		// } else {
			rspWriter := makeResponseWriter()//(c)
			c.mux.ServeHTTP(rspWriter,req)
			log.Println("..local call dispatched - ",string(rspWriter.Body()))
			c.DispatchResponse(rspWriter.Body(),rspWriter.Err(),connid,r.Id)
		// }
	} else {
		//make an external call
		log.Println("External call: ",req.Host,c.Host)
		rsp,err := c.makeExternalHttpCall(req)
		c.DispatchResponse(rsp,err,connid,r.Id)
	}
}

func (c *WsHttpCore) makeExternalHttpCall(req *http.Request) ([]byte,os.Error) {
	return nil,os.NewError("External call not implemented yet!")
}

func (c *WsHttpCore) DispatchResponse(rsp []byte,err os.Error,connid string, txid string) {
	// log.Println("Is this fucking thing dispatching or what?")
	wc := c.lookupConn(connid)
	if wc == nil {
		log.Println("Discard result, connection gone")
		return
	}
	rspchan := wc.Channel.For(txid)
	if rspchan == nil {
		log.Println("Response channel gone")
		return
	}
	// var _,ok = <- rspchan
	// if !ok {
	// 	log.Println("Channel was already closed!")
	// } else {
		log.Println("Sending response ",connid,txid,rsp,err)
		defer func() {
			wc.Channel.Remove(txid)
			if x := recover(); x != nil {
				log.Printf("run time panic: %v", x)
			}
		}()
		rspchan <- wsResp{rsp,err}
	// }
	// log.Println("Done, remove resp channel for",txid)
}

func (c *WsHttpCore) lookupConn(id string) *ConnWrapper {
	return c.connections.Get(id)
}
func (c *WsHttpCore) onDisconnect(id string, fnc func()) {
	c.connections.Get(id).AddDisconnectListener(fnc)
}

func getHandlerFor(pattern string,mux *dynamichttp.ServeMux) urlHandler {
	handler := mux.M[pattern]//mux.(map[string]http.Handle)[pattern]
	if handler == nil {
		handler = makeUrlHandler(pattern,mux)
		mux.Handle(pattern,handler)
	}
	return handler.(urlHandler)
}

func makeUrlHandler(pt string,mux *dynamichttp.ServeMux) urlHandler {
	out := urlHandler{pt,mux,make(map[string]map[string]http.Handler)}
	out.handlers["GET"] = make(map[string]http.Handler)
	out.handlers["POST"] = make(map[string]http.Handler)
	out.handlers["PUT"] = make(map[string]http.Handler)
	out.handlers["DELETE"] = make(map[string]http.Handler)
	out.handlers["HEAD"] = make(map[string]http.Handler)
	return out
}
type urlHandler struct {
	pattern string
	mux *dynamichttp.ServeMux
	handlers map[string]map[string]http.Handler
}
func (h *urlHandler) Add(method, connid string,handler http.Handler) {
	if h.handlers[method]==nil {
		log.Println("Create map for method",method)
		h.handlers[method] = make(map[string]http.Handler)
	}
	h.handlers[method][connid] = handler
}
func (h *urlHandler) RemoveAll(connid string,methods []string) {
	if len(methods)==0 {
		for _,meth := range(h.handlers) {
			meth[connid] = nil,false
		}
	} else {
		for _,meth := range(methods) {
			h.handlers[meth][connid] = nil,false
		}
	}
	sum := 0
	for _,m := range(h.handlers) {
		sum += len(m)
	}
	if sum==0 {
		log.Println("No more handlers, removing resource",h.pattern)
		h.mux.Remove(h.pattern)
	}
}
func (h urlHandler) ServeHTTP(w http.ResponseWriter,r *http.Request) {
	for _,fn := range(h.handlers[r.Method]) {
		fn.ServeHTTP(w,r)
	}
}










