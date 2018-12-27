package netengine

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

func (c *NetEngine) Init() error {
	c.conntion_list = make(map[int]*conntion)
	c.listener_list = make(map[int]*listener)
	c.id = 1
	c.lock = new(sync.Mutex)

	c.add_conntion_chan = make(chan add_conntion_msg)
	c.del_conntion_chan = make(chan int)
	c.stop_chan = make(chan stop_msg)

	c.get_remote_addr_chan = make(chan get_addr_msg)
	c.get_local_addr_chan = make(chan get_addr_msg)
	c.add_listen_chan = make(chan add_listen_msg)
	c.start_chan = make(chan start_msg)
	c.send_chan = make(chan send_msg, 1024)
	c.get_sendfunc_chan = make(chan get_sendfunc_msg)
	c.close_chan = make(chan close_msg)

	go c.manage_run()

	return nil
}
func (c *NetEngine) Stop() {
	var msg stop_msg
	msg.ch = make(chan int)
	c.stop_chan <- msg

	_ = <-msg.ch

	// close all chan
	close(c.get_remote_addr_chan)
	close(c.get_local_addr_chan)
	close(c.add_listen_chan)
	close(c.add_conntion_chan)
	close(c.del_conntion_chan)
	close(c.start_chan)
	close(c.send_chan)
	close(c.close_chan)
}

func (c *NetEngine) GetRemoteAddr(id int) (net.Addr, bool) {
	defer func() {
		recover()
	}()

	var msg get_addr_msg
	msg.ID = id
	msg.ch = make(chan net.Addr)

	c.get_remote_addr_chan <- msg

	addr, ok := <-msg.ch
	if !ok {
		return nil, false
	}
	return addr, true
}
func (c *NetEngine) GetLocalAddr(id int) (net.Addr, bool) {
	defer func() {
		recover()
	}()

	var msg get_addr_msg
	msg.ID = id
	msg.ch = make(chan net.Addr)

	c.get_local_addr_chan <- msg

	addr, ok := <-msg.ch
	if !ok {
		return nil, false
	}
	return addr, true
}

func (c *NetEngine) AddListen(lis net.Listener, notify NetNotify) (id int, err error) {
	defer func() {
		recover()
	}()

	var msg add_listen_msg
	msg.Lis = lis
	msg.Notify = notify
	msg.ch = make(chan listen_ret_msg)

	c.add_listen_chan <- msg

	r, ok := <-msg.ch
	if !ok {
		return 0, errors.New("can't get result")
	}
	return r.ID, r.err
}

func (c *NetEngine) AddConnection(con net.Conn, notify NetNotify, recvBufLen, maxSendBufLen int, readTimeout,writeTimeout time.Duration) (id int, err error) {
	defer func() {
		recover()
	}()
	
	var msg add_conntion_msg
	msg.Con = con
	msg.Notify = notify
	msg.MaxBufLen = maxSendBufLen
	msg.RecvBufLen = recvBufLen
	msg.ReadTimeout = readTimeout
	msg.WriteTimeout = writeTimeout
	msg.ch = make(chan add_conntion_ret_msg)
	c.add_conntion_chan <- msg

	r, ok := <-msg.ch
	if !ok {
		return 0, errors.New("can't get result")
	}
	return r.ID, r.err
}
/*
Listen or CoonnectTo
SetBuffer;SetCloseTime
Start
*/
func (c *NetEngine) Start(id int) {
	defer func() {
		recover()
	}()

	var msg start_msg
	msg.ID = id

	c.start_chan <- msg
}

// Send is asynchronous，不会持有data
func (c *NetEngine) Send(id int, data []byte) {
	defer func() {
		recover()
	}()

	var msg send_msg
	msg.ID = id
	msg.Data = make([]byte, len(data))
	copy(msg.Data, data)

	c.send_chan <- msg
}

// SendFunc is synchronous
func (c *NetEngine) GetSendFunc(id int) SendFunc {
	defer func() {
		recover()
	}()

	var msg get_sendfunc_msg
	msg.ID = id
	msg.ch = make(chan SendFunc)

	c.get_sendfunc_chan <- msg

	r, ok := <-msg.ch
	if !ok {
		fmt.Println("can't get send func")
		return nil
	}
	if r == nil {
		fmt.Println("can't get nil send func")
	}
	return r
}

func (c *NetEngine) Close(id int) {
	defer func() {
		recover()
	}()

	var msg close_msg
	msg.ID = id

	c.close_chan <- msg
}
