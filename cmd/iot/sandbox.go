/*
Copyright © 2024 ks6088ts

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package iot

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/eclipse/paho.golang/paho"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// https://github.com/Azure-Samples/MqttApplicationSamples/blob/main/scenarios/getting_started/go/getting_started.go
type mqttConnectionSettings struct {
	Hostname        string
	TcpPort         int
	UseTls          bool
	CleanSession    bool
	CaFile          string
	CertFile        string
	KeyFile         string
	KeyFilePassword string
	KeepAlive       uint16
	ClientId        string
	Username        string
	Password        string
}

var mqttSettingNames = [12]string{
	"MQTT_HOST_NAME",
	"MQTT_TCP_PORT",
	"MQTT_USE_TLS",
	"MQTT_CLEAN_SESSION",
	"MQTT_KEEP_ALIVE_IN_SECONDS",
	"MQTT_CLIENT_ID",
	"MQTT_USERNAME",
	"MQTT_PASSWORD",
	"MQTT_CA_FILE",
	"MQTT_CERT_FILE",
	"MQTT_KEY_FILE",
	"MQTT_KEY_FILE_PASSWORD",
}

var defaults = map[string]string{
	"MQTT_TCP_PORT":              "8883",
	"MQTT_USE_TLS":               "true",
	"MQTT_CLEAN_SESSION":         "true",
	"MQTT_KEEP_ALIVE_IN_SECONDS": "30",
}

func parseIntValue(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func parseBoolValue(value string) bool {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func getTlsConnection(cs mqttConnectionSettings) *tls.Conn {

	cfg := &tls.Config{}

	if cs.CertFile != "" && cs.KeyFile != "" {
		if cs.KeyFilePassword != "" {
			log.Fatal("Password protected key files are not supported at this time.")
		}

		cert, err := tls.LoadX509KeyPair(cs.CertFile, cs.KeyFile)
		if err != nil {
			log.Fatal(err)
		}

		cfg.Certificates = []tls.Certificate{cert}
	}

	if cs.CaFile != "" {
		ca, err := os.ReadFile(cs.CaFile)
		if err != nil {
			panic(err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(ca)
		cfg.RootCAs = caCertPool
	}

	fmt.Println(cs.Hostname)
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", cs.Hostname, cs.TcpPort), cfg)
	if err != nil {
		panic(err)
	}

	return conn
}

func loadConnectionSettings(path string) mqttConnectionSettings {
	if err := godotenv.Load(path); err != nil {
		log.Fatalf("could not load .env file: %s", err)
	}
	cs := mqttConnectionSettings{}
	envVars := make(map[string]string)

	// Check to see which env vars are set
	for i := 0; i < len(mqttSettingNames); i++ {
		name := mqttSettingNames[i]
		value := os.Getenv(name)
		// If var is not set, check if it has a default value
		if value == "" && defaults[name] != "" {
			value = defaults[name]
		}

		envVars[name] = value
	}

	// Based on which vars are set, construct MqttConnectionSettings
	cs.Hostname = envVars["MQTT_HOST_NAME"]
	cs.TcpPort = parseIntValue(envVars["MQTT_TCP_PORT"])
	cs.UseTls = parseBoolValue(envVars["MQTT_USE_TLS"])
	cs.CleanSession = parseBoolValue(envVars["MQTT_CLEAN_SESSION"])
	cs.KeepAlive = uint16(parseIntValue(envVars["MQTT_KEEP_ALIVE_IN_SECONDS"]))
	cs.ClientId = envVars["MQTT_CLIENT_ID"]
	cs.Username = envVars["MQTT_USERNAME"]
	cs.Password = envVars["MQTT_PASSWORD"]
	cs.CaFile = envVars["MQTT_CA_FILE"]
	cs.CertFile = envVars["MQTT_CERT_FILE"]
	cs.KeyFile = envVars["MQTT_KEY_FILE"]
	cs.KeyFilePassword = envVars["MQTT_KEY_FILE_PASSWORD"]

	return cs
}

// sandboxCmd represents the sandbox command
var sandboxCmd = &cobra.Command{
	Use:   "sandbox",
	Short: "Sandboxes the Paho MQTT client",
	Long:  `This command will create a Paho MQTT client and connect to the specified broker.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Parse flags
		env, err := cmd.Flags().GetString("env")
		if err != nil {
			log.Fatalf("could not get `env` flag: %s", err)
		}
		var cs mqttConnectionSettings = loadConnectionSettings(env)

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		fmt.Println("Creating Paho client")
		c := paho.NewClient(paho.ClientConfig{
			Router: paho.NewSingleHandlerRouter(func(m *paho.Publish) {
				fmt.Printf("received message on topic %s; body: %s (retain: %t)\n", m.Topic, m.Payload, m.Retain)
			}),
			OnClientError: func(err error) { fmt.Printf("server requested disconnect: %s\n", err) },
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					fmt.Printf("server requested disconnect: %s\n", d.Properties.ReasonString)
				} else {
					fmt.Printf("server requested disconnect; reason code: %d\n", d.ReasonCode)
				}
			},
		})

		if cs.UseTls {
			c.Conn = getTlsConnection(cs)
		} else {
			conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", cs.Hostname, cs.TcpPort))
			if err != nil {
				panic(err)
			}
			c.Conn = conn
		}

		cp := &paho.Connect{
			KeepAlive:  cs.KeepAlive,
			ClientID:   cs.ClientId,
			CleanStart: cs.CleanSession,
		}

		if cs.Username != "" {
			cp.Username = cs.Username
			cp.UsernameFlag = true
		}

		if cs.Password != "" {
			cp.Password = []byte(cs.Password)
			cp.PasswordFlag = true
		}

		fmt.Printf("Attempting to connect to %s:%d\n", cs.Hostname, cs.TcpPort)
		ca, err := c.Connect(ctx, cp)
		if err != nil {
			log.Fatalln(err)
		}
		if ca.ReasonCode != 0 {
			log.Fatalf("Failed to connect to %s : %d - %s", cs.Hostname, ca.ReasonCode, ca.Properties.ReasonString)
		}

		fmt.Printf("Connection successful")
		if _, err := c.Subscribe(ctx, &paho.Subscribe{
			Subscriptions: []paho.SubscribeOptions{
				{Topic: "sample/+", QoS: byte(1)},
			},
		}); err != nil {
			log.Fatalf("could not subscribe to topic: %s", err)
		}

		if _, err := c.Publish(context.Background(), &paho.Publish{
			Topic:   "sample/topic1",
			QoS:     byte(1),
			Retain:  false,
			Payload: []byte("hello world"),
		}); err != nil {
			log.Fatalf("could not publish message: %s", err)
		}

		<-ctx.Done() // Wait for user to trigger exit
		fmt.Println("signal caught - exiting")
	},
}

func init() {
	iotCmd.AddCommand(sandboxCmd)

	sandboxCmd.Flags().StringP("env", "e", "", "Path to .env file")

	if err := sandboxCmd.MarkFlagRequired("env"); err != nil {
		log.Fatalf("could not mark `env` as required: %s", err)
	}
}
