package sgcache

import (
	"bufio"
	"fmt"
	"log"
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

//判断节点是否为Leader
func (n *node) isLeader() bool {
	if n.nodeState == 0 {
		return true
	}
	return false
}

//判断节点是否为Candidate
func (n *node) isCandidate() bool {
	if n.nodeState == 1 {
		return true
	}
	return false
}

//判断节点是否为Follower
func (n *node) isFollower() bool {
	if n.nodeState == 2 {
		return true
	}
	return false
}

//发送消息给其他节点,参数为接收者的编号
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
	_, err = conn.Write(msg.tobytes())
	return err
}

/*
func (n *node) sendHeartBeatToOneNode(recvIndex int) error {
	if n.nodeState != leader {
		err := fmt.Errorf("Error: Node is not a leader")
		return err
	}
	msg := n.heartbeatlog(recvIndex)
	n.send(recvIndex , msg)
	return err
}*/

//向所有节点发出心跳
func (n *node) sendHeartBeatToAllNode() error {
	var err error
	for recvIndex, _ := range n.IpPool {
		go func(recvIndex int) {
			err := n.send(recvIndex, n.heartbeatlog(recvIndex))
			if err != nil {
				err = err
			}
		}(recvIndex)
	}
	return err
}

//初始化节点
func InitNode(index int, numOfAllNode int, LeaderIp string, NodeIP string, IpPool map[int]string, maxBytes int64) *node {
	sourseNum := int64(time.Now().Nanosecond())
	sourse := rand.NewSource(sourseNum)
	randSeed := rand.New(sourse)
	n := &node{
		index:     index,
		nodeState: follower,
		nodecache: cache{
			lru:        InitLRUqueen(maxBytes),
			cacheBytes: maxBytes,
		},
		nodelogs:              make([]raftlog, 0),
		timeToCandidate:       time.Duration(randSeed.Int31n(150) + 100),
		timeDurationFromHeart: 0,
		numOfAllNode:          numOfAllNode,
		LeaderIp:              LeaderIp,
		NodeIp:                NodeIP,
		IpPool:                IpPool,
	}
	return n
}

//节点开始接收信息
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

//处理链接,并根据消息类型回复
func serveConn(conn net.Conn, n *node) {
	req := bufio.NewReader(conn)
	msgBytes := make([]byte, 40)
	req.Read(msgBytes)
	msg := toRaftlog(msgBytes)
	switch msg.logType {
	case justHeartBeat:
		n.handlejustHeartBeat(&msg)
	case voteToOther:
		n.handleVoteToOther(&msg)
	case getVoted:
		n.handleGetVoted(&msg)
	case msgAndHeartBeat:
		n.handleMsgAndHeartBeat(&msg)
	case clientGet:
		n.handleClientGet(&msg)
	case clientAddNode:
		n.handleAddNode(&msg)
	}
	conn.Close()
	return
}

//消息处理

//处理简单心跳
func (n *node) handlejustHeartBeat(msg *raftlog) {
	if msg.logType != justHeartBeat {
		panic("something wrong with message")
	}
	if n.term == msg.term {
		n.timeDurationFromHeart = 0
	} else if n.term < msg.term {
		n.term = msg.term
	} else {
		//TODO tell leader to be sender
	}
	return
}

//处理受到被要求去投票的请求
func (n *node) handleGetVoted(msg *raftlog) {
	if msg.logType != getVoted {
		panic("something wrong with message")
	}
	if n.hasVote == true {
		return
	}
	if msg.index < n.nodelogs[len(n.nodelogs)-1].index {
		return
	}
	_ = n.send(msg.LogSenderIndex, &raftlog{
		term:             n.term,
		logType:          voteToOther,
		LogSenderIndex:   n.index,
		LogReceiverIndex: msg.LogSenderIndex,
	})
	n.hasVote = true
}

//处理心跳和msg
func (n *node) handleMsgAndHeartBeat(msg *raftlog) {
	//TODO
}

//处理受到的投票
func (n *node) handleVoteToOther(msg *raftlog) {
	if msg.logType != voteToOther {
		panic("Something wrong with message")
	}
	n.votesNum++
	if n.votesNum > n.numOfAllNode/2 {
		n.beLeader()
	}
}

func (n *node) handleClientGet(msg *raftlog) {
	//TODO
}

func (n *node) handleAddNode(msg *raftlog) {
	//TODO
}

//节点状态转化

//节点转化为Leader,并开始发送心跳
func (n *node) beLeader() {
	if n.nodeState != candidate {
		panic("nodeState error")
	}
	n.nodeState = leader
	n.sendHeartBeat()
}

//节点变为follower
func (n *node) beFollower() {
	n.nodeState = follower
}

//节点变为candidate
func (n *node) beCandidate() {
	n.nodeState = candidate
	n.term++
	n.sendVotedToAll()

}

//发送消息

//发送心跳
func (n *node) sendHeartBeat() {
	go func() {
		for {
			err := n.sendHeartBeatToAllNode()
			if err != nil {
				log.Println(err)
			}
			time.Sleep(time.Nanosecond * 150)
			if n.nodeState != leader {
				return
			}
		}
	}()

}

//向所有人发送被选举请求
func (n *node) sendVotedToAll() error {
	var err error
	for recvIndex, _ := range n.IpPool {
		go func(recvIndex int) {
			err := n.send(recvIndex, n.getVotedlog(recvIndex))
			if err != nil {
				err = err
			}
		}(recvIndex)
	}
	return err
}
