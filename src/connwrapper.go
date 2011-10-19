package wshttp

import (
	"websocket"
	"os"
//	"json"
	"http"
	"log"
	"fmt"
	"container/list"
)

type naiveIdServer struct {
	v *int64
}
func (n naiveIdServer) Next() string {
	val := *(n.v)+1
	*(n.v) = val
	out := fmt.Sprint(val)
	log.Println("return val",val,out)
	return out
}

type IdServer interface {
	Next() string
}

type ConnEventType int
func (c ConnEventType) Connected() bool {return c==1}
func (c ConnEventType) Disconnected() bool {return c==0}

const Connected = ConnEventType(1)
const Disconnected = ConnEventType(0)

func MakeDefaultConnections() Connections {
	v := int64(0)
	return Connections{naiveIdServer{&v},make(map[string]*ConnWrapper),list.New()} //[]func(ConnEventType,*ConnWrapper){}}
}

type Connections struct {
	ids IdServer
	m map[string]*ConnWrapper //I suppose this is by no means thread safe?
	listeners *list.List//[]func(ConnEventType,*ConnWrapper)
}
func (c *Connections) sendEvent(t ConnEventType,cw *ConnWrapper) {
	log.Println("Connections.sendEvent ",t,cw.Id)
	for el := c.listeners.Front(); el != nil; el=el.Next() {
		el.Value.(func (ConnEventType,*ConnWrapper))(t,cw)
	}
	// for _,fn := range(c.listeners) {
	// 	fn(t,cw)
	// }
}
func (c *Connections) Register(w *ConnWrapper) {
	c.m[w.Id] = w
	c.sendEvent(Connected,w)
}
func (c *Connections) Get(id string) *ConnWrapper {
	return c.m[id]
}
func (c *Connections) Remove(id string) (out *ConnWrapper) {
	out = c.m[id]
	out.Channel.Close()
	c.m[id] = nil,false
	c.sendEvent(Disconnected,out)
	return
}
func (c *Connections) RegisterConn(w *websocket.Conn) (out *ConnWrapper) {
	//out = ConnWrapper{c,w,c.ids.Next(),[]func(){},nil}
	out = &ConnWrapper{
		conns: c,
		Conn: w,
		Id: c.ids.Next(),
		DisconnectListeners: []func(){},
	}
	SetHttpChannel(out)
	c.Register(out)
	return
}

func (c *Connections) AddListener(fn func(ConnEventType,*ConnWrapper)) []string {
	c.listeners.PushBack(fn)
	var curr = make([]string,len(c.m))
	var id=0
	for k,_ := range(c.m) {
		curr[id]=k//"{'connid':'"+k+"'}"
		id++
	}
//	bytes,err := json.Marshal(curr)
	return curr
}
func (c *Connections) RemoveListener(fn func(ConnEventType,*ConnWrapper)) {
	for el := c.listeners.Front(); el != nil; el=el.Next() {
		if el.Value == fn {
			log.Println("Found listener")
			c.listeners.Remove(el)
			return
		}
	}
}

type ConnWrapper struct {
	conns *Connections
	Conn *websocket.Conn
	Id string
	DisconnectListeners []func()
	Channel httpChannel
}

func (cw *ConnWrapper) AddDisconnectListener(f func()) {
	cw.DisconnectListeners = append(cw.DisconnectListeners,f)
}

func (cw *ConnWrapper) HTTP(req *http.Request) ([]byte,os.Error) {
	id := cw.conns.ids.Next()
	rsp,err := cw.Channel.SendRcv(id,req)
	if err != nil {
		log.Println("cw.Channel.SendRcv error",err)
		return nil,err
	}
	return []byte(rsp),nil
}


func makeResponseWriter() wsRespWriter{//http.ResponseWriter {
	b := []byte{}
	code := 200
	return wsRespWriter{&b,&code,nil,http.Header{}}
}

type wsRespWriter struct {
	body *[]byte
	code *int
	err os.Error
	header http.Header
}
func (r *wsRespWriter) Body() []byte {
	return *(r.body)
}
func (r *wsRespWriter) Err() os.Error {
	return r.err
}
func (r wsRespWriter) Header() http.Header {
	return r.header
}
func (r wsRespWriter) Write(p []byte) (int,os.Error) {
	*(r.body) = append(*(r.body),p...)
	return len(p),nil
	// fmt.Println("write ",string(p))
	// *(r.body) = *(r.body) + string(p)
//	b := r.body
//	newb := (*b) + string(p)
//	r.body = &newb
	// return len(p),nil
}
func (r wsRespWriter) WriteHeader(code int) {
//	fmt.Println("set code",code)
	*(r.code) = code
}





