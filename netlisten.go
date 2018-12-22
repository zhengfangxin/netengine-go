package netengine

import (
	"net"
)


type listener struct {
	ID			int
	C			*NetEngine
	Listen		net.Listener
	Notify		NetNotify
	IsStart   bool
}

func (lis *listener) start() {
	if lis.IsStart {
		return
	}
	lis.IsStart = true
	go lis.accept_run()
}
func (lis *listener) close() {
	lis.Listen.Close()
}

func (lis *listener) accept_run() {
	listen := lis.Listen
	defer listen.Close()
	defer func() {
		lis.C.del_conntion_chan <- lis.ID
	}()

	notify := lis.Notify

	for {
		con, err := listen.Accept()
		if err != nil {
			break
		}

		notify.OnAccepted(lis.ID, con)		
	}
	notify.OnClosed(lis.ID)
}
