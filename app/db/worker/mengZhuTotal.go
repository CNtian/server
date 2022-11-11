package worker

import (
	"encoding/json"
	"github.com/golang/glog"
	"vvService/appDB/db"
	"vvService/appDB/protoDefine"
	"vvService/appDB/wrapMQ"
	"vvService/commonPackge/mateProto"
	"vvService/dbCollectionDefine"
)

// 日活跃统计
func PutNewRec(gr *protoDefine.SS_GameOverRecord) {

	m, ok := dailyPlayerMap[gr.MZClubID]
	if ok == false {
		m = &DailyTotal{GameToConsumablesMap: make(GameToConsumables), PlayerSet: make(map[int64]struct{})}
		dailyPlayerMap[gr.MZClubID] = m
	}
	m.Consumables += gr.ConsumeCount
	m.GameRoundCount += 1

	g, ok := m.GameToConsumablesMap[gr.GameID]
	if ok == false {
		m.GameToConsumablesMap[gr.GameID] = gr.ConsumeCount
	} else {
		m.GameToConsumablesMap[gr.GameID] = g + gr.ConsumeCount
	}

	for _, v := range gr.PlayerScore {
		m.PlayerSet[v.UID] = struct{}{}
	}

	gameConsumablesArr := make([]dbCollectionDefine.GameToConsumables, 0, len(m.GameToConsumablesMap))
	for k, v := range m.GameToConsumablesMap {
		gameConsumablesArr = append(gameConsumablesArr, dbCollectionDefine.GameToConsumables{GameID: k, Consumables: v})
	}

	noticeMsg := mateProto.MessageMaTe{MessageID: mateProto.ID_NoticDailyChanged}
	noticeMsg.Data, _ = json.Marshal(&mateProto.SS_NoticDailyChanged{
		MZClubID:        gr.MZClubID,
		Consumables:     m.Consumables,
		GameRoundCount:  m.GameRoundCount,
		GameCategoryArr: gameConsumablesArr,
		DailyPlayers:    len(m.PlayerSet)})
	err := wrapMQ.PublishProto("club", &noticeMsg)
	if err != nil {
		glog.Warning("club",
			" ,wrapMQ.PublishData err. err:=", err.Error(),
			",msg ID:=", noticeMsg.MessageID, " ,data:=", string(noticeMsg.Data))
	}
}

func WriteMengZhuDaily(date int) {

	for k, v := range dailyPlayerMap {
		t := dbCollectionDefine.DBDailyMengZHuPlayer{Date: date, ClubID: k}
		t.GameToConsumablesArr = make([]dbCollectionDefine.GameToConsumables, 0, len(v.GameToConsumablesMap))
		t.DailyPlayers = len(v.PlayerSet)
		allPlayers, err := db.GetMengZhuAllPlayers(k)
		if err != nil {
			glog.Warning("db.GetMengZhuAllPlayers err. ", k, ",", err.Error())
		}
		for kg, vg := range v.GameToConsumablesMap {
			t.GameToConsumablesArr = append(t.GameToConsumablesArr,
				dbCollectionDefine.GameToConsumables{GameID: kg, Consumables: vg})
		}
		t.Players = allPlayers
		t.GameRoundCount = v.GameRoundCount
		t.Consumables = v.Consumables

		err = db.PutNewDay(&t)
		if err != nil {
			glog.Warning("db.PutNewDay err. ", date, ",", err.Error())
		}
	}
}
