package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jasonlvhit/gocron"
	"net/http"
	"os"
)

const (
	SensorsHost            = "SENSORS_HOST"
	MqttSenderHost         = "MQTT_SENDER_HOST"
	HomeSensorsHumidity    = "home/sensors/humidity"
	HomeSensorsTemperature = "home/sensors/temperature"
	Table                  = "climate"
)

func main() {
	_ = gocron.Every(45).Second().Do(scanSensors)
	<-gocron.Start()
}

func scanSensors() {
	response := Response{}
	if err := getJSON("http://"+getSensorsHost(), &response); err == nil {
		for _, v := range response.Dht22 {
			if v.Status == http.StatusText(http.StatusOK) {
				_ = sendMessage(getMessage(HomeSensorsTemperature, fmt.Sprintf("%s,pin=%d value=%.2f", Table, v.Pin, v.Temperature)))
				_ = sendMessage(getMessage(HomeSensorsHumidity, fmt.Sprintf("%s,pin=%d value=%.2f", Table, v.Pin, v.Humidity)))
			}
		}
		for _, v := range response.Ds18b20 {
			if v.Status == http.StatusText(http.StatusOK) {
				_ = sendMessage(getMessage(HomeSensorsTemperature, fmt.Sprintf("%s,pin=%d,dec=%s value=%.2f", Table, v.Pin, v.Dec, v.Temperature)))
			}
		}
	}
}

func getMessage(topic string, payload string) Message {
	return Message{Topic: topic, Qos: 2, Retained: false, Payload: payload}
}

func getJSON(url string, result interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("cannot fetch URL %q: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected http GET status: %s", resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		return fmt.Errorf("cannot decode JSON: %v", err)
	}
	return nil
}

func sendMessage(message Message) error {
	uri := fmt.Sprintf("http://%s/publish", getMqttSenderHost())
	//fmt.Printf("POST: %s BODY: %v\n", uri, message)
	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(message)
	if err != nil {
		return fmt.Errorf("%q: %v", uri, err)
	}
	resp, err := http.Post(uri, "application/json; charset=utf-8", body)
	if err != nil {
		return fmt.Errorf("cannot fetch URL %q: %v", uri, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected http POST status: %s", resp.Status)
	}

	return nil
}

func getSensorsHost() string {
	return os.Getenv(SensorsHost)
}

func getMqttSenderHost() string {
	return os.Getenv(MqttSenderHost)
}

type dht22 struct {
	Pin         int     `json:"pin"`
	Temperature float32 `json:"temperature"`
	Humidity    float32 `json:"humidity"`
	Status      string  `json:"status"`
}
type ds18b20 struct {
	Pin         int     `json:"pin"`
	Temperature float32 `json:"temperature"`
	Dec         string  `json:"dec"`
	Status      string  `json:"status"`
}

type Response struct {
	Dht22   []dht22   `json:"dht22"`
	Ds18b20 []ds18b20 `json:"ds18b20"`
}

type Message struct {
	Topic    string `json:"topic"`
	Qos      int    `json:"qos"`
	Retained bool   `json:"retained"`
	Payload  string `json:"payload"`
}
