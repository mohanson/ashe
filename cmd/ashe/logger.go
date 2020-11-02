package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

type file struct {
	name string
	file *os.File
	size uint32
	used uint32
	time time.Time
}

func (f *file) write(b []byte) (n int, err error) {
	split := f.size - f.used
	if split < uint32(len(b)) {
		writeN := 0
		writeN, err = f.file.Write(b[:split])
		n += writeN
		f.used += uint32(writeN)
		if err != nil {
			return
		}
		if err = f.file.Close(); err != nil {
			return
		}
		a := []byte{}
		a, err = ioutil.ReadFile(f.name)
		if err != nil {
			return
		}
		a = a[f.size/2 : f.size]
		f.file, err = os.OpenFile(f.name, os.O_WRONLY|os.O_TRUNC, 0644)
		_, err = f.file.Write(a)
		if err != nil {
			return
		}
		writeN, err = f.file.Write(b[split:])
		n += writeN
		f.used = uint32(writeN)
		if err != nil {
			return
		}
		return
	}
	n, err = f.file.Write(b)
	f.used += uint32(n)
	f.time = time.Now()
	return
}

func (f *file) touch() error {
	if f.file != nil {
		return nil
	}
	d, err := os.OpenFile(f.name, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	f.file = d
	return nil
}

func (f *file) close() error {
	if f.file == nil {
		return nil
	}
	err := f.file.Close()
	if err != nil {
		return err
	}
	f.file = nil
	return err
}

func open(name string, size uint32) (*file, error) {
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	s, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return &file{
		name: name,
		file: f,
		used: uint32(s.Size()),
		size: size,
		time: s.ModTime(),
	}, nil
}

type letter struct {
	name string
	data []byte
}

type writer struct {
	name string
	c    chan letter
}

func (w *writer) Write(b []byte) (n int, err error) {
	l := len(b)
	a := make([]byte, l)
	copy(a, b)
	w.c <- letter{name: w.name, data: a}
	return l, nil
}

type logger struct {
	c                  chan letter
	dict               map[string]*file
	fileSize           uint32
	fileDescriptorKeep time.Duration
	fileKeep           time.Duration
	path               string
}

func (l *logger) open(name string) io.Writer {
	return &writer{
		name: name,
		c:    l.c,
	}
}

func (l *logger) on() {
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case e := <-l.c:
			if err := l.recv(e); err != nil {
				log.Println(err)
			}
		case <-ticker.C:
			l.gc()
		}
	}
}

func (c *logger) recv(l letter) error {
	var (
		f   *file
		b   bool
		err error
	)
	f, b = c.dict[l.name]
	if b == false {
		f, err = open(filepath.Join(c.path, l.name), c.fileSize)
		if err != nil {
			return err
		}
		c.dict[l.name] = f
	}
	err = f.touch()
	if err != nil {
		return err
	}
	_, err = f.write(l.data)
	return err
}

func (c *logger) gc() {
	toc := []*file{}
	tor := []*file{}
	ton := []string{}
	for k, v := range c.dict {
		if time.Since(v.time) > c.fileDescriptorKeep {
			toc = append(toc, v)
		}
		if time.Since(v.time) > c.fileKeep {
			tor = append(tor, v)
			ton = append(ton, k)
		}
	}
	for _, f := range toc {
		f.close()
	}
	for _, f := range tor {
		if err := os.Remove(f.name); err != nil {
			log.Println(err)
		}
	}
	for _, k := range ton {
		delete(c.dict, k)
	}
}

func logcon(path string) (*logger, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err = os.MkdirAll(path, 0755); err != nil {
			return nil, err
		}
	}
	center := &logger{
		c:                  make(chan letter, 1024),
		dict:               map[string]*file{},
		fileSize:           uint32(64 * 1024 * 1024),
		fileDescriptorKeep: time.Minute * 20,
		fileKeep:           time.Hour * 24 * 7,
		path:               path,
	}
	l, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, e := range l {
		n := e.Name()
		f, err := open(filepath.Join(path, n), center.fileSize)
		if err != nil {
			return nil, err
		}
		center.dict[n] = f
	}
	go center.on()
	return center, nil
}
