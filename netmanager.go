package netengine

import (
	"net"
	"time"
)

func (c *NetEngine) manage_run() {
	is_stop := false
	var stop_req stop_msg
for_loop:
	for {
		timeout := time.Hour
		if is_stop {
			timeout = time.Second
		}
		timer := time.NewTimer(timeout)
		select {
		case r := <-c.send_chan:
			c.send_data(r.ID, r.Data)
		case r := <-c.add_conntion_chan:
			con := c.add_conntion(r.Con, r.MaxBufLen, r.Timeout, r.SendValid, r.RecvValid)
			r.ch <- con
		case r := <-c.del_conntion_chan:
			c.del_conntion(r)
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
		case r := <-c.close_chan:
			c.close(r.ID)
		case r := <-c.stop_chan:
			stop_req = r
			is_stop = true
			c.closeall()
		case <-timer.C:
			timer = nil
			if is_stop {
				c.closeall()
				if len(c.conntion_list)+len(c.listener_list) <= 0 {
					break for_loop
				}
			}
		}
		if timer != nil && !timer.Stop() {
			<-timer.C
		}
	}
	stop_req.ch <- 0
}

func (c *NetEngine) get_id() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	r := c.id
	for {
		if r > 100*10000 {
			r = 1
		}
		if _, ok := c.conntion_list[r]; !ok {
			break
		}
		if _, ok := c.listener_list[r]; !ok {
			break
		}
		r = r + 1
	}
	c.id = r + 1
	return r
}
func (c *NetEngine) del_conntion(id int) {
	con, ok := c.conntion_list[id]
	if ok {
		close(con.SendChan)
		delete(c.conntion_list, id)
		return
	}
	delete(c.listener_list, id)
}
func (c *NetEngine) closeall() {
	listen_idlist := make([]int, 0)
	conntion_idlist := make([]int, 0)
	for _, v := range c.listener_list {
		v.Listen.Close()
		if !v.IsStart {
			listen_idlist = append(listen_idlist, v.ID)
		}
	}
	for _, v := range c.conntion_list {
		v.Con.Close()
		if !v.IsStart {
			conntion_idlist = append(conntion_idlist, v.ID)
		}
	}

	for _, v := range listen_idlist {
		delete(c.listener_list, v)
	}
	for _, v := range conntion_idlist {
		delete(c.conntion_list, v)
	}
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
