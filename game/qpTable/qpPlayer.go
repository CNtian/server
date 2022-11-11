package qpTable

import (
	"github.com/golang/glog"
	"qpGame/commonDefine/mateProto"
	"qpGame/wrapMQ"
)

type PlayerID int64

type QPPlayer struct {
	ID      PlayerID `json:"ID"`
	Head    string   `json:"Head"`
	Nick    string   `json:"Nick"`
	Sex     int32    `json:"Sex"`
	IP      string   `json:"-"`
	IsLeave bool
	//Power map[int32]int32

	LoginSrc    string `json:"-"`
	TableNumber int32  `json:"-"`
}

func NewPlayer(id PlayerID) *QPPlayer {
	return &QPPlayer{ID: id}
}

func (this *QPPlayer) SendData(msgID int32, msgBody interface{}) {
	if this.IsLeave == true {
		return
	}
	msg := mateProto.MessageMaTe{
		MessageID: msgID,
		SenderID:  int64(this.ID),
		To:        this.LoginSrc}

	err := wrapMQ.SendMsgTo(&msg, msgBody)
	if err != nil {
		glog.Warning("wrapMQ.PublishData error. ID:=", this.ID)
		return
	}

	// 日志
	//commonDef.LOG_Info("to playerID:=", this.ID, ",table number:=", this.TableNumber, ",to:=", this.LoginSrc, ",msgID:=", msgID)
}

func (this *QPPlayer) SendMsg(msg *mateProto.MessageMaTe) {
	msg.To = this.LoginSrc
	msg.SenderID = int64(this.ID)
	err := wrapMQ.SendMsg(msg)
	if err != nil {
		glog.Warning("wrapMQ.PublishData error. ID:=", this.ID)
		return
	}

	// 日志
	//commonDef.LOG_Info("to playerID:=", this.ID, ",table number:=", this.TableNumber, ",to:=", this.LoginSrc, ",msgID:=", msgID)
}
