package netengine

import (
	"net"
	"time"
)

func (c *NetEngine) manage_run() {
	for {
		select {
		case r := <-c.add_conntion_chan:
			con := c.add_conntion(r.Con, r.MaxBufLen, r.Timeout, r.SendValid, r.RecvValid)
			r.ch <- con
		case r := <-c.listen_chan:
			var msg listen_ret_msg
			msg.ID, msg.err = c.listen(r.Net, r.Addr)
			r.ch <- msg
		case r := <-c.connect_chan:
			var msg listen_ret_msg
			msg.ID, msg.err = c.connectto(r.Net, r.Addr)
			r.ch <- msg
		case r := <-c.get_remote_addr_chan:
			addr, ok := c.get_remote_addr(r.ID)
			if ok {
				r.ch <- addr
			} else {
				r.ch <- nil
			}
		case r := <-c.get_local_addr_chan:
			addr, ok := c.get_local_addr(r.ID)
			if ok {
				r.ch <- addr
			} else {
				r.ch <- nil
			}
		case r := <-c.set_buf_chan:
			c.set_buf(r.ID, r.MaxSendBufLen)
		case r := <-c.set_closetime_chan:
			c.set_close_time(r.ID, r.CloseSecond, r.Send, r.Recv)
		case r := <-c.start_chan:
			c.start(r.ID)
		case r := <-c.send_chan:
			c.send_data(r.ID, r.Data)
		case r := <-c.close_chan:
			c.close(r.ID)
		case <-time.After(time.Hour):

		}
	}
}

func (c *NetEngine) get_id() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	r := c.id
	for ; ; r++ {
		if _, ok := c.conntion_list[r]; !ok {
			break
		}
		if _, ok := c.listener_list[r]; !ok {
			break
		}
	}
	c.id = r + 1
	return r
}

func (c *NetEngine) get_remote_addr(id int) (addr net.Addr, r bool) {
	addr = nil
	r = false

	_, ok := c.listener_list[id]
	if ok {
		return nil, false
	}
	con, ok := c.conntion_list[id]
	if ok {
		return con.Con.RemoteAddr(), true
	}
	return
}
func (c *NetEngine) get_local_addr(id int) (addr net.Addr, r bool) {
	addr = nil
	r = false

	lis, ok := c.listener_list[id]
	if ok {
		return lis.Listen.Addr(), true
	}
	con, ok := c.conntion_list[id]
	if ok {
		return con.Con.RemoteAddr(), true
	}
	return
}
func (c *NetEngine) set_buf(id int, maxBuf int) {
	lis, ok := c.listener_list[id]
	if ok {
		c.set_listen_buf(lis, maxBuf)
	}
	con, ok := c.conntion_list[id]
	if ok {
		c.set_conntion_buf(con, maxBuf)
	}
}
func (c *NetEngine) set_close_time(id int, close_second int, send, recv bool) {
	lis, ok := c.listener_list[id]
	if ok {
		c.set_listen_close_time(lis, close_second, send, recv)
	}
	con, ok := c.conntion_list[id]
	if ok {
		c.set_conntion_close_time(con, close_second, send, recv)
	}
}
func (c *NetEngine) start(id int) {
	lis, ok := c.listener_list[id]
	if ok {
		c.start_listen(lis)
	}
	con, ok := c.conntion_list[id]
	if ok {
		c.start_conntion(con)
	}
}
func (c *NetEngine) close(id int) {
	lis, ok := c.listener_list[id]
	if ok {
		c.close_listen(lis)
	}
	con, ok := c.conntion_list[id]
	if ok {
		c.close_conntion(con)
	}
}
