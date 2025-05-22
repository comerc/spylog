package spylog

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type SomeObject struct {
	log *slog.Logger
	val any
}

func NewSomeObject(val any) *SomeObject {
	return &SomeObject{
		log: slog.With("module", "module_name"),
		val: val,
	}
}

func (o *SomeObject) SomeMethod() {
	o.log.Info("test message from some method", "attr_key", "attr_val")
}

func TestSomeMethod(t *testing.T) {
	var o *SomeObject
	logHandler := GetModuleLogHandler("module_name", t.Name(), func() {
		o = NewSomeObject("val")
	})
	o.SomeMethod()

	slog.Info("test message from default") // not captured

	require.True(t, len(logHandler.Records) == 1)
	r0 := logHandler.Records[0]

	assert.Equal(t, "test message from some method", r0.Message)
	assert.Equal(t, "attr_val", GetAttrValue(r0, "attr_key"))
}
