package logger

import (
	"flag"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Encoder is an enum for the log encoders.
type Encoder string

// Logger is a wrapper for the log encoder.
type Logger struct {
	Encoder string
}

const (
	logEncoderFlag = "log-encoder"

	// EncoderConsole is the console encoder.
	EncoderConsole Encoder = "console"

	// EncoderJSON is the json encoder.
	EncoderJSON Encoder = "json"
)

// New returns a new logger with console encoder as the default.
func New() *Logger {
	return &Logger{
		Encoder: string(EncoderConsole),
	}
}

// AddFlags adds flags for the logger.
func (l *Logger) AddFlags() {
	flag.StringVar(&l.Encoder, logEncoderFlag, string(EncoderConsole), fmt.Sprintf("Sets the log encoder (%s|%s)", EncoderConsole, EncoderJSON))
}

// Get returns a new logr.Logger according to the encoder.
func (l *Logger) Get() logr.Logger {
	switch Encoder(l.Encoder) {
	case EncoderConsole:
		// no-op
	case EncoderJSON:
		return zap.New()
	default:
		klogr.New().WithName("logger").Info("unknown log encoder, using console", "encoder", l.Encoder)
	}
	return klogr.New()
}
