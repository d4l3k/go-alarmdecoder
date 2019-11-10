package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	alarmdecoder "github.com/d4l3k/go-alarmdecoder"
	"github.com/foomo/simplecert"
	"github.com/gorilla/handlers"
	"github.com/jacobsa/go-serial/serial"
	messenger "github.com/mileusna/facebook-messenger"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var (
	bind        = flag.String("bind", ":443", "address for webserver to listen on")
	email       = flag.String("email", "rice@fn.lc", "email address associated with cert")
	domain      = flag.String("domain", "ariel.fn.lc", "domain to get a SSL cert for")
	port        = flag.String("port", "/dev/ttyAMA0", "serial port")
	baudRate    = flag.Uint("baud", 115200, "baud rate of the serial port")
	accessToken = flag.String("accesstoken", "", "access token")
	verifyToken = flag.String("verifytoken", "", "verify token")
	pageID      = flag.String("pageid", "", "page ID")
	usersCSV    = flag.String("users", "", "users to send notifications to")
)

const readyMessage = "****DISARMED****  READY TO ARM"

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatalf("%+v", err)
	}
}

type ADBot struct {
	msng  *messenger.Messenger
	users []int64
}

func run() error {
	var b ADBot
	return b.Run()
}

func (b *ADBot) broadcast(s string, push bool) error {
	for _, user := range b.users {
		msg := b.msng.NewTextMessage(user, s)
		if !push {
			msg.NotificationType = messenger.NotificationTypeNoPush
		}
		if _, err := b.msng.SendMessage(msg); err != nil {
			return err
		}
	}
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

	if len(*accessToken) == 0 {
		return errors.Errorf("must specify an access token")
	}
	if len(*verifyToken) == 0 {
		return errors.Errorf("must specify an verify token")
	}
	if len(*pageID) == 0 {
		return errors.Errorf("must specify a page id")
	}
	if len(*usersCSV) == 0 {
		return errors.Errorf("must specify a user")
	}
	for _, user := range strings.Split(*usersCSV, ",") {
		id, err := strconv.ParseInt(user, 10, 64)
		if err != nil {
			return err
		}
		b.users = append(b.users, id)
	}

	b.msng = &messenger.Messenger{
		AccessToken: *accessToken,
		VerifyToken: *verifyToken,
		PageID:      *pageID,
	}
	b.msng.MessageReceived = b.messageReceived

	log.Printf("Listening...")

	mux := http.NewServeMux()

	mux.Handle("/callback", handlers.LoggingHandler(os.Stdout, b.msng))
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
