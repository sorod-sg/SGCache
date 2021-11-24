package sgcache

import (
	"fmt"
	"net"
	"time"
)

type client struct {
	IpPool         []string
	LeaderIp       string
	ConnectTimeout time.Duration
}

func (c *client) GET(key string) (value ByteView, isExist bool, err error) {
	for i := 1; i < 10; i++ {
		value, isExist, err = c.sendMsgGET(key)
		if err == nil || err != fmt.Errorf("Timeout") {
			break
		}
		if i == 10 {
			value = ByteView{}
			isExist = false
			err = fmt.Errorf("Timeout to many times")
		}
	}
	return
}

func InitClient(ConnectTimeOut time.Duration) *client {
	return &client{
		IpPool:         make([]string, 10, 10),
		ConnectTimeout: ConnectTimeOut,
	}
}

func (c *client) addNode(nodeIp []string) (err error) {
	c.IpPool = append(c.IpPool, nodeIp...)
	for i := 1; i < 10; i++ {
		if c.LeaderIp == "" {
			c.LeaderIp = c.IpPool[i-1]
		}
		err = c.sendMsgNodeAdd(nodeIp)
		if err == nil || err != fmt.Errorf("Timeout") {
			break
		}
		if i == 10 {
			err = fmt.Errorf("Timeout to many times")
		}
	}
	return err
}

func (c *client) sendMsgNodeAdd(nodeIp []string) error {
	sendMsg := ClientAddNode(nodeIp)
	recvMsg := &raftlog{}
	err := c.send(c.LeaderIp, sendMsg, recvMsg)
	if recvMsg.logType != replyClientAddNode {
		return fmt.Errorf("reply log error")
	}
	return err
}

func (c *client) sendMsgGET(key string) (value ByteView, isExist bool, err error) {
	sendMsg := ClientGet(key)
	recvMsg := &raftlog{}
	err = c.send(c.LeaderIp, sendMsg, recvMsg)
	if err != nil {
		value = ByteView{}
		isExist = false
		return
	}
	value = ByteView{
		b: []byte(recvMsg.msg),
	}
	isExist = true
	return
}

func (c *client) send(recvIp string, sendMsg *raftlog, revcMsg *raftlog) (err error) {
	conn, err := net.Dial("tcp", recvIp)
	if err != nil {
		return err
	}
	conn.Write(sendMsg.tobytes())
	conn.Close()
	listener, err := net.Listen("tcp", recvIp)
	if err != nil {
		return err
	}
	done := make(chan int)
	go func(listener net.Listener, done chan int, msg *raftlog, err *error) {
		conn, er := listener.Accept()
		if er != nil {
			*err = er
			return
		}
		_, er = conn.Read(msg.tobytes())
		if er != nil {
			*err = er
			return
		}
	}(listener, done, revcMsg, &err)
	select {
	case <-done:
		err = nil
		return
	case <-time.After(c.ConnectTimeout):
		return fmt.Errorf("Timeout")
	}
}
