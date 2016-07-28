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
		if err := beat.Run("logspout-beat", "1.0.0", logspoutFactory(logspoutBeat)); err != nil {
			fmt.Errorf("error starting beat", err)
			return
		}
	}()
	return logspoutBeat, nil
}

func logspoutFactory(logspotBeat *LogspoutBeat) beat.Creator {
	return func(beat *beat.Beat, config *common.Config) (beat.Beater, error) {
		logspotBeat.beat = beat
		logspotBeat.client = beat.Publisher.Connect()
		logspotBeat.open <- true
		return logspotBeat, nil
	}
}

func (logspotBeat *LogspoutBeat) Run(b *beat.Beat) error {
	select {}
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
				EnsureTimestampField(v, msg.Time)
				logspoutBeat.client.PublishEvent(v)
			}
		}
	}
}

const TsLayout = "2006-01-02T15:04:05.000Z"

func EnsureTimestampField(m common.MapStr, t time.Time) {
	ts, exists := m["@timestamp"]
	if !exists {
		m["@timestamp"] = common.Time(t)
		return
	}

	_, is_common_time := ts.(common.Time)
	if is_common_time {
		// already perfect
		return
	}

	tstime, is_time := ts.(time.Time)
	if is_time {
		m["@timestamp"] = time.Time(tstime)
		return
	}

	tsstr, is_string := ts.(string)
	if is_string {
		var err error
		m["@timestamp"], err = common.ParseTime(tsstr)
		if err == nil {
			return
		}

		// Wrong format let's try RFC3339
		timeValue, err := time.Parse(time.RFC3339, tsstr)
		if err == nil {
			m["@timestamp"] = common.Time(timeValue)
			return
		}
	}

	// No know format use docker timestamp
	m["@timestamp"] = common.Time(t)
	return
}

type DockerInfo struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	Image    string `json:"image"`
	Hostname string `json:"hostname"`
}
