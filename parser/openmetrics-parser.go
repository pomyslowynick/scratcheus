package parser

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
	"unsafe"

	"github.com/pomyslowynick/scratcheus/labels"
)

type MetricType string

const (
	MetricTypeCounter = MetricType("counter")
	MetricTypeUnknown = MetricType("unknown")
)

type token int

const (
	sInit = iota
	sComment
	sMeta1
	sMeta2
	sLabels
	sLValue
	sValue
	sTimestamp
)

const (
	tInvalid   token = -1
	tEOF       token = 0
	tLinebreak token = iota
	tWhitespace
	tHelp
	tType
	tUnit
	tEOFWord
	tText
	tComment
	tBlank
	tMName
	tQString
	tBraceOpen
	tBraceClose
	tLName
	tLValue
	tComma
	tEqual
	tValue
	tTimestamp
)

type OpenMetricsLexer struct {
	b     []byte
	i     int
	start int
	err   error
	state int
}

type ParsedSample struct {
	Labels labels.Labels
	Value  float64
}

func New(data []byte) OpenMetricsLexer {
	l := OpenMetricsLexer{b: data}
	return l
}

// buf returns the buffer of the current token.
func (l *OpenMetricsLexer) buf() []byte {
	return l.b[l.start:l.i]
}

// next advances the openMetricsLexer to the next character.
func (l *OpenMetricsLexer) next() byte {
	l.i++
	if l.i >= len(l.b) {
		l.err = io.EOF
		return byte(tEOF)
	}
	// Lex struggles with null bytes. If we are in a label value or help string, where
	// they are allowed, consume them here immediately.
	for l.b[l.i] == 0 && (l.state == sLValue || l.state == sMeta2 || l.state == sComment) {
		l.i++
		if l.i >= len(l.b) {
			l.err = io.EOF
			return byte(tEOF)
		}
	}
	return l.b[l.i]
}

func (l *OpenMetricsLexer) Error(es string) {
	l.err = errors.New(es)
}

type OpenMetricsParser struct {
	l         *OpenMetricsLexer
	series    []byte
	mfNameLen int // length of metric family name to get from series.
	text      []byte
	mtype     MetricType
	val       float64
	ts        int64
	start     int
	// offsets is a list of offsets into series that describe the positions
	// of the metric name and label names and values for this series.
	// p.offsets[0] is the start character of the metric name.
	// p.offsets[1] is the end of the metric name.
	// Subsequently, p.offsets is a pair of pair of offsets for the positions
	// of the label name and value start and end characters.
	offsets []int

	eOffsets []int

	// Created timestamp parsing state.
	ct        int64
	ctHashSet uint64
	// visitedMFName is the metric family name of the last visited metric when peeking ahead
	// for _created series during the execution of the CreatedTimestamp method.
	visitedMFName []byte
}

// Entry represents the type of a parsed entry.
type Entry int

const (
	EntryInvalid Entry = -1
	EntryType    Entry = 0
	EntrySeries  Entry = 1 // EntrySeries marks a series with a simple float64 as value.
	EntryUnit    Entry = 2
)

func NewParser(b []byte) *OpenMetricsParser {
	parser := &OpenMetricsParser{
		l: &OpenMetricsLexer{b: b},
	}

	return parser
}

// nextToken returns the next token from the openMetricsLexer.
func (p *OpenMetricsParser) nextToken() token {
	tok := p.l.Lex()
	return tok
}

func (p *OpenMetricsParser) parseError(exp string, got token) error {
	e := p.l.i + 1
	if len(p.l.b) < e {
		e = len(p.l.b)
	}
	return fmt.Errorf("%s, got %q (%q) while parsing: %q", exp, p.l.b[p.l.start:e], got, p.l.b[p.start:e])
}

func (t token) String() string {
	switch t {
	case tInvalid:
		return "INVALID"
	case tEOF:
		return "EOF"
	case tLinebreak:
		return "LINEBREAK"
	case tWhitespace:
		return "WHITESPACE"
	case tType:
		return "TYPE"
	case tUnit:
		return "UNIT"
	case tEOFWord:
		return "EOFWORD"
	case tText:
		return "TEXT"
	case tBlank:
		return "BLANK"
	case tMName:
		return "MNAME"
	case tQString:
		return "QSTRING"
	case tBraceOpen:
		return "BOPEN"
	case tBraceClose:
		return "BCLOSE"
	case tLName:
		return "LNAME"
	case tLValue:
		return "LVALUE"
	case tEqual:
		return "EQUAL"
	case tComma:
		return "COMMA"
	case tValue:
		return "VALUE"
	}
	return fmt.Sprintf("<invalid: %d>", t)
}

func (p *OpenMetricsParser) Next() (Entry, error) {
	var err error

	p.start = p.l.i
	p.offsets = p.offsets[:0]

	switch t := p.nextToken(); t {
	case tEOFWord:
		if t := p.nextToken(); t != tEOF {
			return EntryInvalid, errors.New("unexpected data after # EOF")
		}
		return EntryInvalid, io.EOF
	case tEOF:
		return EntryInvalid, errors.New("data does not end with # EOF")
	case tType, tUnit:
		switch t2 := p.nextToken(); t2 {
		case tMName:
			mStart := p.l.start
			mEnd := p.l.i
			if p.l.b[mStart] == '"' && p.l.b[mEnd-1] == '"' {
				mStart++
				mEnd--
			}
			p.mfNameLen = mEnd - mStart
			p.offsets = append(p.offsets, mStart, mEnd)
		default:
			return EntryInvalid, p.parseError("expected metric name after "+t.String(), t2)
		}
		switch t2 := p.nextToken(); t2 {
		case tText:
			if len(p.l.buf()) > 1 {
				p.text = p.l.buf()[1 : len(p.l.buf())-1]
			} else {
				p.text = []byte{}
			}
		default:
			return EntryInvalid, fmt.Errorf("expected text in %s", t.String())
		}
		switch t {
		case tType:
			switch s := yoloString(p.text); s {
			case "counter":
				p.mtype = MetricTypeCounter
			case "unknown":
				p.mtype = MetricTypeUnknown
			default:
				return EntryInvalid, fmt.Errorf("invalid metric type %q", s)
			}
		}
		switch t {
		case tType:
			return EntryType, nil
		case tUnit:
			m := yoloString(p.l.b[p.offsets[0]:p.offsets[1]])
			u := yoloString(p.text)
			if len(u) > 0 {
				if !strings.HasSuffix(m, u) || len(m) < len(u)+1 || p.l.b[p.offsets[1]-len(u)-1] != '_' {
					return EntryInvalid, fmt.Errorf("unit %q not a suffix of metric %q", u, m)
				}
			}
			return EntryUnit, nil
		}

	case tBraceOpen:
		// We found a brace, so make room for the eventual metric name. If these
		// values aren't updated, then the metric name was not set inside the
		// braces and we can return an error.
		if len(p.offsets) == 0 {
			p.offsets = []int{-1, -1}
		}
		if p.offsets, err = p.parseLVals(p.offsets, false); err != nil {
			return EntryInvalid, err
		}

		p.series = p.l.b[p.start:p.l.i]
		if err := p.parseSeriesEndOfLine(p.nextToken()); err != nil {
			return EntryInvalid, err
		}
		// if p.skipCTSeries && p.isCreatedSeries() {
		// 	return p.Next()
		// }
		return EntrySeries, nil
	case tMName:
		p.offsets = append(p.offsets, p.start, p.l.i)
		p.series = p.l.b[p.start:p.l.i]

		t2 := p.nextToken()
		if t2 == tBraceOpen {
			p.offsets, err = p.parseLVals(p.offsets, false)
			if err != nil {
				return EntryInvalid, err
			}
			p.series = p.l.b[p.start:p.l.i]
			t2 = p.nextToken()
		}

		if err := p.parseSeriesEndOfLine(t2); err != nil {
			return EntryInvalid, err
		}
		// if p.skipCTSeries && p.isCreatedSeries() {
		// 	return p.Next()
		// }
		return EntrySeries, nil
	default:
		err = p.parseError("expected a valid start token", t)
	}
	return EntryInvalid, err
}

func yoloString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func (p *OpenMetricsParser) parseLVals(offsets []int, isExemplar bool) ([]int, error) {
	t := p.nextToken()
	for {
		curTStart := p.l.start
		curTI := p.l.i
		switch t {
		case tBraceClose:
			return offsets, nil
		case tLName:
		case tQString:
		default:
			return nil, p.parseError("expected label name", t)
		}

		t = p.nextToken()
		// A quoted string followed by a comma or brace is a metric name. Set the
		// offsets and continue processing. If this is an exemplar, this format
		// is not allowed.
		if t == tComma || t == tBraceClose {
			if isExemplar {
				return nil, p.parseError("expected label name", t)
			}
			if offsets[0] != -1 || offsets[1] != -1 {
				return nil, fmt.Errorf("metric name already set while parsing: %q", p.l.b[p.start:p.l.i])
			}
			offsets[0] = curTStart + 1
			offsets[1] = curTI - 1
			if t == tBraceClose {
				return offsets, nil
			}
			t = p.nextToken()
			continue
		}
		// We have a label name, and it might be quoted.
		if p.l.b[curTStart] == '"' {
			curTStart++
			curTI--
		}
		offsets = append(offsets, curTStart, curTI)

		if t != tEqual {
			return nil, p.parseError("expected equal", t)
		}
		if t := p.nextToken(); t != tLValue {
			return nil, p.parseError("expected label value", t)
		}
		if !utf8.Valid(p.l.buf()) {
			return nil, fmt.Errorf("invalid UTF-8 label value: %q", p.l.buf())
		}

		// The openMetricsLexer ensures the value string is quoted. Strip first
		// and last character.
		offsets = append(offsets, p.l.start+1, p.l.i-1)

		// Free trailing commas are allowed.
		t = p.nextToken()
		if t == tComma {
			t = p.nextToken()
		} else if t != tBraceClose {
			return nil, p.parseError("expected comma or brace close", t)
		}
	}
}

// isCreatedSeries returns true if the current series is a _created series.
func (p *OpenMetricsParser) isCreatedSeries() bool {
	metricName := p.series[p.offsets[0]-p.start : p.offsets[1]-p.start]
	// check length so the metric is longer than len("_created")
	if typeRequiresCT(p.mtype) && len(metricName) >= 8 && string(metricName[len(metricName)-8:]) == "_created" {
		return true
	}
	return false
}

// parseSeriesEndOfLine parses the series end of the line (value, optional
// timestamp, commentary, etc.) after the metric name and labels.
// It starts parsing with the provided token.
func (p *OpenMetricsParser) parseSeriesEndOfLine(t token) error {
	if p.offsets[0] == -1 {
		return fmt.Errorf("metric name not set while parsing: %q", p.l.b[p.start:p.l.i])
	}

	var err error
	p.val, err = p.getFloatValue(t, "metric")
	if err != nil {
		return err
	}

	switch t2 := p.nextToken(); t2 {
	case tEOF:
		return errors.New("data does not end with # EOF")
	case tLinebreak:
		break
	}

	return nil
}

// typeRequiresCT returns true if the metric type requires a _created timestamp.
func typeRequiresCT(t MetricType) bool {
	switch t {
	case MetricTypeCounter:
		return true
	default:
		return false
	}
}

func (p *OpenMetricsParser) getFloatValue(t token, after string) (float64, error) {
	if t != tValue {
		return 0, p.parseError(fmt.Sprintf("expected value after %v", after), t)
	}
	val, err := parseFloat(yoloString(p.l.buf()[1:]))
	if err != nil {
		return 0, fmt.Errorf("%w while parsing: %q", err, p.l.b[p.start:p.l.i])
	}
	return val, nil
}

func parseFloat(s string) (float64, error) {
	// Keep to pre-Go 1.13 float formats.
	if strings.ContainsAny(s, "pP_") {
		return 0, errors.New("unsupported character in float")
	}
	return strconv.ParseFloat(s, 64)
}

func (p *OpenMetricsParser) Series() ([]byte, float64) {
	return p.series, p.val
}

func (p *OpenMetricsParser) labels() (string, labels.Labels) {

	labelsCount := len(p.offsets) / 2

	allLabels := make(labels.Labels, 0, labelsCount)

	metricName := string(p.l.b[p.offsets[0]:p.offsets[1]])
	allLabels = append(allLabels, labels.Label{
		Name:  "__name__",
		Value: metricName,
	})

	if len(p.offsets) > 2 {
		for i := 2; len(p.offsets)-2 > i; i += 4 {
			labelName := string(p.l.b[p.offsets[i]:p.offsets[i+1]])
			labelValue := string(p.l.b[p.offsets[i+2]:p.offsets[i+3]])
			allLabels = append(allLabels, labels.Label{
				Name:  labelName,
				Value: labelValue,
			})

		}
	}
	return metricName, allLabels
}

func ParseScrapeData(scrapeData []byte) map[string]ParsedSample {
	newParser := NewParser(scrapeData)

	seriesSamples := make(map[string]ParsedSample)
	for {
		var (
			et  Entry
			err error
		)

		if et, err = newParser.Next(); err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			}
			break
		}

		if et == EntrySeries {
			_, value := newParser.Series()
			metricName, labels := newParser.labels()
			seriesSamples[metricName] = ParsedSample{
				Value:  value,
				Labels: labels,
			}
		}
	}

	return seriesSamples
}
