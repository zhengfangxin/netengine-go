package netengine

import (
	"net"
	"sync/atomic"
	"time"
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
	n.MaxBufLen = default_max_buf_len
	n.RecvBufLen = default_read_buf_len
	n.ReadTimeout = 0
	n.WriteTimeout = 0
	n.IsStart = false

	c.listener_list[n.ID] = n

	return n.ID, nil
}
func (c *NetEngine) set_listen_buf(lis *listener, maxBuf,recvBuf int) {
	atomic.StoreInt32(&lis.MaxBufLen, int32(maxBuf))
	atomic.StoreInt32(&lis.RecvBufLen, int32(recvBuf))
}
func (c *NetEngine) set_listen_timeout(lis *listener, readTimeout,writeTimeout time.Duration) {
	lis.ReadTimeout = readTimeout
	lis.WriteTimeout = writeTimeout
}
func (c *NetEngine) start_listen(lis *listener) {
	if lis.IsStart {
		return
	}
	lis.IsStart = true
	go c.accept_run(lis)
}
func (c *NetEngine) close_listen(lis *listener) {
	lis.Listen.Close()
}

func (c *NetEngine) accept_run(lis *listener) {
	listen := lis.Listen
	defer listen.Close()
	defer func() {
		c.del_conntion_chan <- lis.ID
	}()

	for {
		con, err := listen.AcceptTCP()
		if err != nil {
			break
		}

		addr := con.RemoteAddr()
		r := c.notify.OnAcceptBefore(lis.ID, addr)
		if !r {
			con.Close()
			continue
		}

		maxBufLen := atomic.LoadInt32(&lis.MaxBufLen)
		recvBufLen := atomic.LoadInt32(&lis.RecvBufLen)
		readTimeout := lis.ReadTimeout
		writeTimeout := lis.WriteTimeout

		var msg add_conntion_msg
		msg.Con = con
		msg.MaxBufLen = maxBufLen
		msg.RecvBufLen = recvBufLen
		msg.ReadTimeout = readTimeout
		msg.WriteTimeout = writeTimeout
		msg.ch = make(chan *conntion)
		c.add_conntion_chan <- msg

		ncon := <-msg.ch

		c.notify.OnAccept(lis.ID, ncon.ID, addr)

		c.start_conntion(ncon)
	}
	c.notify.OnClosed(lis.ID)
}
