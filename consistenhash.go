package sgcache

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

type Map struct {
	hash     Hash           //使用的哈希函数
	replicas int            //虚拟节点倍数
	keys     []int          //哈希环
	hashMap  map[int]string //虚拟节点和真正节点的映射表
}

func NewMap(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))

	idxRight := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	idxLeft := sort.Search((len(m.keys)), func(i int) bool {
		return m.keys[i] <= hash
	})
	idxMax := m.keys[len(m.keys)-1]
	var sortIntSlice sort.IntSlice
	sortIntSlice = make(sort.IntSlice, 0)
	sortIntSlice = append(sortIntSlice, absInt(hash-idxLeft), absInt(hash-idxRight), absInt(hash-idxMax))
	sort.Ints(sortIntSlice)
	idx := sortIntSlice[0]
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
