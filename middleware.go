package otito

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	otito "github.com/ayinke-llc/otito-go"
)

// used to retrieve the ID of the app
type AppIDFn func(r *http.Request) string

type MiddlewareConfig struct {
	appIDFn                          AppIDFn
	client                           *otito.APIClient
	numberOfMessagesBeforePublishing int64
	ipStrategy                       IPStrategy
	ingesterURL                      string
	apiKey                           string
}

type MessageHTTPDefinition struct {
	Header map[string][]string `json:"header"`
	Body   string              `json:"body"`
}

type Message struct {
	CreatedAt int64                 `json:"created_at"`
	App       string                `json:"app"`
	Request   MessageHTTPDefinition `json:"request"`
	Response  MessageHTTPDefinition `json:"response"`
	IPAddress string                `json:"ip_address"`
}

func New(apiKey string, ingesterURL string, fn AppIDFn, numberOfMessagesBeforePublishing int64, ipstrat IPStrategy) *MessageStore {
	cfg := otito.NewConfiguration()

	client := otito.NewAPIClient(cfg)

	return &MessageStore{
		messages: make([]Message, 0),
		mutex:    sync.RWMutex{},
		config: MiddlewareConfig{
			appIDFn:                          fn,
			numberOfMessagesBeforePublishing: numberOfMessagesBeforePublishing,
			ipStrategy:                       ipstrat,
			client:                           client,
			apiKey:                           apiKey,
		},
	}
}

type MessageStore struct {
	messages []Message
	mutex    sync.RWMutex
	config   MiddlewareConfig
}

func (m *MessageStore) flush() error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	ctx := context.WithValue(context.Background(), otito.ContextAPIKeys, map[string]otito.APIKey{
		"ApiKeyAuth": {
			Key:    m.config.apiKey,
			Prefix: "Bearer",
		},
	})

	msg := otito.NewServerMessageRequest()

	for _, v := range m.messages {
		c := int32(v.CreatedAt)

		msg.Messages = append(msg.Messages, otito.ServerMessageRequestMessagesInner{
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
		})
	}

	status, resp, err := m.config.client.MessageApi.MessagesPost(ctx).
		Message(*msg).Execute()
	if err != nil {
		panic("oops")
	}

	if resp.StatusCode > http.StatusContinue {
		panic("oops")
	}

	fmt.Println(status)
	if !status.GetStatus() {
		panic("oops")
	}

	return nil
}

func (m *MessageStore) add(msg Message) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.messages = append(m.messages, msg)

	if len(m.messages) >= int(m.config.numberOfMessagesBeforePublishing) {
		go m.flush()
	}
}

func (m *MessageStore) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		rec := httptest.NewRecorder()

		next.ServeHTTP(rec, r)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}

		r.Body = io.NopCloser(r.Body)

		b := bytes.NewBuffer(nil)

		_, err := io.Copy(b, r.Body)
		if err != nil {
			// remove this rubbish
			// TODO(adelowo): please ffs
			panic("oops")
		}

		w.WriteHeader(rec.Code)

		buf := rec.Body.Bytes()

		w.Write(buf)

		msg := Message{
			CreatedAt: time.Now().Unix(),
			App:       m.config.appIDFn(r),
			IPAddress: getIP(r, m.config.ipStrategy),
			Request: MessageHTTPDefinition{
				Header: r.Header.Clone(),
				Body:   b.String(),
			},
			Response: MessageHTTPDefinition{
				Header: rec.Header().Clone(),
				Body:   string(buf),
			},
		}

		m.add(msg)
	})
}
