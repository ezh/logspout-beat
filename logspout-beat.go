package beat

import (
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/gliderlabs/logspout/router"
	"time"
	"github.com/elastic/beats/libbeat/publisher"
)

func init() {
	router.AdapterFactories.Register(NewLogspoutBeat, "beat")
}

type LogspoutBeat struct {
	open   chan bool
	isOpen bool
	beat   *beat.Beat
	client publisher.Client
}

func NewLogspoutBeat(route *router.Route) (router.LogAdapter, error) {
	logspoutBeat := &LogspoutBeat{open: make(chan bool, 1)}
	go func() {
		if err := beat.Run("logspout-beat", "1.0.0", logspoutBeat); err != nil {
			fmt.Errorf("error starting beat", err)
			return
		}
	}()
	return logspoutBeat, nil
}

func (logspotBeat *LogspoutBeat) Config(b *beat.Beat) error {
	return nil
}

func (logspotBeat *LogspoutBeat) Setup(b *beat.Beat) error {
	logspotBeat.beat = b
	logspotBeat.client = b.Publisher.Connect()
	logspotBeat.open <- true
	return nil
}

func (logspotBeat *LogspoutBeat) Run(b *beat.Beat) error {
	select {}
	return nil
}

func (logspotBeat *LogspoutBeat) Cleanup(b *beat.Beat) error {
	return nil
}

func (logspotBeat *LogspoutBeat) Stop() {
	logspotBeat.open <- false
}

func (logspoutBeat *LogspoutBeat) Stream(logstream chan *router.Message) {
	fmt.Println("Streaming logspout")
	for {
		select {
		case shouldOpen := <-logspoutBeat.open:
			if !shouldOpen {
				fmt.Println("Closing beat")
				return
			}
			fmt.Println("Opening beat")
			logspoutBeat.isOpen = true
		case msg, ok := <-logstream:
			if !ok {
				return
			}
			if logspoutBeat.isOpen {
				dockerInfo := DockerInfo{
					Name:     msg.Container.Name,
					ID:       msg.Container.ID,
					Image:    msg.Container.Config.Image,
					Hostname: msg.Container.Config.Hostname,
				}

				v := common.MapStr{}

				v["type"] = "dockerlog"

				err := json.Unmarshal([]byte(msg.Data), &v)
				if err != nil {
					// The message was not JSON, create a new JSON message
					v["message"] = msg.Data
				}

				v["docker"] = dockerInfo
				if err := v.EnsureTimestampField(func() time.Time { return msg.Time }); err != nil {
					fmt.Println("logspout-beat:", err)
					return
				}
				logspoutBeat.client.PublishEvent(v)
			}
		}
	}
}

type DockerInfo struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	Image    string `json:"image"`
	Hostname string `json:"hostname"`
}
