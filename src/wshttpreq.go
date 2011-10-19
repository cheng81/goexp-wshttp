package wshttp

import (
	"http"
	"url"
	"os"
	"log"
)

type fakeHttpRequest struct {
	Method string
	URL string
//	Body []byte
	Body string
}

type byteReadCloser []byte
func (f byteReadCloser) Read(p []byte) (read int,err os.Error) {
	log.Println("Trying to read body - ",len(p),len(f))
	lenP := len(p)
	lenF := len(f)
	for read=0;read<lenP && read<lenF;read++ {
		p[read] = f[read]
	}
	log.Println("done",read)
	return
}
func (f byteReadCloser) Close() os.Error {
	return nil
}

func convert(host string,r *fakeHttpRequest) (*http.Request,os.Error) {
	header := http.Header{}
	header.Add("User-Agent","Mozilla/5.0 (Macintosh; Intel Mac OS X 10_6_8) AppleWebKit/535.1 (KHTML, like Gecko) Chrome/14.0.835.202 Safari/535.1")
	header.Add("Accept","text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	theurl,err := url.ParseRequest(r.URL)
	if err != nil {
		log.Println("Error in url: ",err)
		return nil,err
	}

	out := &http.Request{
		Method: r.Method,
		URL: theurl,
		Proto: "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: header,
//		Body: byteReadCloser([]byte(r.Body)),
		ContentLength: int64(len(r.Body)),
		TransferEncoding: []string{},
		Close: true,
		Host: host,
		Form: nil,
		MultipartForm: nil,
		Trailer: nil,
		RemoteAddr: "fake:req",
		TLS: nil,
	}
	if len(r.Body) == 0 {
		out.Body=nil
	} else {
		out.Body=byteReadCloser([]byte(r.Body))
	}
	return out,nil
}