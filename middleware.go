package otito

import (
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

type MessageStore struct {
	messages []Message
	mutex    sync.RWMutex
	config   MiddlewareConfig
}

func (m *MessageStore) flush() error {
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

		w.WriteHeader(rec.Code)

		buf := rec.Body.Bytes()

		w.Write(buf)

		msg := &Message{
			CreatedAt: time.Now().Unix(),
			App:       m.config.appIDFn(r),
			IPAddress: getIP(r, m.config.ipStrategy),
			Request: MessageHTTPDefinition{
				Header: r.Header.Clone(),
			},
		}

		msg.Response.Header = rec.Header().Clone()
		msg.Response.Body = string(buf)
	})
}
