package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	labelTopic     = "topic"
	labelErrorCode = "code"

	workerReader = "reader"
	workerWriter = "writer"
)

type Reader interface {
	AddReaderTopicsInUse() Reader
	RmReaderTopicsInUse() Reader
	AddReaderTopicFatalError(topic string, code int) Reader
	AddReaderTopicProcessError(topic string) Reader
	AddReaderTopicReadEvents(topic string, num int) Reader
	AddReaderTopicCommitEvents(topic string, num int) Reader
	AddReaderTopicDecodeEvents(topic string, num int) Reader
	AddReaderTopicSkippedEvents(topic string, num int) Reader
	AddReaderTopicErrorEvents(topic string, num int) Reader
	AddReaderTopicHandlingTime(topic string, duration time.Duration) Reader

	AddGarbageRecords(num int) Reader
}

type mReader struct {
	// new reader
	// topics currently in process
	ReaderTopicsInUse prometheus.Gauge

	// error - start reconnection by topic
	ReaderTopicFatalError *prometheus.CounterVec
	// process error (include decode fatal, postgres error) - start reconnection by topic
	ReaderTopicProcessError *prometheus.CounterVec

	// counter read events by topic
	ReaderTopicReadEvents *prometheus.CounterVec
	// counter commit events by topic
	ReaderTopicCommitEvents *prometheus.CounterVec

	// count all decodes events (except skipped & error)
	ReaderTopicDecodeEvents  *prometheus.CounterVec
	ReaderTopicSkippedEvents *prometheus.CounterVec
	ReaderTopicErrorEvents   *prometheus.CounterVec

	// count success batch insertion by topic
	ReaderHandlingTime *prometheus.GaugeVec

	garbageRecords prometheus.Gauge
}

func NewCollectorMetricsReader() Reader {
	// new reader
	readerTopicsInUse := prometheus.NewGauge(prometheus.GaugeOpts{
		Subsystem: workerReader,
		Name:      "consuming_topics",
		Help:      "Number of topics consumed now",
	})

	readerTopicFatalError := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: workerReader,
		Name:      "consuming_fatal_errors",
		Help:      "Number of topic critical (start reconnection) errors",
	}, []string{labelTopic, labelErrorCode})

	readerTopicProcessError := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: workerReader,
		Name:      "consuming_process_error",
		Help:      "Number of process errors on topic",
	}, []string{labelTopic})

	readerTopicReadEvents := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: workerReader,
		Name:      "events_read",
		Help:      "Number of read events on topic",
	}, []string{labelTopic})

	readerTopicCommitEvents := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: workerReader,
		Name:      "events_commit",
		Help:      "Number of commit events on topic",
	}, []string{labelTopic})

	readerTopicDecodeEvents := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: workerReader,
		Name:      "events_decode",
		Help:      "Number of decode events on topic",
	}, []string{labelTopic})

	readerTopicSkippedEvents := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: workerReader,
		Name:      "events_skipped",
		Help:      "Number of skipped events on topic",
	}, []string{labelTopic})

	readerTopicErrorEvents := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: workerReader,
		Name:      "events_error",
		Help:      "Number of events on topic",
	}, []string{labelTopic})

	readerHandlingTime := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: workerReader,
		Name:      "handling_time",
		Help:      "Processing time of received messages by topic processors in seconds",
	}, []string{labelTopic})

	garbageRecords := prometheus.NewGauge(prometheus.GaugeOpts{
		Subsystem: workerWriter,
		Name:      "garbage_records_total",
		Help:      "Number of garbage records",
	})

	prometheus.DefaultRegisterer.MustRegister(
		readerTopicsInUse, readerTopicFatalError, readerTopicProcessError, readerTopicReadEvents,
		readerTopicCommitEvents, readerTopicDecodeEvents, readerTopicSkippedEvents, readerTopicErrorEvents,
		readerHandlingTime,

		garbageRecords,
	)

	return &mReader{
		ReaderTopicsInUse:        readerTopicsInUse,
		ReaderTopicFatalError:    readerTopicFatalError,
		ReaderTopicProcessError:  readerTopicProcessError,
		ReaderTopicReadEvents:    readerTopicReadEvents,
		ReaderTopicCommitEvents:  readerTopicCommitEvents,
		ReaderTopicDecodeEvents:  readerTopicDecodeEvents,
		ReaderTopicSkippedEvents: readerTopicSkippedEvents,
		ReaderTopicErrorEvents:   readerTopicErrorEvents,
		ReaderHandlingTime:       readerHandlingTime,

		garbageRecords: garbageRecords,
	}
}

// new reader
func (m *mReader) AddReaderTopicsInUse() Reader {
	m.ReaderTopicsInUse.Inc()

	return m
}

func (m *mReader) RmReaderTopicsInUse() Reader {
	m.ReaderTopicsInUse.Dec()

	return m
}

func (m *mReader) AddReaderTopicFatalError(topic string, code int) Reader {
	m.ReaderTopicFatalError.With(map[string]string{
		labelTopic:     topic,
		labelErrorCode: strconv.Itoa(code),
	}).Inc()

	return m
}

func (m *mReader) AddReaderTopicProcessError(topic string) Reader {
	m.ReaderTopicProcessError.With(map[string]string{
		labelTopic: topic,
	}).Inc()

	return m
}

func (m *mReader) AddReaderTopicReadEvents(topic string, num int) Reader {
	m.ReaderTopicReadEvents.With(map[string]string{
		labelTopic: topic,
	}).Add(float64(num))

	return m
}

func (m *mReader) AddReaderTopicCommitEvents(topic string, num int) Reader {
	m.ReaderTopicCommitEvents.With(map[string]string{
		labelTopic: topic,
	}).Add(float64(num))

	return m
}

func (m *mReader) AddReaderTopicDecodeEvents(topic string, num int) Reader {
	m.ReaderTopicDecodeEvents.With(map[string]string{
		labelTopic: topic,
	}).Add(float64(num))

	return m
}

func (m *mReader) AddReaderTopicSkippedEvents(topic string, num int) Reader {
	m.ReaderTopicSkippedEvents.With(map[string]string{
		labelTopic: topic,
	}).Add(float64(num))

	return m
}

func (m *mReader) AddReaderTopicErrorEvents(topic string, num int) Reader {
	m.ReaderTopicErrorEvents.With(map[string]string{
		labelTopic: topic,
	}).Add(float64(num))

	return m
}

func (m *mReader) AddReaderTopicHandlingTime(topic string, duration time.Duration) Reader {
	m.ReaderHandlingTime.With(map[string]string{
		labelTopic: topic,
	}).Set(duration.Seconds())

	return m
}

// new writer
func (m *mReader) AddGarbageRecords(num int) Reader {
	m.garbageRecords.Add(float64(num))

	return m
}
