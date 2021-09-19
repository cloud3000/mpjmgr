package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type reader struct {
	name  string
	input io.ReadCloser
}

type writer struct {
	name   string
	output io.WriteCloser
}

type job struct {
	jid       int
	spoolfile string
	stdlist   chan []byte
	cmd       *exec.Cmd
	in        writer
	out       reader
	err       reader
	scl       []string
}

func newJob() (*job, error) {
	var e error = nil
	if len(os.Args) == 1 {
		e = fmt.Errorf(" Script file name required as command-line arg")
		return nil, e
	}

	fmt.Printf("Arg: [%s]", os.Args[1])
	fmt.Printf("Arg count: [%2d]", len(os.Args))

	j := &job{}
	j.err.name = "joberr"
	j.out.name = "jobout"
	j.in.name = "jobin"

	e = j.newStdlist()
	if e != nil {
		return nil, e
	}

	j.cmd = exec.Command("/usr/bin/bash", "-v")

	j.in.output, e = j.cmd.StdinPipe()
	if e != nil {
		e = fmt.Errorf("Error creating jobin pipe\n%w\n", e)
		return nil, e
	}

	j.out.input, e = j.cmd.StdoutPipe()
	if e != nil {
		e = fmt.Errorf("Error creating jobout pipe\n%w\n", e)
		return nil, e
	}

	j.err.input, e = j.cmd.StderrPipe()
	if e != nil {
		e = fmt.Errorf("Error creating joberr pipe\n%w\n", e)
		return nil, e
	}

	return j, nil
}

func getScript(fname string) ([]string, error) {

	file, err := os.Open(fname)

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var text []string

	for scanner.Scan() {
		text = append(text, fmt.Sprintf("%s\n", scanner.Text()))
	}
	file.Close()
	text = append(text, fmt.Sprintf("%s\n", "date"))
	text = append(text, fmt.Sprintf("%s\n", "exit"))
	return text, nil
}

func (j *job) newFname() error {
	// read the whole file at once
	b, err := ioutil.ReadFile("seqnum.txt")
	if err != nil {
		return err
	}
	jobid, err := strconv.Atoi(fmt.Sprintf("%s", b))
	if err != nil {
		return err
	}

	jobid++
	b = []byte(fmt.Sprintf("%v", jobid))

	// write the whole body at once
	err = ioutil.WriteFile("seqnum.txt", b, 0644)
	if err != nil {
		return err
	}

	j.spoolfile = fmt.Sprintf("O%s.spd", string(b))
	j.jid = jobid
	return nil
}

func (j *job) manageStdlist() {
	f, err := os.OpenFile(j.spoolfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()
	for {
		select {
		case stdline := <-j.stdlist:
			if _, err := f.Write(stdline); err != nil {
				fmt.Println(err)
			}
		}
	}
}

func (j *job) newStdlist() error {
	j.stdlist = make(chan []byte)
	err := j.newFname()
	if err != nil {
		return err
	}

	// Start accepting StdList records
	go j.manageStdlist()

	// JOB BACKUP_HOME,MICHAEL.MICHAEL,scripts
	// PRIORITY=DS;INPRI=8;TIME=UNLIMITED secs.
	// Job Number=#J###
	j.stdlist <- []byte(fmt.Sprintf("%s\n", fmt.Sprintf("#J%2d\n", j.jid)))
	// DATETIME STAMP
	j.stdlist <- []byte(fmt.Sprintf("%s\n", fmt.Sprintf(time.Now().Format("Mon Jan 2 15:04:05 -0700 MST 2006"))))
	// UNAME:
	// STREAMED BY:
	// STREAM DATE:
	// WELCOME
	// You are now signed in	return nil
}

func (j *job) stdCopy(buf *bufio.Reader) {
	for run := true; run; {
		result, _, err := buf.ReadLine()
		if err != nil {
			if err != io.EOF {
				j.stdlist <- []byte(fmt.Sprintf("listStderr Error: %s", err))
			}
			run = false
		}

		if run {
			j.stdlist <- []byte(fmt.Sprintf("%s\n", result))
		}
	}
}

func (j *job) listStderr() {
	errbuf := bufio.NewReader(j.err.input)
	j.stdCopy(errbuf)

}

func (j *job) listStdout() {
	outbuf := bufio.NewReader(j.out.input)
	j.stdCopy(outbuf)
}

func (j *job) cleanup() {
	err := j.cmd.Wait()
	if err != nil {
		fmt.Printf("cleanup Wait Error: %s", err)
	}
}

func main() {
	job, err := newJob()
	job.stdlist <- []byte(fmt.Sprintf("main Arg1: [%s]\n", os.Args[1]))

	err = job.cmd.Start()
	if err != nil {
		fmt.Printf("cmd.Start Error: %s", err)
	}

	fmt.Printf(" #J%2d\n", job.jid)
	job.stdlist <- []byte(fmt.Sprintf(" #J%2d\n", job.jid))

	go job.listStderr()
	go job.listStdout()

	text, err := getScript(os.Args[1])
	if err != nil {
		fmt.Printf("Unable to stream %s, %v", os.Args[1], err.Error())
		job.stdlist <- []byte(fmt.Sprintf("Unable to stream %s, %v", os.Args[1], err.Error()))
	}

	for _, each_ln := range text {
		if each_ln != "#!/bin/bash\n" {

			_, err := job.in.output.Write([]byte(each_ln))
			if err != nil {
				fmt.Printf("Error: write to job input stream failed, %s\n", err)
			}

		}

	}
	job.cleanup()
}
