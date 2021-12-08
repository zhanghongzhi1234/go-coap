package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/hongzhi/go-coap"
)

// User command
type userCommand struct {
	usageLine string
	run       func(c *userCommand, args []string)
}

var pathResponseMap map[string]string
var printLog bool

func (c *userCommand) name() string {
	name := c.usageLine
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

func (c *userCommand) usage() {
	fmt.Println(c.usageLine)
}

func init() {
	pathResponseMap = make(map[string]string)
	pathResponseMap["temperature"] = "15"
	pathResponseMap["temperature/enable"] = "10"
	printLog = false
}

//path is a string seperate with '/', and no leading '/', Message.SetPathString will also remove leading '/' when called by client
var cmdSet = &userCommand{
	usageLine: "set [path value] --set value by path, e.g set some/path 12",
	run: func(c *userCommand, args []string) {
		//fmt.Println(args)
		if len(args) == 1 {
			for k, v := range pathResponseMap {
				fmt.Println(k, v)
			}
		} else if len(args) == 2 {
			fmt.Println("No value entered")
		} else if len(args) == 3 {
			/*i := ""
			for k, _ := range pathResponseMap {
				if k == args[1] {
					i = k
					break
				}
			}
			if i == "" {
				c.usage()
				return
			}*/
			pathResponseMap[args[1]] = args[2]
		}
	},
}

var cmdExit = &userCommand{
	usageLine: "exit --quit program",
	run: func(c *userCommand, args []string) {
		os.Exit(3)
	},
}

var cmdShow = &userCommand{
	usageLine: "show --show all path value",
	run: func(c *userCommand, args []string) {
		for k, v := range pathResponseMap {
			fmt.Println(k, v)
		}
	},
}

var cmdOn = &userCommand{
	usageLine: "on --on /off log",
	run: func(c *userCommand, args []string) {
		printLog = !printLog
	},
}

var userCommands = []*userCommand{
	cmdSet,
	cmdExit,
	cmdShow,
	cmdOn,
}

func helpCmd() {
	fmt.Println("Support command")
	for _, c := range userCommands {
		c.usage()
	}
}

func args() []string {
	s, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err == nil {
		return strings.Fields(s)
	}
	return nil
}

func periodicTransmitter(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) {
	//subded := time.Now()
	msg := coap.Message{
		Type:      coap.Acknowledgement,
		Code:      coap.Content,
		MessageID: m.MessageID,
		Payload:   []byte(fmt.Sprintf(pathResponseMap[m.PathString()])),
	}

	if printLog {
		log.Printf("%v : %v -> %v", m.PathString(), pathResponseMap[m.PathString()], msg.Payload)
	}

	msg.SetOption(coap.ContentFormat, coap.TextPlain)
	msg.SetOption(coap.LocationPath, m.Path())

	if printLog {
		log.Printf("Transmitting %v", msg)
	}

	err := coap.Transmit(l, a, msg)
	if err != nil {
		log.Printf("Error on transmitter, stopping: %v", err)
		return
	}

	//time.Sleep(time.Second)
}
func runServer() {
	log.Fatal(coap.ListenAndServe("udp", ":5683",
		coap.FuncHandler(func(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) *coap.Message {
			if printLog {
				log.Printf("Got message path=%q: %#v from %v", m.Path(), m, a)
			}
			/*if m.Code == coap.GET && m.Option(coap.Observe) != nil {
				if value, ok := m.Option(coap.Observe).([]uint8); ok &&
					len(value) >= 1 && value[0] == 1 {
					go periodicTransmitter(l, a, m)
				}
			}*/
			if m.Code == coap.GET {
				go periodicTransmitter(l, a, m)
			} else if m.Code == coap.POST {
				pathResponseMap[m.PathString()] = string(m.Payload)
				log.Printf("set path=%v, value=%v from %v", m.PathString(), string(m.Payload), a)
				go periodicTransmitter(l, a, m)
			}
			return nil
		})))
}

func main() {
	go runServer()
	helpCmd()
	for {
		a := args()
		if len(a) > 0 {
			r := false
			for _, c := range userCommands {
				if c.name() == a[0] {
					c.run(c, a)
					r = true
					break
				}
			}
			if !r {
				helpCmd()
			}
		}
	}
}
