package lib

import (
	"encoding/json"
	"github.com/AnotherProksY/Happ-TUI/core/structs"
	"log"
)

func CreateJsonNotification(msg string, obj any) []byte {
	data := structs.Message[any]{
		Msg:  msg,
		Data: obj,
	}
	json_data, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("failed to parse json trying to send a message %v %s", data, err)
	}
	return json_data
}

// CreateJsonReply makes the answer to a command that carried request_id.
func CreateJsonReply(msg string, request_id string, obj any) []byte {
	json_data, err := json.Marshal(structs.Reply{Msg: msg, RequestId: request_id, Data: obj})
	if err != nil {
		log.Fatalf("failed to parse json trying to send a reply %v %s", obj, err)
	}
	return json_data
}
