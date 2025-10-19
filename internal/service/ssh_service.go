package service

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHServicer interface {
	Close() error
	RunCommand(string) error
	GetOutputCh() chan string
}

func NewSSHService(host, username string, port int, privateKey []byte) *SSHService {
	return &SSHService{
		host:       host,
		username:   username,
		privateKey: privateKey,
	}
}

type SSHService struct {
	host       string
	username   string
	privateKey []byte

	client *ssh.Client
	mu     sync.Mutex
}

func (s *SSHService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil {
		return nil
	}
	err := s.client.Close()
	s.client = nil
	return err
}

func (s *SSHService) RunCommand(cmd string) (chan string, error) {
	if err := s.connect(); err != nil {
		return nil, err
	}

	session, err := s.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("err creating new session: %+w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("err getting stdout pipe: %+w", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("err getting stderr pipe: %+w", err)
	}

	combined := io.MultiReader(stdout, stderr)
	outputCh := make(chan string)

	go func() {
		defer close(outputCh)
		defer session.Close()

		scanner := bufio.NewScanner(combined)
		for scanner.Scan() {
			outputCh <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			log.Printf("scanner error: %+v\n", err)
		}
	}()

	if err := session.Start(cmd); err != nil {
		close(outputCh)
		session.Close()
		return nil, fmt.Errorf("err starting ssh session: %+w", err)
	}

	go func() {
		err := session.Wait()
		if err != nil {
			log.Printf("err waiting session: %+v\n", err)
		}
	}()

	return outputCh, nil
}

func (s *SSHService) connect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		return nil
	}

	auth, err := s.getAuth(s.privateKey)
	if err != nil {
		return err
	}
	config := s.getConfig(s.username, auth, 3*time.Hour)

	client, err := ssh.Dial("tcp", s.host, config)
	if err != nil {
		return err
	}

	s.client = client
	return nil
}

func (s *SSHService) getAuth(privateKey []byte) (ssh.AuthMethod, error) {
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	auth := ssh.PublicKeys(signer)
	return auth, nil
}

func (s *SSHService) getConfig(
	username string,
	auth ssh.AuthMethod,
	timeout time.Duration,
) *ssh.ClientConfig {
	cc := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}
	return cc
}
