package main

import (
	"fmt"
	"sgcache"
)

type T struct {
	a int
}

func (t *T) Len() int {
	return t.a
}

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	var f sgcache.Getter = sgcache.GetterFunc(func(key string) ([]byte, error) {
		return []byte(db[key]), nil
	})
	g1 := sgcache.NewGroup("1", 10000, f)
	a, err := g1.Get("Tom")
	if err != nil {
		fmt.Println("error for get g1")
	}
	fmt.Println(a.ByteSlice())

}
