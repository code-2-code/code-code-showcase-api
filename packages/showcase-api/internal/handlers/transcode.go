package handlers

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// transcodeMessage converts between proto messages with overlapping field
// numbers by round-tripping through proto JSON. This is the same pattern used
// by console-api to transcode between provider-service and management types.
func transcodeMessage(src proto.Message, dst proto.Message) error {
	if src == nil || dst == nil {
		return nil
	}
	body, err := protojson.MarshalOptions{EmitUnpopulated: false}.Marshal(src)
	if err != nil {
		return fmt.Errorf("showcase-api/handlers: marshal message: %w", err)
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, dst); err != nil {
		return fmt.Errorf("showcase-api/handlers: unmarshal message: %w", err)
	}
	return nil
}
