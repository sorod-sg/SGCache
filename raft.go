package sgcache

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
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
	index                 int           //节点的编号
	nodeState             int16         //节点的状态
	nodecache             cache         //节点的主缓存
	nodelogs              []raftlog     //节点的日志
	term                  int           //节点所认为的当前的term
	timeToCandidate       time.Duration //节点为follower时,经过多长时间变为candidate
	timeDurationFromHeart time.Duration //收到上次心跳距现在的时间
	hasVote               bool          //是否已投出选票
	numOfAllNode          int           //节点总数
	votesNum              int           //得到选票数
	LeaderIp              string        //Leader 节点的ip
	LeaderIndex           int           //Leader 节点的编号
	NodeIp                string        //节点的ip
	clientIp              string
	IpPool                []string //所有节点的ip
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

//获取即将加入日志表的日志编号
func (n *node) nextNodelogIndex() int {
	return n.nodelogs[len(n.nodelogs)-1].index + 1
}

//将日志加入日志表
func (n *node) addNodeLog(newLog *raftlog) {
	n.nodelogs = append(n.nodelogs, *newLog)
}

//发送消息给其他节点,参数为接收者的编号
func (n *node) sendToNode(recvIndex int, msg *raftlog) error {
	var conn net.Conn
	var err error
	if ip := n.IpPool[recvIndex]; ip != "" {
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

func (n *node) sendToClient(msg *raftlog) error {
	conn, err := net.Dial("tcp", n.clientIp)
	if err != nil {
		return err
	}
	_, err = conn.Write(msg.tobytes())
	return err
}

//向所有节点发出心跳
func (n *node) sendHeartBeatToAllNode() error {
	var err error
	for recvIndex, _ := range n.IpPool {
		go func(recvIndex int) {
			err := n.sendToNode(recvIndex, n.heartbeatlog(recvIndex))
			if err != nil {
				err = err
			}
		}(recvIndex)
	}
	return err
}

func (n *node) sendAddNodeToAllNode() error {
	var err error
	for recvIndex, _ := range n.IpPool {
		go func(recvIndex int) {
			err := n.sendToNode(recvIndex, n.addNodelog(recvIndex))
			if err != nil {
				err = err
			}
			//TODO 2 pc
		}(recvIndex)
	}
	return err
}

//TODO 封装一个sendToAll,重写所有的森的ToAll相关方法

func (n *node) sendDoneToLeader(msg *raftlog) error {

}

//初始化节点
func InitNode(index int, numOfAllNode int, LeaderIp string, NodeIP string, IpPool []string, maxBytes int64) *node {
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
	case addNode:
		n.handleAddNode(&msg)
	case clientGet:
		n.clientIp = conn.RemoteAddr().String()
		n.handleClientGet(&msg)
	case clientAddNode:
		n.clientIp = conn.RemoteAddr().String()
		n.handleClientAddNode(&msg)
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
		n.timeDurationFromHeart = 0
		if n.LeaderIndex != msg.LogSenderIndex {
			n.LeaderIndex = msg.LogSenderIndex
			n.LeaderIp = n.IpPool[msg.LogSenderIndex]
		}
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
	_ = n.sendToNode(msg.LogSenderIndex, &raftlog{
		term:             n.term,
		logType:          voteToOther,
		LogSenderIndex:   n.index,
		LogReceiverIndex: msg.LogSenderIndex,
	})
	n.hasVote = true
	return
}

//处理心跳和msg
func (n *node) handleAddNode(msg *raftlog) {
	if msg.logType != addNode {
		panic("Something wrong with message")
	}
	newNodeIpSlice := strings.Fields(msg.msg)
	n.IpPool = append(n.IpPool, newNodeIpSlice...)
	msg.term = n.term
	msg.index = n.nextNodelogIndex()
	n.addNodeLog(msg)
	n.sendDoneToLeader(n.donelog(n.LeaderIndex))
	return
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
	return
}

func (n *node) handleClientGet(msg *raftlog) {
	if msg.logType != clientGet {
		panic("Something wrong with message")
	}
	if n.nodeState != leader {
		_ = n.sendToNode(n.LeaderIndex, msg)
		return
	}
	value, isExist := n.nodecache.Get(msg.msg)
	valuestring := string(value.ByteSlice())
	n.sendToClient(ReplyClientGET(n.clientIp, &valuestring, isExist))
	return
}

func (n *node) handleClientAddNode(msg *raftlog) {
	if msg.logType != clientAddNode {
		panic("Someting wrong with message")
	}
	if n.nodeState != leader {
		_ = n.sendToNode(n.LeaderIndex, msg)
		return
	}
	newNodeIpSlice := strings.Fields(msg.msg)
	n.IpPool = append(n.IpPool, newNodeIpSlice...)
	msg.term = n.term
	msg.index = n.nextNodelogIndex()
	n.addNodeLog(msg)
	n.sendAddNodeToAllNode()
	return
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
			err := n.sendToNode(recvIndex, n.getVotedlog(recvIndex))
			if err != nil {
				err = err
			}
		}(recvIndex)
	}
	return err
}

//日志同步
//TODO 增加日志判断和同步

func (n *node) syncLog()

//TODO 日志的持久化
