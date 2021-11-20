package sgcache

import (
	"unsafe"
)

const (
	justHeartBeat   = iota //只有心跳
	voteToOther            //发送选票
	getVoted               //给别的节点发起得到选票的请求
	msgAndHeartBeat        //心跳和事务
)

type raftlog struct {
	term             int    //日志产生时的任期
	logType          int    //事务类型
	msg              string //消息体
	index            int    //日志索引
	LogSenderIndex   int    //发送节点索引
	LogReceiverIndex int    //接受节点索引
}

//消息序列化中间结构体,模拟byte切片底层
type sliceMock struct {
	addr uintptr
	len  int
	cap  int
}

//反序列化为字节切片
func (r *raftlog) tobytes() []byte {
	len := unsafe.Sizeof(*r)
	bytes := &sliceMock{
		addr: uintptr(unsafe.Pointer(r)),
		cap:  int(len),
		len:  int(len),
	} //构造bytes切片底层
	data := *(*[]byte)(unsafe.Pointer(bytes)) //将构造的切片的指针转化为byte切片的指针
	return data
}

//序列化为日志
func toRaftlog(bytes []byte) raftlog {
	newlog := (**raftlog)(unsafe.Pointer(&bytes))
	return **newlog
}

//创建心跳日志
func (n *node) heartbeatlog(recv int) *raftlog {
	log := raftlog{
		term:             n.term,
		logType:          justHeartBeat,
		LogSenderIndex:   n.index,
		LogReceiverIndex: recv,
	}
	return &log
}

//创建请求被选举的日志
func (n *node) getVotedlog(recv int) *raftlog {
	log := &raftlog{
		term:             n.term,
		logType:          getVoted,
		index:            n.nodelogs[len(n.nodelogs)-1].index,
		LogSenderIndex:   n.index,
		LogReceiverIndex: recv,
	}
	return log
}
