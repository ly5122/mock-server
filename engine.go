package main

import (
	"errors"
	"net/http"
	"strings"

	"bytes"
	"encoding/base64"
	"fmt"
	logger "github.com/jbrodriguez/mlog"
	"io"
	"io/ioutil"
	"strconv"
)

const (
	INTERNAL = "/internal"
	POST     = "POST"
)

type (
	engine struct {
		regions map[string]routeTable
	}
	routeTable map[string]*route
	route      struct {
		method string
		path   string
		cmd    string
		hf     handleFunc
		hr     []histroyRaw
	}
	histroyRaw struct {
		queryRaw string
		bodyRaw  string
	}
	handleFunc func(w http.ResponseWriter, r *http.Request)
)

var (
	err404 = errors.New("404")
)

func newEngine() (e *engine) {
	return &engine{regions: make(map[string]routeTable)}
}

func (e *engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, INTERNAL) {
		e.process(e.regions[INTERNAL], w, r)
	} else {
		e.process(e.regions[e.getRequestKey(r)], w, r)
	}
}

func (e *engine) getRequestKey(r *http.Request) (mf string) {
	mf = r.Header.Get("Mock-From")
	if mf == "" {
		mf = r.RemoteAddr[:strings.Index(r.RemoteAddr, ":")]
	}
	return
}

func (e *engine) process(rt routeTable, w http.ResponseWriter, r *http.Request) {
	if rt != nil {
		for key, value := range rt {
			if key == e.createKey(r.Method, r.URL.Path) {
				if value.hr == nil {
					value.hr = make([]histroyRaw, 0)
				}
				var buf bytes.Buffer
				ct := r.Header.Get("Content-Type")
				switch {
				case ct == "application/x-www-form-urlencoded":
					r.ParseForm()
					for key, value := range r.PostForm {
						prefix := key + "="
						for _, v := range value {
							if buf.Len() > 0 {
								buf.WriteByte('&')
							}
							buf.WriteString(prefix)
							buf.WriteString(v)
						}
					}
				default:
					var reader io.Reader = r.Body
					b, _ := ioutil.ReadAll(reader)
					buf.Write(b)
				}
				tempString := "body too big"
				if(buf.Len() <= maxSize) {
					tempString = buf.String()
				}
				logger.Info("path:%v, queryRaw:%v, bodyRaw:%v", r.URL.Path, r.URL.RawQuery, tempString)
				value.hr = append(value.hr, histroyRaw{
					queryRaw: r.URL.RawQuery,
					bodyRaw:  buf.String(),
				})
				//匹配情况下有handleFunc就执行，没有就解析cmd
				if value.hf != nil {
					value.hf(w, r)
				} else {
					e.execCmd(value.cmd, w, r)
				}
				return
			}
		}
	}
	e.renderErr(err404, w, r)
}

func (e *engine) execCmd(cmd string, w http.ResponseWriter, r *http.Request) {
	// 有可能cmd写的顺序有问题导致返回的response有问题，最好能最后一次性写io
	b, _ := base64.StdEncoding.DecodeString(cmd)
	cmds := strings.Split(string(b), "\n")
	var header = make(map[string]string, 0)
	var code = 200
	var body []byte
	for _, s := range cmds {
		s = e.trim(s)
		tag := strings.Split(s, " ")
		if len(tag) <= 0 {
			continue
		}
		switch {
		case strings.HasPrefix(tag[0], "res_header"):
			if len(tag) != 3 {
				e.writeResponse(w, 400, "cmd res_header foramt error")
				break
			}
			header[tag[1]] = tag[2]
		case strings.HasPrefix(tag[0], "res_code"):
			if len(tag) != 2 {
				e.writeResponse(w, 400, "cmd res_code foramt error")
				break
			}
			c, err := strconv.Atoi(tag[1])
			if err != nil {
				e.writeResponse(w, 400, "cmd res_code foramt error")
			}
			code = c
		case strings.HasPrefix(tag[0], "res_body"):
			if len(tag) != 2 {
				e.writeResponse(w, 400, "cmd res_body foramt error")
				break
			}
			body = []byte(tag[1])
		}
	}
	for key, value := range header {
		w.Header().Set(key, value)
	}
	w.WriteHeader(code)
	w.Write(body)
}

func (e *engine) trim(s string) string {
	length := len(s)
	index := 0
	for index < length && s[index] <= ' ' {
		index++
	}
	for index < length && s[length-1] <= ' ' {
		length--
	}
	return s[index:length]
}

func (e *engine) renderErr(err error, w http.ResponseWriter, r *http.Request) {
	if err == err404 {
		w.WriteHeader(404)
		w.Write([]byte(err404.Error()))
	}
}

func (e *engine) init() {
	rt := make(routeTable, 0)
	e.addInternalRoute(&rt, INTERNAL+"/add", e.add)
	e.addInternalRoute(&rt, INTERNAL+"/remove", e.remove)
	e.addInternalRoute(&rt, INTERNAL+"/removeAll", e.removeAll)
	e.addInternalRoute(&rt, INTERNAL+"/histroy", e.histroy)
	e.addInternalRoute(&rt, INTERNAL+"/clearHistroy", e.clearHistroy)
	e.regions[INTERNAL] = rt
}

func (e *engine) addInternalRoute(rt *routeTable, path string, hf handleFunc) {
	(*rt)[e.createKey(POST, path)] = &route{
		method: POST,
		path:   path,
		hf:     hf,
	}
}

func (e *engine) add(w http.ResponseWriter, r *http.Request) {
	key := e.getRequestKey(r)
	if _, ok := e.regions[key]; !ok {
		e.regions[key] = make(routeTable, 0)
	}
	method, okMethod := r.PostForm["method"]
	path, okPath := r.PostForm["path"]
	cmd, okCmd := r.PostForm["cmd"]
	if !okMethod || !okPath || !okCmd {
		e.writeResponse(w, 1, "param invalid")
		return
	}
	rt := e.regions[key]
	e.createRoute(rt, method[0], path[0], cmd[0])
	e.writeResponse(w, 0, "")
}

func (e *engine) remove(w http.ResponseWriter, r *http.Request) {
	key := e.getRequestKey(r)
	if _, ok := e.regions[key]; !ok {
		e.writeResponse(w, 2, "route not exist")
		return
	}
	method, okMethod := r.PostForm["method"]
	path, okPath := r.PostForm["path"]
	if !okMethod || !okPath {
		e.writeResponse(w, 1, "param invalid")
		return
	}
	e.removeRoute(e.regions[key], method[0], path[0])
	e.writeResponse(w, 0, "")
}

func (e *engine) removeAll(w http.ResponseWriter, r *http.Request) {
	key := e.getRequestKey(r)
	delete(e.regions, key)
	e.writeResponse(w, 0, "")
}

func (e *engine) histroy(w http.ResponseWriter, r *http.Request) {
	key := e.getRequestKey(r)
	if _, ok := e.regions[key]; !ok {
		e.writeResponse(w, 3, "history not exist")
		return
	}
	method, okMethod := r.PostForm["method"]
	path, okPath := r.PostForm["path"]
	if !okMethod || !okPath {
		e.writeResponse(w, 1, "param invalid")
		return
	}
	rt := e.regions[key]
	route, ok := rt[e.createKey(method[0], path[0])]
	if !ok {
		e.writeResponse(w, 3, "history not exist")
		return
	}
	var s string = "["
	for i, hr := range route.hr {
		if i != 0 {
			s += ","
		}
		s += fmt.Sprintf(
			"{\"queryRaw\":\"%v\",\"bodyRaw\":\"%v\"}",
			base64.StdEncoding.EncodeToString([]byte(hr.queryRaw)),
			base64.StdEncoding.EncodeToString([]byte(hr.bodyRaw)),
		)
	}
	s += "]"
	w.Write([]byte(fmt.Sprintf("{\"status\":0,\"histroy\":%s}", s)))
}

func (e *engine) clearHistroy(w http.ResponseWriter, r *http.Request) {
	key := e.getRequestKey(r)
	if _, ok := e.regions[key]; !ok {
		e.writeResponse(w, 3, "history not exist")
		return
	}
	method, okMethod := r.PostForm["method"]
	path, okPath := r.PostForm["path"]
	if !okMethod || !okPath {
		e.writeResponse(w, 1, "param invalid")
		return
	}
	rt := e.regions[key]
	route, ok := rt[e.createKey(method[0], path[0])]
	if !ok {
		e.writeResponse(w, 3, "history not exist")
		return
	}
	route.hr = nil
	e.writeResponse(w, 0, "")
}

func (e *engine) writeResponse(w http.ResponseWriter, status int, msg string) {
	var code int
	if status == 0 {
		code = 200
		if msg == "" {
			msg = "success"
		}
	} else {
		code = 400
	}
	w.WriteHeader(code)
	w.Write([]byte(fmt.Sprintf("{\"status\":%v,\"msg\":\"%s\"}", status, msg)))
}

func (e *engine) removeRoute(rt routeTable, method string, path string) {
	delete(rt, e.createKey(method, path))
}

func (e *engine) createRoute(rt routeTable, method string, path string, cmd string) {
	rt[e.createKey(method, path)] = &route{
		method: method,
		path:   path,
		cmd:    cmd,
	}
}

func (e *engine) createKey(method string, path string) string {
	return method + "@" + path
}
