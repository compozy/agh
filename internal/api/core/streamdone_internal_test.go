package core

import (
	"reflect"
	"testing"
)

func TestBaseHandlersSetStreamDoneShouldInstallFallbackChannelWithoutLogger(t *testing.T) {
	t.Parallel()

	var handlers BaseHandlers
	handlers.SetStreamDone(nil)

	streamDone := reflect.ValueOf(&handlers).Elem().FieldByName("streamDone")
	if !streamDone.IsValid() || streamDone.IsNil() {
		t.Fatal("streamDone = nil, want fallback channel installed")
	}
}
