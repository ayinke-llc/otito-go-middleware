package otito

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	otito "github.com/ayinke-llc/otito-go"
	"github.com/ayinke-llc/otito-go-middleware/util"
)

// used to retrieve the ID of the app
type AppIDFn func(r *http.Request) string

// used to determine if we should add this message.
// Sometimes, you want to only set this log for a select few users
// By default, all messages are recorded. Implement your logic with this
// function type to be selective of which messages are recorded
type AppFilterMessageFn func(r *http.Request) bool

var defaultHeadersToStrip = []string{"Authorization"}

func defaultFilterFn(_ *http.Request) bool { return true }

type middlewareConfig struct {
	appIDFn                          AppIDFn
	appFilterFn                      AppFilterMessageFn
	client                           *otito.APIClient
	numberOfMessagesBeforePublishing int64
	ipStrategy                       IPStrategy
	apiKey                           string
	headersToStrip                   []string
}

type messageHTTPDefinition struct {
	Header map[string][]string `json:"header"`
	Body   string              `json:"body"`
}

type message struct {
	CreatedAt  int64                 `json:"created_at"`
	App        string                `json:"app"`
	Request    messageHTTPDefinition `json:"request"`
	Response   messageHTTPDefinition `json:"response"`
	IPAddress  string                `json:"ip_address"`
	Path       string                `json:"path"`
	Method     string                `json:"method"`
	StatusCode int                   `json:"status_code"`
}

func New(opts ...Option) (*MessageStore, error) {
	cfg := otito.NewConfiguration()

	client := otito.NewAPIClient(cfg)

	msg := &MessageStore{
		messages: make([]message, 0),
		mutex:    sync.RWMutex{},
		config: middlewareConfig{
			appIDFn:                          func(r *http.Request) string { return "" },
			numberOfMessagesBeforePublishing: 100,
			ipStrategy:                       ForwardedOrRealIPStrategy,
			client:                           client,
			apiKey:                           "",
			appFilterFn:                      defaultFilterFn,
			headersToStrip:                   defaultHeadersToStrip,
		},
	}

	for _, opt := range opts {
		opt(msg)
	}

	if msg.config.numberOfMessagesBeforePublishing > 1000 {
		return nil, errors.New("you can only batch a maximum of 1,000 requests at a time")
	}

	if util.IsStringEmpty(msg.config.apiKey) {
		return nil, errors.New("please provide an api key")
	}

	return msg, nil
}

type MessageStore struct {
	messages []message
	mutex    sync.RWMutex
	config   middlewareConfig
}

func maskLeft(s string) string {
	return strings.Repeat("*", 15)
}

func (m *MessageStore) flush() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	ctx := context.WithValue(context.Background(), otito.ContextAPIKeys, map[string]otito.APIKey{
		"ApiKeyAuth": {
			Key:    m.config.apiKey,
			Prefix: "Bearer",
		},
	})

	msg := otito.NewServerMessageRequest()

	for _, v := range m.messages {
		c := int32(v.CreatedAt)

		for _, value := range m.config.headersToStrip {
			for hdKey, headerValue := range v.Request.Header[value] {
				v.Request.Header[value][hdKey] = maskLeft(headerValue)
			}
		}

		var ss int32 = int32(v.StatusCode)
		msg.Messages = append(msg.Messages, otito.ServerMRequest{
			App:       &v.App,
			CreatedAt: &c,
			IpAddress: &v.IPAddress,
			Request: &otito.ServerMessageHTTPDefinition{
				Body:   &v.Request.Body,
				Header: &v.Request.Header,
			},
			Response: &otito.ServerMessageHTTPDefinition{
				Body:   &v.Response.Body,
				Header: &v.Response.Header,
			},
			Method:     &v.Method,
			Path:       &v.Path,
			StatusCode: &ss,
		})
	}

	status, resp, err := m.config.client.MessageApi.MessagesPost(ctx).
		Message(*msg).Execute()
	if err != nil {
		return err
	}

	if resp.StatusCode > http.StatusAccepted {
		return err
	}

	if !status.GetStatus() {
		return errors.New("an error occurred")
	}

	m.messages = make([]message, 0)
	return nil
}

func (m *MessageStore) add(msg message) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.messages = append(m.messages, msg)

	if len(m.messages) >= int(m.config.numberOfMessagesBeforePublishing) {
		go m.flush()
	}
}

func (m *MessageStore) Close() error { return m.flush() }

func (m *MessageStore) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		rec := httptest.NewRecorder()

		// 25MB max
		r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*(25))

		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status":false, "message":"internal error"}`))
			return
		}

		r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		// close here since we put this in a no closer
		defer r.Body.Close()

		next.ServeHTTP(rec, r)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}

		w.WriteHeader(rec.Code)

		buf := rec.Body.Bytes()

		w.Write(buf)

		if !m.config.appFilterFn(r) {
			return
		}

		msg := message{
			CreatedAt: time.Now().Unix(),
			App:       m.config.appIDFn(r),
			IPAddress: getIP(r, m.config.ipStrategy),
			Request: messageHTTPDefinition{
				Header: r.Header.Clone(),
				Body:   string(bodyBytes),
			},
			Response: messageHTTPDefinition{
				Header: rec.Header().Clone(),
				Body:   string(buf),
			},
			Path:       r.URL.Path,
			Method:     r.Method,
			StatusCode: rec.Code,
		}

		go m.add(msg)
	})
}
