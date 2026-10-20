package respond

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-playground/form/v4"
)

const defaultMaxBodyBytes = 1 << 20

var formDecoder = newFormDecoder()

func newFormDecoder() *form.Decoder {
	d := form.NewDecoder()
	// HTML checkboxes send "on" when ticked.
	d.RegisterCustomTypeFunc(func(vals []string) (any, error) {
		switch vals[0] {
		case "on":
			return true, nil
		case "":
			return false, nil
		}
		return strconv.ParseBool(vals[0])
	}, false)
	return d
}

type BodyOption func(*bodyConfig)

type bodyConfig struct {
	maxBytes int64
}

// MaxBytes caps the request body; larger bodies get 413. Default 1 MiB.
func MaxBytes(n int64) BodyOption {
	return func(c *bodyConfig) { c.maxBytes = n }
}

func newBodyConfig(opts []BodyOption) bodyConfig {
	cfg := bodyConfig{maxBytes: defaultMaxBodyBytes}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

type Validator interface {
	Validate() map[string]string
}

func (res Response) ParseJSON(dst any, opts ...BodyOption) bool {
	cfg := newBodyConfig(opts)
	dec := json.NewDecoder(http.MaxBytesReader(res.w, res.r.Body, cfg.maxBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		res.BadRequest()
		return false
	}
	if dec.More() {
		res.BadRequest(WithDetail("body must contain a single JSON value"))
		return false
	}
	return res.validate(dst)
}

func (res Response) ParseForm(dst any, opts ...BodyOption) bool {
	cfg := newBodyConfig(opts)
	res.r.Body = http.MaxBytesReader(res.w, res.r.Body, cfg.maxBytes)
	// ParseMultipartForm drops ParseForm's error for urlencoded bodies, so parse that first.
	err := res.r.ParseForm()
	if err == nil {
		if err = res.r.ParseMultipartForm(cfg.maxBytes); errors.Is(err, http.ErrNotMultipart) {
			err = nil
		}
	}
	if err != nil {
		res.BadRequest()
		return false
	}
	if err := formDecoder.Decode(dst, res.r.PostForm); err != nil {
		res.BadRequest()
		return false
	}
	return res.validate(dst)
}

func (res Response) validate(dst any) bool {
	v, ok := dst.(Validator)
	if !ok {
		return true
	}
	if errs := v.Validate(); len(errs) > 0 {
		res.ValidationError(WithErrors(errs))
		return false
	}
	return true
}
