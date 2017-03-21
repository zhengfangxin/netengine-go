package netengine

import (
	"net"
	//"time"
	"errors"
	"sync"
)

func (c *NetEngine) Init(notify NetNotify) error {
	c.conntion_list = make(map[int]*conntion)
	c.listener_list = make(map[int]*listener)
	c.id = 1
	c.notify = notify
	c.lock = new(sync.Mutex)

	c.add_conntion_chan = make(chan add_conntion_msg)
	c.del_conntion_chan = make(chan int)
	c.stop_chan = make(chan stop_msg)

	c.get_remote_addr_chan = make(chan get_addr_msg)
	c.get_local_addr_chan = make(chan get_addr_msg)
	c.set_buf_chan = make(chan set_buf_msg)
	c.set_closetime_chan = make(chan set_closetime_msg)
	c.listen_chan = make(chan listen_msg)
	c.connect_chan = make(chan connect_msg)
	c.start_chan = make(chan start_msg)
	c.send_chan = make(chan send_msg, 1024)
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
	close(c.set_buf_chan)
	close(c.set_closetime_chan)
	close(c.listen_chan)
	close(c.connect_chan)
	close(c.start_chan)
	close(c.send_chan)
	close(c.close_chan)
}

func (c *NetEngine) GetRemoteAddr(id int) (net.Addr, bool) {
	defer recover()
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
	defer recover()
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

// default：1m
func (c *NetEngine) SetBuffer(id int, maxSendBufLen int) {
	defer recover()
	var msg set_buf_msg
	msg.ID = id
	msg.MaxSendBufLen = maxSendBufLen

	c.set_buf_chan <- msg
}

// default：1m,send,recv:true
func (c *NetEngine) SetCloseTime(id int, close_second int, send, recv bool) {
	defer recover()
	var msg set_closetime_msg
	msg.ID = id
	msg.CloseSecond = close_second
	msg.Send = send
	msg.Recv = recv

	c.set_closetime_chan <- msg
}
func (c *NetEngine) Listen(net, addr string) (id int, err error) {
	defer recover()
	var msg listen_msg
	msg.Net = net
	msg.Addr = addr
	msg.ch = make(chan listen_ret_msg)

	c.listen_chan <- msg

	r, ok := <-msg.ch
	if !ok {
		return 0, errors.New("can't get result")
	}
	return r.ID, r.err
}
func (c *NetEngine) ConnectTo(net, addr string) (id int, err error) {
	defer recover()
	var msg connect_msg
	msg.Net = net
	msg.Addr = addr
	msg.ch = make(chan listen_ret_msg)

	c.connect_chan <- msg

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
	defer recover()
	var msg start_msg
	msg.ID = id

	c.start_chan <- msg
}

// Send is asynchronous
func (c *NetEngine) Send(id int, data []byte) {
	defer recover()
	var msg send_msg
	msg.ID = id
	msg.Data = data

	c.send_chan <- msg
}

func (c *NetEngine) Close(id int) {
	defer recover()
	var msg close_msg
	msg.ID = id

	c.close_chan <- msg
}
