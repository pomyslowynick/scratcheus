package lexer

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
	"unsafe"
)

type MetricType string

const (
	MetricTypeCounter        = MetricType("counter")
	MetricTypeGauge          = MetricType("gauge")
	MetricTypeHistogram      = MetricType("histogram")
	MetricTypeGaugeHistogram = MetricType("gaugehistogram")
	MetricTypeSummary        = MetricType("summary")
	MetricTypeInfo           = MetricType("info")
	MetricTypeStateset       = MetricType("stateset")
	MetricTypeUnknown        = MetricType("unknown")
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
	sExemplar
	sEValue
	sETimestamp
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
	tTimestamp
	tValue
)

type OpenMetricsLexer struct {
	b     []byte
	i     int
	start int
	err   error
	state int
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
	hasTS     bool
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
	EntryInvalid   Entry = -1
	EntryType      Entry = 0
	EntryHelp      Entry = 1
	EntrySeries    Entry = 2 // EntrySeries marks a series with a simple float64 as value.
	EntryComment   Entry = 3
	EntryUnit      Entry = 4
	EntryHistogram Entry = 5 // EntryHistogram marks a series with a native histogram as a value.
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
	case tHelp:
		return "HELP"
	case tType:
		return "TYPE"
	case tUnit:
		return "UNIT"
	case tEOFWord:
		return "EOFWORD"
	case tText:
		return "TEXT"
	case tComment:
		return "COMMENT"
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
	case tTimestamp:
		return "TIMESTAMP"
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
	case tHelp, tType, tUnit:
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
			case "gauge":
				p.mtype = MetricTypeGauge
			case "histogram":
				p.mtype = MetricTypeHistogram
			case "gaugehistogram":
				p.mtype = MetricTypeGaugeHistogram
			case "summary":
				p.mtype = MetricTypeSummary
			case "info":
				p.mtype = MetricTypeInfo
			case "stateset":
				p.mtype = MetricTypeStateset
			case "unknown":
				p.mtype = MetricTypeUnknown
			default:
				return EntryInvalid, fmt.Errorf("invalid metric type %q", s)
			}
		case tHelp:
			if !utf8.Valid(p.text) {
				return EntryInvalid, fmt.Errorf("help text %q is not a valid utf8 string", p.text)
			}
		}
		switch t {
		case tHelp:
			return EntryHelp, nil
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

func (p *OpenMetricsParser) parseComment() error {
	var err error

	// if p.ignoreExemplar {
	// 	for t := p.nextToken(); t != tLinebreak; t = p.nextToken() {
	// 		if t == tEOF {
	// 			return errors.New("data does not end with # EOF")
	// 		}
	// 	}
	// 	return nil
	// }

	// Parse the labels.
	p.eOffsets, err = p.parseLVals(p.eOffsets, true)
	if err != nil {
		return err
	}
	// p.exemplar = p.l.b[p.start:p.l.i]

	// Get the value.
	// p.exemplarVal, err = p.getFloatValue(p.nextToken(), "exemplar labels")
	if err != nil {
		return err
	}

	// Read the optional timestamp.
	// p.hasExemplarTs = false
	switch t2 := p.nextToken(); t2 {
	case tEOF:
		return errors.New("data does not end with # EOF")
	case tLinebreak:
		break
	// case tTimestamp:
	// 	p.hasExemplarTs = true
	// 	var ts float64
	// 	// A float is enough to hold what we need for millisecond resolution.
	// 	if ts, err = parseFloat(yoloString(p.l.buf()[1:])); err != nil {
	// 		return fmt.Errorf("%w while parsing: %q", err, p.l.b[p.start:p.l.i])
	// 	}
	// 	if math.IsNaN(ts) || math.IsInf(ts, 0) {
	// 		return fmt.Errorf("invalid exemplar timestamp %f", ts)
	// 	}
	// 	p.exemplarTs = int64(ts * 1000)
	// 	switch t3 := p.nextToken(); t3 {
	// 	case tLinebreak:
	// 	default:
	// 		return p.parseError("expected next entry after exemplar timestamp", t3)
	// }
	default:
		return p.parseError("expected timestamp or comment", t2)
	}
	return nil
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

	p.hasTS = false
	switch t2 := p.nextToken(); t2 {
	case tEOF:
		return errors.New("data does not end with # EOF")
	case tLinebreak:
		break
	case tComment:
		if err := p.parseComment(); err != nil {
			return err
		}
	case tTimestamp:
		p.hasTS = true
		var ts float64
		// A float is enough to hold what we need for millisecond resolution.
		if ts, err = parseFloat(yoloString(p.l.buf()[1:])); err != nil {
			return fmt.Errorf("%w while parsing: %q", err, p.l.b[p.start:p.l.i])
		}
		if math.IsNaN(ts) || math.IsInf(ts, 0) {
			return fmt.Errorf("invalid timestamp %f", ts)
		}
		p.ts = int64(ts * 1000)
		switch t3 := p.nextToken(); t3 {
		case tLinebreak:
		case tComment:
			if err := p.parseComment(); err != nil {
				return err
			}
		default:
			return p.parseError("expected next entry after timestamp", t3)
		}
	}

	return nil
}

// typeRequiresCT returns true if the metric type requires a _created timestamp.
func typeRequiresCT(t MetricType) bool {
	switch t {
	case MetricTypeCounter, MetricTypeSummary, MetricTypeHistogram:
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

func (p *OpenMetricsParser) Series() ([]byte, *int64, float64) {
	if p.hasTS {
		ts := p.ts
		return p.series, &ts, p.val
	}
	return p.series, nil, p.val
}
