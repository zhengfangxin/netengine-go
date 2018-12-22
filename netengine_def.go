package netengine

import (
	"net"
	"time"
)

type SendFunc func(data []byte) error
type NetNotify interface {
	/* 同一个listenid 会在一个goroute中调用，
		新的连接需要手动, id,err := AddConnection Start(id)
	*/
	OnAccepted(listenid int, con net.Conn)

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

type add_conntion_ret_msg struct {
	ID  int
	err error
}
type add_conntion_msg struct {
	Con				net.Conn
	Notify			NetNotify
	MaxBufLen		int
	RecvBufLen		int
	ReadTimeout		time.Duration
	WriteTimeout	time.Duration
	ch				chan add_conntion_ret_msg
}

type stop_msg struct {
	ch chan int
}

type get_addr_msg struct {
	ID int
	ch chan net.Addr
}

type listen_ret_msg struct {
	ID  int
	err error
}
type add_listen_msg struct {
	Lis		net.Listener
	Notify	NetNotify
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