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

type sliceMock struct {
	addr uintptr
	len  int
	cap  int
}

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

func toRaftlog(bytes []byte) raftlog {
	newlog := (**raftlog)(unsafe.Pointer(&bytes))
	return **newlog
}
