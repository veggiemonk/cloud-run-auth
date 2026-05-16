package closeutil_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/veggiemonk/cloud-run-auth/internal/closeutil"
	"github.com/veggiemonk/cloud-run-auth/internal/is"
)

func TestDoNilFnErrorLeavesReturnUnchanged(t *testing.T) {
	var err error
	closeutil.Do(&err, func() error { return nil }, "close x")
	is.NoErr(t, err)
}

func TestDoCapturesWhenReturnIsNil(t *testing.T) {
	var err error
	boom := errors.New("boom")
	closeutil.Do(&err, func() error { return boom }, "close %s", "x")
	is.True(t, err != nil)
	is.True(t, errors.Is(err, boom))
	is.True(t, strings.Contains(err.Error(), "close x"))
}

func TestDoJoinsWhenReturnAlreadySet(t *testing.T) {
	orig := errors.New("orig")
	err := orig
	boom := errors.New("boom")
	closeutil.Do(&err, func() error { return boom }, "close x")
	is.True(t, errors.Is(err, orig))
	is.True(t, errors.Is(err, boom))
}
