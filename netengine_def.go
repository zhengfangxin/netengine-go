package netengine

import (
	"net"
	"sync"
	"time"
)

const default_read_buf_len = 5 * 1024
const default_max_buf_len = 1024 * 1024

type SendFunc func(data []byte) error
type NetNotify interface {
	// return false to close
	// 同一个listenid 会在一个goroute中调用
	OnAcceptBefore(listenid int, addr net.Addr) bool
	OnAccept(listenid int, id int, addr net.Addr)

	/* return -1 to close
	return >0 to consume data
	return 0 need more data
	use send to send data get more performance
	data is valid only in OnRecv, so careful to use it
	*/
	OnRecv(id int, data []byte, send SendFunc) int

	// OnClosed,OnBufferLimit,OnAccept不会在同一个goroute中调用
	OnClosed(id int)
	// write buffer limit, OnClosed always will call
	OnBufferLimit(id int)
}

type conntion struct {
	ID        int
	SendChan  chan []byte
	Con       *net.TCPConn
	MaxBufLen int32
	RecvBufLen int32

	ReadTimeout   time.Duration
	WriteTimeout   time.Duration
	IsStart   bool
	Send      SendFunc
}
type listener struct {
	ID        int
	Listen    *net.TCPListener
	MaxBufLen int32
	RecvBufLen int32

	ReadTimeout   time.Duration
	WriteTimeout   time.Duration

	IsStart   bool
}

type NetEngine struct {
	conntion_list map[int]*conntion
	listener_list map[int]*listener

	lock sync.Locker
	id   int

	notify NetNotify

	add_conntion_chan chan add_conntion_msg
	del_conntion_chan chan int
	stop_chan         chan stop_msg

	get_remote_addr_chan chan get_addr_msg
	get_local_addr_chan  chan get_addr_msg
	set_buf_chan         chan set_buf_msg
	set_timeout_chan   chan set_timeout_msg
	listen_chan          chan listen_msg
	connect_chan         chan connect_msg
	start_chan           chan start_msg
	send_chan            chan send_msg
	get_sendfunc_chan    chan get_sendfunc_msg
	close_chan           chan close_msg
}

type add_conntion_msg struct {
	Con				*net.TCPConn
	MaxBufLen		int32
	RecvBufLen		int32
	ReadTimeout		time.Duration
	WriteTimeout	time.Duration
	ch				chan *conntion
}
type stop_msg struct {
	ch chan int
}
type get_addr_msg struct {
	ID int
	ch chan net.Addr
}
type set_buf_msg struct {
	ID            int
	MaxSendBufLen int
	RecvBufLen	  int
}
type set_timeout_msg struct {
	ID				int
	ReadTimeout		time.Duration
	WriteTimeout	time.Duration
}

type listen_ret_msg struct {
	ID  int
	err error
}
type listen_msg struct {
	Net  string
	Addr string
	ch   chan listen_ret_msg
}
type connect_msg struct {
	Net  string
	Addr string
	ch   chan listen_ret_msg
}
type start_msg struct {
	ID int
}
type send_msg struct {
	ID   int
	Data []byte
}
type get_sendfunc_msg struct {
	ID int
	ch chan SendFunc
}
type close_msg struct {
	ID int
}
type get_conntion_count_msg struct {
	ID int
	ch chan int
}
