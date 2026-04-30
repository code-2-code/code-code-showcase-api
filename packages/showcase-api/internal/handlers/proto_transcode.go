package handlers

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func transcodeMessage(src proto.Message, dst proto.Message) error {
	if src == nil || dst == nil {
		return fmt.Errorf("showcase/handlers: transcode source and destination are required")
	}
	payload, err := protojson.Marshal(src)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(payload, dst)
}
