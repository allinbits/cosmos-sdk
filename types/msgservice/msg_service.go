package msgservice

import (
	"strings"
)

// IsServiceMsg checks if a type URL corresponds to a service method name,
// i.e. /cosmos.bank.Msg/Send vs /cosmos.bank.MsgSend
func IsServiceMsg(typeURL string) bool {
	return strings.Count(typeURL, "/") >= 2
}
