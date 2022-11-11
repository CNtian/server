package worker

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/mongo"
	"strconv"
	"time"
	"vvService/appDB/db"
	"vvService/appDB/wrapMQ"
	"vvService/commonPackge/mateProto"
)

func onUpdatePlayerLastClubTime(msg *mateProto.MessageMaTe) {
	updatePlayerLastClubTime := mateProto.SS_UpdatePlayerClubTime{}
	json.Unmarshal(msg.Data, &updatePlayerLastClubTime)

	db.UpdatePlayerLastClubTime(updatePlayerLastClubTime.ClubID, updatePlayerLastClubTime.UID)
}

func onDeleteExpiredData() {

	db.DeleteExpiredData()
	clubPlayPercentMap = make(map[string]*clubPercentValue)
	_recPlayerToClubJL = make(map[string]*playerTempTotal)
}

func onDeletePlayerUnusableScore(msg *mateProto.MessageMaTe) {
	msgBody := mateProto.SS_DeletePlayerUnusableScore{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		glog.Warning("onDeletePlayerUnusableScore() err:=", err.Error())
		return
	}

	temp := unusableInfo{ClubID: msgBody.ClubID}

	player := map[int64]*unusableInfo{}
	player[msgBody.PlayerID] = &temp

	if msgBody.CurScore >= 0 {
		temp.Score, err = db.DeleteClubPlayerUnusableScore(msgBody.PlayerID, msgBody.ClubID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return
			}
			glog.Warning("DeleteClubPlayerUnusableScore() err:=", err.Error(), ", uid:=", msgBody.PlayerID, ",clubID:=", msgBody.ClubID)
			return
		}
		temp.Score *= -1
		updateClubUnusableScore(player)
	} else {
		temp.Score = msgBody.CurScore - msgBody.BeforeScore

		putClubPlayerUnusableScore(player)
		updateClubUnusableScore(player)
	}
}

func onUpdateClubPlayPercent(msg *mateProto.MessageMaTe) {
	msgBody := mateProto.SS_UpdateClubPlayPercent{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		glog.Warning("onUpdateClubPlayPercent() err:=", err.Error())
		return
	}

	key := fmt.Sprintf("%d_%d_%d", msgBody.MZClubID, msgBody.ClubPlayID, msgBody.ClubID)
	v, ok := clubPlayPercentMap[key]
	if ok == false {
		clubPlayV, clubBaoDiV := float64(0), int32(0)
		GetClubPlayPercent(msgBody.MZClubID, msgBody.ClubID, msgBody.ClubPlayID, &clubPlayV, &clubBaoDiV)
		v = &clubPercentValue{ClubPlay: clubPlayV, ClubBaoDi: clubBaoDiV}
		clubPlayPercentMap[key] = v
	}
	v.ClubPlay = msgBody.Percent
}

func onUpdateClubBaoDiPercent(msg *mateProto.MessageMaTe) {
	msgBody := mateProto.SS_UpdateClubBaoDiPercent{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		glog.Warning("onUpdateClubBaoDiPercent() err:=", err.Error())
		return
	}

	key := fmt.Sprintf("%d_%d_%d", msgBody.MZClubID, msgBody.ClubPlayID, msgBody.ClubID)
	v, ok := clubPlayPercentMap[key]
	if ok == false {
		clubPlayV, clubBaoDiV := float64(0), int32(0)
		GetClubPlayPercent(msgBody.MZClubID, msgBody.ClubID, msgBody.ClubPlayID, &clubPlayV, &clubBaoDiV)
		v = &clubPercentValue{ClubPlay: clubPlayV, ClubBaoDi: clubBaoDiV}
		clubPlayPercentMap[key] = v
	}
	v.ClubBaoDi = msgBody.Percent
}

func onClubActivityLaunch(msg *mateProto.MessageMaTe) {
	msgBody := mateProto.SS_NoticeClubActivity{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		glog.Warning("onUpdateClubPlayPercent() err:=", err.Error())
		return
	}

	glog.Warning("activity. clubID:=", msgBody.ClubID, ", isOpen :=", msgBody.IsOpen)

	if msgBody.IsOpen == true {
		_activityMap[msgBody.ClubID] = activityData{AcID: msgBody.AcID, EndTime: msgBody.EndTime}
	} else {
		delete(_activityMap, msgBody.ClubID)
	}
}

func onWriteDaily() {
	year, month, day := time.Now().Date()
	date, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", year, month, day))

	WriteMengZhuDaily(date)
	dailyPlayerMap = make(map[int32]*DailyTotal)

	noticeMsg := mateProto.MessageMaTe{MessageID: mateProto.ID_NoticDailyChanged}
	noticeMsg.Data, _ = json.Marshal(&mateProto.SS_NoticDailyChanged{MZClubID: 0})
	err := wrapMQ.PublishProto("club", &noticeMsg)
	if err != nil {
		glog.Warning("club",
			" ,wrapMQ.PublishData err. err:=", err.Error(),
			",msg ID:=", noticeMsg.MessageID, " ,data:=", string(noticeMsg.Data))
	}
}
