package netengine

import (
	"fmt"
	"net"
	"time"
)

type conntion struct {
	ID			int
	C			*NetEngine
	SendChan	chan []byte
	Con			net.Conn
	Notify		NetNotify
	MaxBufLen	int
	RecvBufLen	int

	ReadTimeout		time.Duration
	WriteTimeout	time.Duration
	IsStart   bool
	Send      SendFunc
}

func (con *conntion) get_send() SendFunc {
	writeTimeout := con.WriteTimeout

	r := func(d []byte) error {
		netcon := con.Con		
		
		if writeTimeout != 0 {
			netcon.SetWriteDeadline(time.Now().Add(writeTimeout))
		}

		for {
			n, err := netcon.Write(d)
			if err != nil {
				netcon.Close()
				return err
			}
			if n == len(d) {
				break
			} else {
				fmt.Println("write return", n, len(d))
				d = d[n:]
			}
		}
		return nil
	}
	return r
}

func (con *conntion) start() {
	if con.IsStart {
		return
	}
	con.IsStart = true
	go con.conntion_run()
}
func (con *conntion) close() {
	con.Con.Close()
}
func (con *conntion) conntion_run() {
	go con.conntion_write()
	con.conntion_recv()

	con.C.del_conntion_chan <- con.ID
	con.Notify.OnClosed(con.ID)	
}


func (con *conntion) conntion_recv() {
	net_con := con.Con
	defer net_con.Close()

	send_fun := con.Send

	buflen := int(con.RecvBufLen)
	all_buf := make([]byte, buflen)	
	
	notify := con.Notify
	readTimeout := con.ReadTimeout

	valid_begin_pos := 0
	valid_end_pos := 0

recv_loop:
	for {		
		if readTimeout != 0 {
			net_con.SetReadDeadline(time.Now().Add(readTimeout))
		}

		if valid_end_pos >= len(all_buf) {
			if valid_begin_pos > 2048 {
				copy(all_buf, all_buf[valid_begin_pos:valid_end_pos])
			} else {
				all_len := len(all_buf)
				org_buf := all_buf
				all_buf = make([]byte, all_len*2)
				copy(all_buf, org_buf[valid_begin_pos:valid_end_pos])
			}
			valid_end_pos = valid_end_pos - valid_begin_pos
			valid_begin_pos = 0
		}

		currecv := all_buf[valid_end_pos:len(all_buf)]
		n, err := net_con.Read(currecv)
		if err != nil {
			break
		}
		valid_end_pos = valid_end_pos + n

		for {
			curdata := all_buf[valid_begin_pos:valid_end_pos]
			r := notify.OnRecv(con.ID, curdata, send_fun)

			if r < 0 {
				break recv_loop
			}
			if r > 0 {
				if r > len(curdata) {
					break recv_loop
				}
				valid_begin_pos = valid_begin_pos + r
			} else {
				break
			}
			if valid_begin_pos == valid_end_pos {
				valid_begin_pos = 0
				valid_end_pos = 0
				break
			}
		}		
	}
}
func (con *conntion) send_data(data []byte) {	
	con.SendChan <- data
}
func (con *conntion) conntion_write() {
	defer con.Con.Close()

	data_chan := make(chan []byte)
	defer close(data_chan)

	go con.conntion_write_net(data_chan)

	notify := con.Notify

	datalen := 0
	data := make([][]byte, 0, 1)
	maxBufLen := int(con.MaxBufLen)
	
	timer := time.NewTicker(time.Hour)
	defer timer.Stop()

for_loop:
	for {		
		
		if len(data) > 0 {
			s := data[0]
			select {
			case msg, ok := <-con.SendChan:
				if !ok {
					break for_loop
				}
				datalen += len(msg)
				data = append(data, msg)
				if datalen > maxBufLen {
					notify.OnBufferLimit(con.ID)
					break for_loop
				}
			case data_chan <- s:
				data = data[1:]
				datalen = datalen - len(s)
			case <-timer.C:
			}
		} else {
			select {
			case msg, ok := <-con.SendChan:
				if !ok {
					break for_loop
				}
				datalen += len(msg)
				data = append(data, msg)
				if datalen > maxBufLen {
					notify.OnBufferLimit(con.ID)
					break for_loop
				}
			case <-timer.C:
			}
		}		
	}
}
func (con *conntion) conntion_write_net(data chan []byte) {
	netcon := con.Con
	defer netcon.Close()	

	writeTimeout := con.WriteTimeout

	for d := range data {
		if writeTimeout != 0 {
			netcon.SetWriteDeadline(time.Now().Add(writeTimeout))
		}

		err := con.Send(d)
		if err != nil {
			break
		}
	}
}
