package sgcache

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"

	//"net/http"
	"time"
	//"unsafe"
)

const (
	leader    = iota //节点为leader
	candidate        //节点为candidate
	follower         //节点为follower
)

type node struct {
	index                 int            //节点的编号
	nodeState             int16          //节点的状态
	nodecache             cache          //节点的主缓存
	nodelogs              []raftlog      //节点的日志
	term                  int            //节点所认为的当前的term
	timeToCandidate       time.Duration  //节点为follower时,经过多长时间变为candidate
	timeDurationFromHeart time.Duration  //收到上次心跳距现在的时间
	hasVote               bool           //是否已投出选票
	numOfAllNode          int            //节点总数
	votesNum              int            //得到选票数
	LeaderIp              string         //Leader 节点的ip
	NodeIp                string         //节点的ip
	IpPool                map[int]string //所有节点的ip
}

func (n *node) isLeader() bool {
	if n.nodeState == 0 {
		return true
	}
	return false
}

func (n *node) isCandidate() bool {
	if n.nodeState == 1 {
		return true
	}
	return false
}

func (n *node) isFollower() bool {
	if n.nodeState == 2 {
		return true
	}
	return false
}

func (n *node) waitForHeart() {
	//TODO
}

func (n *node) send(recvIndex int, msg *raftlog) error {
	var conn net.Conn
	var err error
	if ip, ok := n.IpPool[recvIndex]; !ok {
		return fmt.Errorf("ipPool error")
	} else {
		conn, err = net.Dial("tcp", ip)
	}
	if err != nil {
		return err
	}
	switch msg.logType {
	case justHeartBeat:
		err = n.sendHeartBeatToOneNode(conn, recvIndex, msg)
	}
	return err
}

func (n *node) sendHeartBeatToOneNode(conn net.Conn, recvIndex int, msg *raftlog) error {
	if n.nodeState != leader {
		err := fmt.Errorf("Error: Node is not a leader")
		return err
	}
	_, err := conn.Write(msg.tobytes())
	return err
}

func (n *node) sendHeartBeatToAllNode() error {
	var err error
	for recvIndex, ip := range n.IpPool {
		go func(recvIndex int, ip string) {
			err := n.send(recvIndex, heartbeatlog(n.term, n.index, recvIndex))
			if err != nil {
				err = err
			}
		}(recvIndex, ip)
	}
	return err
}

func heartbeatlog(term int, senderIndex int, recvIndex int) *raftlog {
	log := raftlog{
		term:             term,
		logType:          justHeartBeat,
		LogSenderIndex:   senderIndex,
		LogReceiverIndex: recvIndex,
	}
	return &log
}

func (n *node) Step() error {
	if n.isLeader() == true {
		for {
			n.sendHeartBeat()
			//TODO : time.wait(100ms)
		}
	}

}

func InitNode() *node {
	sourseNum := int64(time.Now().Nanosecond())
	sourse := rand.NewSource(sourseNum)
	randSeed := rand.New(sourse)
	n := &node{
		nodeState:       follower,
		nodelogs:        make([]raftlog, 0),
		timeToCandidate: time.Duration(randSeed.Int31n(150) + 100),
	}
	return n
}

func (n *node) Accept() error {
	listener, err := net.Listen("tcp", ":2803")
	if err != nil {
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go serveConn(conn, n)
	}
}

func serveConn(conn net.Conn, n *node) {
	req := bufio.NewReader(conn)
	msgBytes := make([]byte, 40)
	req.Read(msgBytes)
	msg := toRaftlog(msgBytes)
	switch msg.logType {
	case justHeartBeat:
		n.handlejustHeartBeat(&msg)
	case getVoted:
		n.handleGetVoted(&msg)
	case msgAndHeartBeat:
		n.handleMsgAndHeartBeat(&msg)
	case voteToOther:
		n.handleVoteToOther(&msg)
	}
	return
}

func (n *node) handlejustHeartBeat(msg *raftlog) {
	if msg.logType != justHeartBeat {
		panic("something wrong with message")
	}
	if n.term == msg.term {
		n.timeDurationFromHeart = 0
	} else if n.term < msg.term {
		n.term = msg.term
	}
	return
}

func (n *node) handleGetVoted(msg *raftlog) {
	if msg.logType != getVoted {
		panic("something wrong with message")
	}
	if n.hasVote == true {
		return
	}
	_ = n.send(msg.LogSenderIndex, &raftlog{
		term:             n.term,
		logType:          voteToOther,
		LogSenderIndex:   n.index,
		LogReceiverIndex: msg.LogSenderIndex,
	})
}

func (n *node) handleMsgAndHeartBeat(msg *raftlog) {
	//TODO
}

func (n *node) handleVoteToOther(msg *raftlog) {
	if msg.logType != voteToOther {
		panic("Something wrong with message")
	}
	n.votesNum++
	if n.votesNum > n.numOfAllNode/2 {
		n.beLeader()
	}
}

func (n *node) beLeader() {
	//TODO
}

func (n *node) beFollower() {
	//TODO
}

func (n *node) beCandidate() {
	//TODO
}
