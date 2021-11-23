package sgcache

import "testing"

func TestRaftlogTobytes(t *testing.T) {
	a := raftlog{
		term:    1,
		logType: justHeartBeat,
		msg:     "123",
	}
	if toRaftlog(a.tobytes()) != a {
		t.Errorf("error")
	}
}

func TestNodoTobeLeader(t *testing.T) {
	aNode := InitNode()
	bNode := InitNode()
	cNode := InitNode()
	dNode := InitNode()
	fNode := InitNode()
	aNode.Accept()
	bNode.Accept()
	cNode.Accept()
	dNode.Accept()
	fNode.Accept()
	aNode.beCandidate()
	aNode.beLeader()

}
