package mateProto

type MessageMaTe struct {
	Source   string `json:"src,omitempty"`
	SenderID int64  `json:"id,omitempty"`

	To        string      `json:"to"`
	MessageID int32       `json:"msgID"`
	Data      []byte      `json:"data"`
	MsgBody   interface{} `json:"-"`
	MZID string `json:"mzid,omitempty"`
}
