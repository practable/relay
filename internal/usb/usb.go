package usb

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"go.bug.st/serial"
)

type USB struct {
	mu      *sync.Mutex
	sp      serial.Port
	port    string
	timeout time.Duration
}

func NewUSB() *USB {
	return &USB{
		mu:   &sync.Mutex{},
		port: "unknown",
		//don't initialise sp - use Open() for that
	}
}

func (r *USB) Open(port string, baud int, timeout time.Duration) error {

	r.timeout = timeout

	mode := &serial.Mode{
		BaudRate: baud,
	}

	p, err := serial.Open(port, mode)

	if err != nil {
		log.WithFields(log.Fields{"port": port, "baud": baud, "timeout": timeout.String()}).Errorf("failed to open usb port")
		return err
	}

	r.sp = p

	err = r.sp.SetReadTimeout(timeout)

	if err != nil {
		log.WithFields(log.Fields{"port": port, "baud": baud, "timeout": timeout.String()}).Errorf("failed to set timeout when opening usb port")
		return err
	}

	log.WithFields(log.Fields{"port": port, "baud": baud, "timeout": timeout.String()}).Infof("opened usb port")

	return nil

}

func (r *RFUSB) Close() error {
	// don't take lock because there is read, close concurrency
	// https://github.com/bugst/go-serial/blob/e381f2c1332081ea593d73e97c71342026876857/serial_linux_test.go#L35
	return r.sp.Close()
}

// set up something same as jump for read/write which is io.Copy()

/*
func (r *RFUSB) SetPort(port string) error {

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sp == nil {
		return errors.New("port is nil")
	}

	resp := make([]byte, 128)

	// read any stale messages before we send our command
	// make a short timeout temporarily to avoid wasting time
	err := r.sp.SetReadTimeout(10 * time.Millisecond)
	if err != nil {
		return fmt.Errorf("setting short timeout before drain failed because %s", err.Error())
	}
DRAINED:
	for {

		n, err := r.sp.Read(resp)
		if err != nil {
			return err //port probably closed
		}
		//https://github.com/bugst/go-serial/blob/e381f2c1332081ea593d73e97c71342026876857/serial_unix.go#L94
		// timeout is n==0, err==nil
		if n == 0 {
			break DRAINED
		}
		continue
	}

	// restore normal timeout
	err = r.sp.SetReadTimeout(r.timeout)

	if err != nil {
		return fmt.Errorf("restoring timeout after drain failed because %s", err.Error())
	}

	request := Command{
		Set: "port",
		To:  port,
	}

	req, err := json.Marshal(request)

	if err != nil {
		return fmt.Errorf("marshal request failed because %s", err.Error())
	}

	n, err := r.sp.Write(req)

	log.WithFields(log.Fields{"count_expected": len(req), "count_actual": n, "data_expected": string(req), "data_actual": string(req[:n])}).Trace("wrote message to usb")

	if err != nil {
		return err
	}

	if n < len(req) {
		// TODO consider a follow up write?
		return errors.New("did not finish writing message")
	}

	// Get the response
	// note we do a drain afterwards to avoid this error:
	// unmarshalling reply failed because because unexpected end of JSON input. Reply was {"report":"port","is":"sho

	reply := make([]byte, 128)

	n, err = r.sp.Read(resp)

	if err != nil {
		return fmt.Errorf("reading reply failed because because %s", err.Error())
	}

	if n == 0 {
		return fmt.Errorf("empty reply")
	}

	idx := n - 1
	copy(reply[:], resp[:])

	//check we drained the whole message
	// make a short timeout temporarily to avoid wasting time if we got the whole message already
	err = r.sp.SetReadTimeout(100 * time.Millisecond) //don't make it too short or else get partial messages (that happens at 10ms)

	if err != nil {
		return fmt.Errorf("setting short timeout before drain failed because %s", err.Error())
	}
COMPLETED:
	for {

		n, err := r.sp.Read(resp)
		if err != nil {
			return err //port probably closed
		}
		//https://github.com/bugst/go-serial/blob/e381f2c1332081ea593d73e97c71342026876857/serial_unix.go#L94
		// timeout is n==0, err==nil
		if n == 0 {
			break COMPLETED
		}
		if (idx + n) < len(reply) {
			copy(reply[idx+1:idx+n], resp[:]) //TODO check if copies null?
			idx = idx + n

		} else {
			log.Fatal("pkg/rfusb: serial read buffer full")
		}
		continue
	}

	var report Report
	log.Debugf("(%d)%s", idx, string(reply[:idx]))
	err = json.Unmarshal(reply[:idx], &report) //truncate to bytes read to avoid \x00 char which breaks unmarshal

	if err != nil {
		return fmt.Errorf("unmarshalling reply failed because because %s. Reply was %s", err.Error(), string(resp))
	}
	log.WithFields(log.Fields{"count_actual": n, "data_actual": string(resp[:n])}).Trace("read message from usb")
	if strings.ToLower(report.Report) != "port" {
		return errors.New("response was not a port report")
	}
	if strings.ToLower(report.Is) != strings.ToLower(port) {
		return err
	}
	r.port = port
	return nil

}
*/
