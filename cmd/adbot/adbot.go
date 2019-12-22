package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"sort"
	"time"

	alarmdecoder "github.com/d4l3k/go-alarmdecoder"
	"github.com/foomo/simplecert"
	"github.com/jacobsa/go-serial/serial"
	messenger "github.com/mileusna/facebook-messenger"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var (
	bind      = flag.String("bind", ":443", "address for webserver to listen on")
	email     = flag.String("email", "rice@fn.lc", "email address associated with cert")
	domain    = flag.String("domain", "ariel.fn.lc", "domain to get a SSL cert for")
	port      = flag.String("port", "/dev/ttyAMA0", "serial port")
	baudRate  = flag.Uint("baud", 115200, "baud rate of the serial port")
	secretKey = flag.String("secret", "", "shared secret with clients")
)

const (
	readyMessage = "****DISARMED****  READY TO ARM"
	maxAge       = 7 * 24 * time.Hour
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatalf("%+v", err)
	}
}

type Event struct {
	Time time.Time
	Body string
}

type ADBot struct {
	recentEvents []Event
}

func run() error {
	var b ADBot
	return b.Run()
}

func dropOldEvents(events []Event, before time.Time) []Event {
	keepIdx := sort.Search(len(events), func(i int) bool {
		return events[i].Time.After(before)
	})
	return events[keepIdx:]
}

func (b *ADBot) broadcast(s string, push bool) error {
	b.recentEvents = dropOldEvents(b.recentEvents, time.Now().Add(-maxAge))
	b.recentEvents = append(b.recentEvents, Event{Time: time.Now(), Body: s})
	return nil
}

func (b *ADBot) Run() error {
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

	if len(*secretKey) == 0 {
		return errors.Errorf("must specify an secret key")
	}

	log.Printf("Listening...")

	mux := http.NewServeMux()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return simplecert.ListenAndServeTLS(*bind, mux, *email, cancel, *domain)
	})

	var lastMsg string

	eg.Go(func() error {
		ad := alarmdecoder.New(port)
		for ctx.Err() == nil {
			msg, err := ad.Read()
			if err != nil {
				log.Printf("error! %+v", err)
				continue
			}
			log.Printf("%+v", msg)

			if msg.KeypadMessage == lastMsg {
				continue
			}
			lastMsg = msg.KeypadMessage

			push := msg.Beeps > 0 || msg.AlarmSounding

			if !push && msg.KeypadMessage == readyMessage {
				continue
			}

			if err := b.broadcast(msg.KeypadMessage, push); err != nil {
				log.Printf("broadcast error %+v", err)
			}
		}
		return nil
	})

	return eg.Wait()
}

func (b *ADBot) messageReceived(msng *messenger.Messenger, userID int64, m messenger.FacebookMessage) {
	log.Printf("message from %d: %q", userID, m.Text)
}
