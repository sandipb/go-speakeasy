package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	zmq "github.com/pebbe/zmq4"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var percentiles = []float64{0.5, 0.75, 0.95, 0.99}

type MetricInput struct {
	App    string
	Metric string
	Type   string
	Value  float64
}

const (
	metricInputBuffer      = 1000 // channel buffer for queueing incoming metrics
	metricSocketHWM        = 20000
	socketReadPollDuration = 1 * time.Second
)

// SpeakEasyServer reads client submissions from the metrics socket and sends them to an emitter
// at a configurable interval
type SpeakEasyServer struct {
	socketPath string
	//	interval         time.Duration
	metricsReader    *metricsReader
	metricsInputChan chan MetricInput
	logger           log.FieldLogger
	metricsCancel    context.CancelFunc
	wg               sync.WaitGroup
}

//  NewSpeakeasyServer creates a speakeast server listening to socketpath and emitting every interval
func NewSpeakeasyServer(socketPath string) (*SpeakEasyServer, error) {
	//func NewSpeakeasyServer(socketPath string, interval time.Duration) (*SpeakEasyServer, error) {

	miChan := make(chan MetricInput, metricInputBuffer)

	return &SpeakEasyServer{
		socketPath: socketPath,
		//		interval:         interval,
		metricsInputChan: miChan,
		metricsReader: &metricsReader{
			socketPath:        socketPath,
			metricsOutputChan: miChan,
			logger:            logger.WithField("section", "metricsReader"),
		},
		logger: logger.WithField("section", "speakeasyServer"),
	}, nil
}

// start will start the metrics reading thread which will accept submissions
// from clients
func (server *SpeakEasyServer) start() {
	var newCtx context.Context

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGPIPE)

	server.wg.Add(1)
	go server.signalHandler(sigChan)

	server.wg.Add(1)
	newCtx, server.metricsCancel = context.WithCancel(context.Background())

	go server.metricsReader.receiveMetrics(newCtx, &server.wg)
	server.logger.Info("Metrics reader started")

	server.wg.Wait()

	server.logger.Info("Metrics reader has exited")
}

// handleSignals catches external signals and calls the cancel function
func (server *SpeakEasyServer) signalHandler(s chan os.Signal) {
	defer server.wg.Done()
	logger.Infof("Got signal %d", <-s)
	server.logger.Info("Cancelling metrics reader")
	server.metricsCancel()
}

type metricsReader struct {
	socketPath        string
	logger            log.FieldLogger
	metricsOutputChan chan MetricInput
}

func (mr *metricsReader) receiveMetrics(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	sock, err := zmq.NewSocket(zmq.PULL)
	if err != nil {
		mr.logger.Errorf("Could not create ZMQ socket: %v", err)
		return
	}
	defer sock.Close()

	err = sock.SetRcvhwm(metricSocketHWM)
	if err != nil {
		mr.logger.Errorf("Could not set HWM", err)
		return
	}

	err = sock.Bind(fmt.Sprintf("ipc://%s", mr.socketPath))
	if err != nil {
		mr.logger.Errorf("Could not bind server to %s: %v", mr.socketPath, err)
		return
	}

	mr.logger.Infof("Bound to ipc://%s", mr.socketPath)
	poller := zmq.NewPoller()
	poller.Add(sock, zmq.POLLIN)

	count := 0
LOOP:
	for {
		// non blocking read on context cancellation
		select {
		case <-ctx.Done():
			mr.logger.Info("Stopping")
			break LOOP
		default:
		}

		sockets, err := poller.Poll(socketReadPollDuration)
		if err != nil {
			mr.logger.Errorf("Polling error: %v", err)
			return
		}
		if len(sockets) < 1 {
			continue
		}

		data, err := sock.RecvBytes(0)
		if err != nil {
			mr.logger.Errorf("Error reading from socket: %v", err)
			continue
		}
		raw := []interface{}{}
		err = json.Unmarshal(data, &raw)
		if err != nil {
			mr.logger.Errorf("Error decoding client input: %v", err)
			mr.logger.Errorf("Had got input: %s", string(data))
			continue
		}
		mr.logger.Infof("Received: client input: %q", string(data))
		count++

	}
	logger.Infof("Stop receiving metrics. %d metrics received", count)
}

// runServerCommand implements the 'server' command
func runServerCommand(cmd *cobra.Command, args []string) {
	server, err := NewSpeakeasyServer(metricsSocket)
	if err != nil {
		logger.Fatalf("Error creating server: %v", err)
	}

	server.start()
}
