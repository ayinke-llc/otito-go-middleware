package otito

type Option func(*MessageStore)

func WithFilterFn(fn AppFilterMessageFn) Option {
	return func(ms *MessageStore) {
		ms.config.appFilterFn = fn
	}
}

func WithAppIDFn(fn AppIDFn) Option {
	return func(ms *MessageStore) {
		ms.config.appIDFn = fn
	}
}

func WithNumberOfMessagesBeforePublishing(v int64) Option {
	return func(ms *MessageStore) {
		ms.config.numberOfMessagesBeforePublishing = v
	}
}

func WithIPStrategy(i IPStrategy) Option {
	return func(ms *MessageStore) {
		ms.config.ipStrategy = i
	}
}

func WithAPIKey(s string) Option {
	return func(ms *MessageStore) {
		ms.config.apiKey = s
	}
}
