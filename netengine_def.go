package netengine

import (
	"net"
	"sync"
)

const default_buf_len = 1024 * 1024
const default_timeout = 120 // second

type SendFunc func(data []byte) error
type NetNotify interface {
	// return false to close
	// 同一个listenid 会在一个goroute中调用
	OnAcceptBefore(listenid int, addr net.Addr) bool
	OnAccept(listenid int, id int, addr net.Addr)

	/* return -1 to close
	return >0 to consume data
	return 0 need more data
	use send go send data get more performance
	data is valid only in OnRecv, so to careful use it,
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
	Timeout   int32 // second
	SendValid int32
	RecvValid int32
	IsStart   bool
}
type listener struct {
	ID        int
	Listen    *net.TCPListener
	MaxBufLen int32
	Timeout   int32 // second
	SendValid int32
	RecvValid int32
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
	set_closetime_chan   chan set_closetime_msg
	listen_chan          chan listen_msg
	connect_chan         chan connect_msg
	start_chan           chan start_msg
	send_chan            chan send_msg
	close_chan           chan close_msg
}

type add_conntion_msg struct {
	Con       *net.TCPConn
	MaxBufLen int32
	Timeout   int32 // second
	SendValid int32
	RecvValid int32
	ch        chan *conntion
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
}
type set_closetime_msg struct {
	ID          int
	CloseSecond int
	Send        bool
	Recv        bool
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
type close_msg struct {
	ID int
}
type get_conntion_count_msg struct {
	ID int
	ch chan int
}
