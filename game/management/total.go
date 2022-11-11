package management

import (
	"qpGame/commonDefine/mateProto"
	"qpGame/commonDefine/mateProto/protoGameBasic"
	"qpGame/wrapMQ"
)

func NoticeHallNewTable(gameID, tableID int32, playerID int64) {
	mtMsg := mateProto.MessageMaTe{
		SenderID:  playerID,
		To:        "hall",
		MessageID: protoGameBasic.ID_AddNewTable}

	body := protoGameBasic.SS_AddNewTable{
		GameID: gameID, TableID: tableID,
	}

	wrapMQ.SendMsgTo(&mtMsg, &body)
}

func NoticeHallDeleteTable(tableID int32, seatArr []int64) {
	mtMsg := mateProto.MessageMaTe{
		To:        "hall",
		MessageID: protoGameBasic.ID_DeleteTable}

	body := protoGameBasic.SS_DeleteTable{TableID: tableID, Players: seatArr}

	wrapMQ.SendMsgTo(&mtMsg, &body)
}

func NoticeHallPlayerJoinTable(tableID int32, playerID int64) {
	mtMsg := mateProto.MessageMaTe{
		SenderID:  playerID,
		To:        "hall",
		MessageID: protoGameBasic.ID_PlayerJoinTable}

	body := protoGameBasic.SS_PlayerJoinTable{TableID: tableID}

	wrapMQ.SendMsgTo(&mtMsg, &body)
}

func NoticeHallPlayerLeaveTable(tableID int32, playerID int64) {
	mtMsg := mateProto.MessageMaTe{
		SenderID:  playerID,
		To:        "hall",
		MessageID: protoGameBasic.ID_PlayerLeaveTable}

	body := protoGameBasic.SS_PlayerLeaveTable{TableID: tableID}

	wrapMQ.SendMsgTo(&mtMsg, &body)
}
