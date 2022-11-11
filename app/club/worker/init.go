package worker

import (
	"encoding/json"
	"github.com/golang/glog"
	clubProto "vvService/appClub/protoDefine"
	"vvService/appClub/worker/clubEvent"
	"vvService/appClub/worker/tableSnapshot"
	commonDef "vvService/commonPackge"
	"vvService/commonPackge/mateProto"
)

func HandleMQData(data []byte) {
	msg := mateProto.MessageMaTe{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		glog.Warning("proto.Unmarshal", err.Error())
		return
	}

	switch msg.MessageID {
	case mateProto.ID_BroadGameServiceStatus, clubProto.ID_GetClubPlayList:
	case clubProto.ID_TableGet, clubProto.ID_PerSeconGetTables, mateProto.ID_PlayerOnline:
	default:
		commonDef.Info.Info("uid:=", msg.SenderID, ",msgID:=", msg.MessageID, ",source:=", msg.Source, ",data:=", string(msg.Data))
	}

	if msg.MessageID >= 400 {
		tableSnapshot.SelfPostEvent.PostMaTeEvent(&msg)
	} else {
		clubEvent.SelfPostEvent.PostMaTeEvent(&msg)
	}
}
