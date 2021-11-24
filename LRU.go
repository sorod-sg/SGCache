package sgcache

import (
	"container/list"
	//"fmt"
)

type LRUqueen struct {
	list     *list.List               //核心队列,队首的元素先被淘汰
	cache    map[string]*list.Element //存储健和值的对应关系,方便查找
	maxBytes int64
	nBytes   int64
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

//实现了Len方法的byte切片
type ByteView struct {
	b []byte
}

func (v ByteView) Len() int {
	return len(v.b)
}

//t提取byteview中的切片
func (v *ByteView) ByteSlice() []byte {
	var bClone = make([]byte, len(v.b))
	copy(bClone, v.b)
	return bClone
}

func (q *LRUqueen) NBytes() int64 {
	return q.nBytes
}

func InitLRUqueen(maxBytes int64) *LRUqueen {
	return &LRUqueen{
		list:     list.New(),
		cache:    make(map[string]*list.Element),
		maxBytes: maxBytes,
		nBytes:   0,
	}
}

func (e *entry) Len() int {
	return len(e.key) + e.value.Len()
}

func (q *LRUqueen) Push(key string, value Value) {
	if ele, ok := q.cache[key]; ok {
		q.list.MoveToBack(ele)
		oldlen := ele.Value.(*entry).value.Len()
		ele.Value.(*entry).value = value
		q.nBytes += int64(value.Len()) - int64(oldlen)
	} else {
		newEntry := &entry{
			key:   key,
			value: value,
		}
		newElement := q.list.PushBack(newEntry)
		q.cache[newEntry.key] = newElement
		q.nBytes += int64(newEntry.Len())
	}
	if q.nBytes >= q.maxBytes {
		for {
			if q.nBytes >= q.maxBytes {
				q.Delete(q.list.Front().Value.(*entry).key)
			} else {
				break
			}
		}
	}
	return
}

func (q *LRUqueen) Get(key string) (value interface{}, isExist bool) {
	if ele, ok := q.cache[key]; ok {
		q.list.MoveToBack(ele)
		value = ele.Value
		isExist = true
		return
	} else {
		value = nil
		isExist = false
		return
	}
}

func (q *LRUqueen) Delete(key string) {
	if ele, ok := q.cache[key]; ok {
		q.list.Remove(ele)
		delete(q.cache, key)
		q.nBytes -= int64(ele.Value.(*entry).Len())
	}
	return
}
