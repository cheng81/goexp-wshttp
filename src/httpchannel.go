package wshttp

import (
	"json"
	"os"
	"log"
	"http"
//	"websocket"
)

func SetHttpChannel(cw *ConnWrapper) {
	ch := make(chan WsHttpMsg)
	cw.Channel = httpChannel{ch,make(map[string]chan wsResp)}
	go func() {
		for wsreq := range(ch) {
			go func() {
				bytes,err := json.Marshal(wsreq)
				if err != nil {
					log.Println("json.Marshal",err)
					return
				}
				log.Println("Writing on the wire")
				n,werr := cw.Conn.Write(bytes)
				log.Println("Wrote ",n," out of ",len(bytes))
				if werr != nil {
					log.Println("Error writing to websocket",werr)
				}
			}()
		}
	}()
}

// func MakeChanWriter(cw *ConnWrapper) ChanWriter {
// 	ch := make(chan []byte)
// 	out := ChanWriter{cw,ch}
// 	return out
// }
// type ChanWriter struct {
// 	cw *ConnWrapper
// 	ch chan []byte
// }
// func (c *ChanWriter) Write(p []byte) (int,os.Error) {
// 	c.ch <- p
// 	return len(p),nil
// }
// func (c *ChanWriter) Close() {
// 	close(c.ch)
// }


type WsHttpMsg struct {
	Id string
	IsReq bool
	Payload string
}
type wsResp struct {
	Body []byte
	Err os.Error
}
type httpChannel struct {
	ch chan WsHttpMsg
	resps map[string]chan wsResp//([]byte,os.Error)
}
func (c *httpChannel) SendRcv(id string, r *http.Request) ([]byte,os.Error) {
	jr,err := json.Marshal(r)
	if err != nil {
		log.Println("Error marshaling request",err)
		return nil,err
	}
	log.Println("Marshalled request",string(jr))
	wsr := WsHttpMsg{id,true,string(jr)}
//	jwsr,err1 := json.Marshal(wsr)
//	if err != nil {
//		return nil,err
//	}
	resch := make(chan wsResp)//([]byte,os.Error))
	c.ResChan(id,resch)
	go func() {c.ch <- wsr}()
	log.Println("awaiting response..",id)
	resp := <- resch
	log.Println("..got response!",id,string(resp.Body))
	return resp.Body,resp.Err
}
func (c *httpChannel) Close() {
	log.Println("Closing httpChannel")
	close(c.ch)
	c.ch = nil
	for k,rc := range(c.resps) {
		close(rc)
		c.resps[k] = nil,false
	}

}
func (c *httpChannel) Remove(id string) {
	c.resps[id] = nil,false
	log.Println("Removing response channel",id,c.resps[id])
}
func (c *httpChannel) ResChan(id string, ch chan wsResp) {//([]byte,os.Error)) {
	c.resps[id] = ch
}
func (c *httpChannel) For(id string) chan wsResp{ //([]byte,os.Error) {
	return c.resps[id]
}