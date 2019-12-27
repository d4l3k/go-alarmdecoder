package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	fcm "github.com/appleboy/go-fcm"
	alarmdecoder "github.com/d4l3k/go-alarmdecoder"
	"github.com/foomo/simplecert"
	"github.com/gorilla/handlers"
	"github.com/jacobsa/go-serial/serial"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var (
	bind      = flag.String("bind", ":443", "address for webserver to listen on")
	mock      = flag.Bool("mock", false, "whether to mock out the alarm decoder device")
	email     = flag.String("email", "rice@fn.lc", "email address associated with cert")
	domain    = flag.String("domain", "ariel.fn.lc", "domain to get a SSL cert for")
	fcmKey    = flag.String("fcm", "", "api key for Firebase cloud messaging")
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
	alarmdecoder.Message
}

type ADBot struct {
	fcm *fcm.Client
	mu  struct {
		sync.Mutex

		// map from device ID to token
		registrationTokens map[string]string
		recentEvents       []Event
	}
}

func run() error {
	log.SetFlags(log.Lshortfile | log.Flags())
	var b ADBot
	b.mu.registrationTokens = map[string]string{}
	return b.Run()
}

func dropOldEvents(events []Event, before time.Time) []Event {
	keepIdx := sort.Search(len(events), func(i int) bool {
		return events[i].Time.After(before)
	})
	return events[keepIdx:]
}

func (b *ADBot) tokens() []string {
	b.mu.Lock()
	defer b.mu.Unlock()

	var tokens []string
	for _, token := range b.mu.registrationTokens {
		tokens = append(tokens, token)
	}
	return tokens
}

func (b *ADBot) broadcast(ctx context.Context, e Event, push bool) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	for _, token := range b.tokens() {
		log.Printf("sending to %s", token)
		_, err := b.fcm.SendWithContext(ctx, &fcm.Message{
			To: token,
			Notification: &fcm.Notification{
				Title: "Alarm Bot",
				Body:  e.KeypadMessage,
			},
			Data: map[string]interface{}{
				"event": e,
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func jsonHandler(f func(r *http.Request) (interface{}, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, err := f(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("%+v", err), 500)
			return
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, fmt.Sprintf("%+v", err), 500)
			return
		}
	})
}

func enforceAuth(h http.Handler, token string) http.Handler {
	if len(token) == 0 {
		log.Fatal("must specify an secret key")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header != ("Bearer " + token) {
			http.Error(w, "invalid token", http.StatusForbidden)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func (b *ADBot) registerHandler(r *http.Request) (interface{}, error) {
	type registerRequest struct {
		Token            string
		InstallationID   string
		DeviceName       string
		NativeAppVersion string
	}
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.mu.registrationTokens[req.InstallationID] = req.Token

	return nil, nil
}

func (b *ADBot) alarmHandler(r *http.Request) (interface{}, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.mu.recentEvents, nil
}

func (b *ADBot) thermostatHandler(r *http.Request) (interface{}, error) {
	return nil, errors.Errorf("unimplemented")
}

func (b *ADBot) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var err error
	b.fcm, err = fcm.NewClient(*fcmKey)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/register", jsonHandler(b.registerHandler))
	mux.Handle("/alarm", jsonHandler(b.alarmHandler))
	mux.Handle("/thermostat", jsonHandler(b.thermostatHandler))

	handler := enforceAuth(mux, *secretKey)
	handler = handlers.LoggingHandler(os.Stderr, handler)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		log.Printf("Listening on %s...", *bind)

		if *mock {
			return http.ListenAndServe(*bind, handler)
		} else {
			return simplecert.ListenAndServeTLS(*bind, handler, *email, cancel, *domain)
		}
	})

	var lastMsg string

	var ad AlarmReader
	if *mock {
		log.Printf("Using mock alarm decoder...")
		ad = mockAlarmDecoder{}
	} else {
		log.Printf("Reading from %s...", *port)
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
		ad = alarmdecoder.New(port)
	}

	eg.Go(func() error {
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

			e := Event{
				Time:    time.Now(),
				Message: msg,
			}
			b.addEvent(e)

			if err := b.broadcast(ctx, e, push); err != nil {
				log.Printf("broadcast error %+v", err)
			}
		}
		return nil
	})

	return eg.Wait()
}

func (b *ADBot) addEvent(e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.mu.recentEvents = dropOldEvents(b.mu.recentEvents, time.Now().Add(-maxAge))
	b.mu.recentEvents = append(b.mu.recentEvents, e)
}

type mockAlarmDecoder struct{}

func (mockAlarmDecoder) Read() (alarmdecoder.Message, error) {
	time.Sleep(5 * time.Second)
	return alarmdecoder.Message{
		KeypadMessage: "foo " + strconv.Itoa(rand.Intn(100)),
		AlarmSounding: rand.Float64() < 0.1,
	}, nil
}

type AlarmReader interface {
	Read() (alarmdecoder.Message, error)
}
