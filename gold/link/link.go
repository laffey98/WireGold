package link

import (
	"errors"
	"net"
	"sync"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/sirupsen/logrus"
)

// Link 是本机到 peer 的连接抽象
type Link struct {
	// peer 的公钥
	pubk *[32]byte
	// peer 的公网 ip:port
	pep string
	// 决定本机是否定时向 peer 发送 hello 保持 NAT。
	// 以秒为单位，小于等于 0 不发送
	keepalive int64
	// 收到的包的队列
	pipe chan *head.Packet
	// peer 的虚拟 ip
	peerip net.IP
	// peer 的公网 endpoint
	endpoint *net.UDPAddr
	// 本机允许接收/发送的 ip 网段
	allowedips []*net.IPNet
	// 是否已经调用过 keepAlive
	haskeepruning bool
	// 是否允许转发
	allowtrans bool
	// 连接的状态，详见下方 const
	status int
	// 连接所用对称加密密钥
	key *[]byte
}

const (
	LINK_STATUS_DOWN = iota
	LINK_STATUS_HALFUP
	LINK_STATUS_UP
)

var (
	// 本机活跃的所有连接
	connections = make(map[string]*Link)
	// 读写同步锁
	connmapmu sync.RWMutex
	// 本机监听的 endpoint
	myconn *net.UDPConn
)

// Connect 初始化与 peer 的连接
func Connect(peer string) (*Link, error) {
	p, ok := IsInPeer(net.ParseIP(peer).String())
	if ok {
		p.keepAlive()
		return p, nil
	}
	return nil, errors.New("peer not exist")
}

// Close 关闭到 peer 的连接
func (l *Link) Close() {
	connmapmu.Lock()
	delete(connections, l.peerip.String())
	connmapmu.Unlock()
	l.status = LINK_STATUS_DOWN
}

// Read 从 peer 收包
func (l *Link) Read() *head.Packet {
	return <-l.pipe
}

// Write 向 peer 发包
func (l *Link) Write(p *head.Packet) (n int, err error) {
	p.Data, err = l.Encode(p.Data)
	if err == nil {
		var d []byte
		d, err = p.Mashal(me.String(), l.peerip.String())
		logrus.Debugln("[link] write data", string(d))
		if err == nil {
			n, err = myconn.WriteToUDP(d, l.NextHop(l.peerip).endpoint)
		}
	}
	return
}
