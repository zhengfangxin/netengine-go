package netengine

import (
	"net"
	"sync/atomic"
)

func (c *NetEngine) listen(nettype, addr string) (int, error) {
	tcpaddr, err := net.ResolveTCPAddr(nettype, addr)
	if err != nil {
		return 0, err
	}
	tcplisten, err := net.ListenTCP(nettype, tcpaddr)
	if err != nil {
		return 0, err
	}

	n := new(listener)
	n.ID = c.get_id()
	n.Listen = tcplisten
	n.MaxBufLen = default_buf_len
	n.Timeout = default_timeout
	n.RecvValid = 1
	n.SendValid = 1

	c.listener_list[n.ID] = n

	return n.ID, nil
}
func (c *NetEngine) set_listen_buf(lis *listener, maxBuf int) {
	atomic.StoreInt32(&lis.MaxBufLen, int32(maxBuf))
}
func (c *NetEngine) set_listen_close_time(lis *listener, close_second int, send, recv bool) {
	var isend int32 = 0
	if send {
		isend = 1
	}
	var irecv int32 = 0
	if recv {
		irecv = 1
	}

	atomic.StoreInt32(&lis.Timeout, int32(close_second))
	atomic.StoreInt32(&lis.SendValid, isend)
	atomic.StoreInt32(&lis.RecvValid, irecv)
}
func (c *NetEngine) start_listen(lis *listener) {
	go c.accept_run(lis)
}
func (c *NetEngine) close_listen(lis *listener) {
	lis.Listen.Close()
}

func (c *NetEngine) accept_run(lis *listener) {
	listen := lis.Listen
	defer listen.Close()
	for {
		con, err := listen.AcceptTCP()
		if err != nil {
			return
		}

		addr := con.RemoteAddr()
		r := c.notify.OnAcceptBefore(lis.ID, addr)
		if !r {
			con.Close()
			continue
		}

		maxBufLen := atomic.LoadInt32(&lis.MaxBufLen)
		timeout := atomic.LoadInt32(&lis.Timeout)
		send := atomic.LoadInt32(&lis.SendValid)
		recv := atomic.LoadInt32(&lis.RecvValid)
		var msg add_conntion_msg
		msg.Con = con
		msg.MaxBufLen = maxBufLen
		msg.Timeout = timeout
		msg.SendValid = send
		msg.RecvValid = recv
		msg.ch = make(chan *conntion)
		c.add_conntion_chan <- msg

		ncon := <-msg.ch

		c.notify.OnAccept(lis.ID, ncon.ID)

		c.start_conntion(ncon)
	}
	c.notify.OnClosed(lis.ID)
}
