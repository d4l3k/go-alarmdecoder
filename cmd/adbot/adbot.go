package main

import (
	"flag"
	"log"

	alarmdecoder "github.com/d4l3k/go-alarmdecoder"
	"github.com/jacobsa/go-serial/serial"
)

var (
	port     = flag.String("port", "/dev/ttyAMA0", "serial port")
	baudRate = flag.Uint("baud", 115200, "baud rate of the serial port")
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("%+v", err)
	}
}

func run() error {
	options := serial.OpenOptions{
		PortName:        *port,
		BaudRate:        *baudRate,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 4,
	}
	port, err := serial.Open(options)
	if err != nil {
		return err
	}
	defer port.Close()

	ad := alarmdecoder.New(port)
	for {
		msg, err := ad.Read()
		if err != nil {
			log.Printf("error! %+v", err)
			continue
		}
		log.Printf("%+v", msg)
	}
}
