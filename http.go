package sgcache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

type HTTPPool struct {
	self     string
	basePath string
}

type message struct {
	message string //path + group + key
}

func (c *client) buildMessage(path string, group string, key string) message {
	var msg = message{
		message: fmt.Sprintln(path, group, "/", key),
	}
	return msg
}

const defaultBasePath = "/sgcache/"

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) ServerHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		log.Println("Unexpected path:", r.URL.Path)
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	groupName := parts[1]
	key := parts[2]
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group:"+groupName, http.StatusNotFound)
		return
	}
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())

}
