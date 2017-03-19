package netengine

import (
	"bytes"
	"net"
	"sync/atomic"
	"time"
)

func (c *NetEngine) connectto(nettype, addr string) (int, error) {
	tcpaddr, err := net.ResolveTCPAddr(nettype, addr)
	if err != nil {
		return 0, err
	}

	con, err := net.DialTCP(nettype, tcpaddr, nil)
	if err != nil {
		return 0, err
	}

	r := c.add_conntion(con, default_buf_len, default_timeout, 1, 1)

	return r.ID, nil
}
func (c *NetEngine) add_conntion(con *net.TCPConn, maxBufLen, timeout, send, recv int32) *conntion {
	n := new(conntion)
	n.ID = c.get_id()
	n.Con = con
	n.MaxBufLen = maxBufLen
	n.Timeout = timeout
	n.RecvValid = recv
	n.SendValid = send
	n.SendChan = make(chan []byte, 128)

	c.conntion_list[n.ID] = n

	return n
}

func (c *NetEngine) set_conntion_buf(con *conntion, maxBuf int) {
	atomic.StoreInt32(&con.MaxBufLen, int32(maxBuf))
}
func (c *NetEngine) set_conntion_close_time(con *conntion, close_second int, send, recv bool) {
	var isend int32 = 0
	if send {
		isend = 1
	}
	var irecv int32 = 0
	if recv {
		irecv = 1
	}

	add := time.Second * time.Duration(close_second)
	atomic.StoreInt32(&con.Timeout, int32(close_second))
	atomic.StoreInt32(&con.SendValid, isend)
	atomic.StoreInt32(&con.RecvValid, irecv)
	if send {
		con.Con.SetWriteDeadline(time.Now().Add(add))
	} else {
		con.Con.SetWriteDeadline(time.Time{})
	}
	if recv {
		con.Con.SetReadDeadline(time.Now().Add(add))
	} else {
		con.Con.SetReadDeadline(time.Time{})
	}
}
func (c *NetEngine) start_conntion(con *conntion) {
}
func (c *NetEngine) close_conntion(con *conntion) {
	con.Con.Close()
}
func (c *NetEngine) send_data(id int, data []byte) {
}

func (c *NetEngine) conntion_run(con *conntion) {
	//go c.conntion_write(con)
	c.conntion_recv(con)

	c.notify.OnClosed(con.ID)

}
func (c *NetEngine) conntion_recv(con *conntion) {
	buf := bytes.NewBufferString("")
	net_con := con.Con
	defer net_con.Close()

recv_loop:
	for {
		recv := atomic.LoadInt32(&con.RecvValid)
		timeout := atomic.LoadInt32(&con.Timeout)
		if recv != 0 {
			add := time.Duration(timeout)
			net_con.SetReadDeadline(time.Now().Add(add))
		}

		all_buf := make([]byte, 1024*5)
		n, err := net_con.Read(all_buf)
		if err != nil {
			break
		}
		cur := all_buf[:n]
		buf.Write(cur)
		var recvd = buf.Bytes()

		for {
			r := c.notify.OnRecv(con.ID, recvd)
			if r < 0 {
				break recv_loop
			}
			if r > 0 {
				recvd = recvd[r:]
				buf = bytes.NewBuffer(recvd)
			} else {
				break
			}
		}
	}

}
