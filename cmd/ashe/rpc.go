package main

import (
	"net/http"
	"net/rpc"
	"path/filepath"
	"sync"
)

type Server struct {
	center *ProcessManager
}

func (s *Server) Init(o *ProcessConfig, d *ProcessStatus) error {
	docker, err := s.center.init(o)
	if err != nil {
		return err
	}
	*d = *docker.ProcessStatus
	return nil
}

func (s *Server) Kill(name string, _ *uint64) error {
	return s.center.kill(name)
}

func (s *Server) Info(name string, d *ProcessStatus) error {
	docker, err := s.center.info(name)
	if err != nil {
		return err
	}
	*d = *docker.ProcessStatus
	return nil
}

func (s *Server) List(_ uint64, d *[]*ProcessStatus) error {
	a := s.center.list()
	for _, e := range a {
		*d = append(*d, e.ProcessStatus)
	}
	return nil
}

type client struct {
	core *rpc.Client
}

func (c *client) init(o *ProcessConfig) (*ProcessStatus, error) {
	docker := &ProcessStatus{}
	err := c.core.Call("Server.Init", o, docker)
	return docker, err
}

func (c *client) kill(name string) error {
	var a uint64 = 0
	return c.core.Call("Server.Kill", name, &a)
}

func (c *client) info(name string) (*ProcessStatus, error) {
	docker := &ProcessStatus{}
	err := c.core.Call("Server.Info", name, docker)
	return docker, err
}

func (c *client) list() ([]*ProcessStatus, error) {
	dockerList := []*ProcessStatus{}
	err := c.core.Call("Server.List", uint64(0), &dockerList)
	return dockerList, err
}

func listen(path string) error {
	pathLogs := filepath.Join(path, "logs")

	logger, err := logcon(pathLogs)
	if err != nil {
		return err
	}
	rpc.Register(&Server{
		&ProcessManager{
			logger: logger,
			hubber: &sync.Map{},
		},
	})
	rpc.HandleHTTP()

	go http.ListenAndServe(":8080", nil)
	return nil

	// for {
	// 	conn, err := l.Accept()
	// 	if err != nil {
	// 		log.Println(err)
	// 	}
	// 	go func() {
	// 		defer conn.Close()
	// 		server.ServeCodec(rpc.NewServer().ServeRequest())
	// 	}()
	// }
}

func dial(path string) (*client, error) {
	c, err := rpc.DialHTTP("tcp", "127.0.0.1:8080")
	if err != nil {
		return nil, err
	}
	client := &client{core: c}
	if err != nil {
		return nil, err
	}
	return client, nil
}
