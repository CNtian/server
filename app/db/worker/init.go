package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"runtime"
	"sync"
	"vvService/appDB/protoDefine"
	"vvService/commonPackge/mateProto"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
)

type GameToConsumables map[int32]int32 // key:游戏ID  value:消耗房卡
type DailyTotal struct {
	GameRoundCount       int32
	Consumables          int32
	GameToConsumablesMap GameToConsumables
	PlayerSet            map[int64]struct{} // key:玩家ID
}

type activityData struct {
	AcID    primitive.ObjectID
	EndTime int64
}

type clubPercentValue struct {
	ClubPlay  float64
	ClubBaoDi int32
}

type playerTempTotal struct {
	jlItem    []dbCollectionDefine.ClubJiangliItem
	GameTotal []dbCollectionDefine.GameItem
}

var (
	waitgroup sync.WaitGroup

	clubPlayPercentMap = make(map[string]*clubPercentValue) // key:盟主俱乐部ID_玩法ID_下级俱乐部ID  value:百分比

	// 日活统计
	dailyPlayerMap = make(map[int32]*DailyTotal) // key:盟主俱乐部ID
	DailyTotalDay  int

	_activityMap = make(map[int32]activityData) // key:圈子ID value:结束时间

	_recPlayerToClubJL = make(map[string]*playerTempTotal) // key:20220801_clubID_uid  value:[]ClubJiangliItem
)

func HandleMQData(data *[]byte) {
	msg := mateProto.MessageMaTe{}
	err := json.Unmarshal(*data, &msg)
	if err != nil {
		glog.Warning("proto.Unmarshal", err.Error(), ",data:=", string(*data))
		return
	}
	if msg.MessageID != 333 {
		glog.Info("HandleMQData() msgID:=", msg.MessageID)
	}

	waitgroup.Add(1)
	go working(data, &msg)
	waitgroup.Wait()
}

func working(data *[]byte, msg *mateProto.MessageMaTe) {

	defer func() {
		if err := recover(); err != nil {
			buf := new(bytes.Buffer)
			fmt.Fprintf(buf, "%v\n", err)
			for i := 1; ; i++ {
				pc, file, line, ok := runtime.Caller(i)
				if !ok {
					break
				}
				fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
			}
			glog.Warning("bug:=", buf.String(), ",data:=", string(*data))
		}
		waitgroup.Done()
	}()

	switch msg.MessageID {
	case protoDefine.ID_GameOver:
		onGameOver(msg)
	case protoDefine.ID_RoundOver:
		onRoundOver(msg)
	case mateProto.ID_UpdateClubData:
		onUpdateClubData(msg)
	case protoDefine.ID_TotalMangeFee:
		//onTotalManageFee(msg)
	case protoDefine.ID_DeleteExpiredData:
		onDeleteExpiredData()
	case mateProto.ID_UpdatePlayerClubTime:
		onUpdatePlayerLastClubTime(msg)
	case mateProto.ID_DeleteClubTotal:
		onDeleteTotal(msg)
	case mateProto.ID_DeletePlayerUnusableScore:
		onDeletePlayerUnusableScore(msg)
	case mateProto.ID_UpdateClubPlayPercent:
		onUpdateClubPlayPercent(msg)
	case mateProto.ID_UpdateClubBaoDiPercent:
		onUpdateClubBaoDiPercent(msg)
	case mateProto.ID_NoticeClubActivity:
		onClubActivityLaunch(msg)
	case protoDefine.ID_WriteDaily:
		onWriteDaily()
	default:

	}
}

func onUpdateClubData(msg *mateProto.MessageMaTe) {
	param := mateProto.SS_UpdateClubData{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		glog.Warning("onUpdateClubData() err:=", err.Error(), ",data:=", string(msg.Data))
		return
	}
	if param.ClubID == 0 {
		clubDataMap = nil
		clubDataMap = make(map[int32]*collClub.DBClubData)
		return
	}

	delete(clubDataMap, param.ClubID)

	_, err = loadClubData(param.ClubID)
	if err != nil {
		glog.Warning("onUpdateClubData() err:=", err.Error(), ",clubID:=", param.ClubID)
		return
	}
}

func RecoverMengZhuDaily(item *dbCollectionDefine.DBDailyMengZHuPlayer) {
	m, ok := dailyPlayerMap[item.ClubID]
	if ok == false {
		m = &DailyTotal{}
		dailyPlayerMap[item.ClubID] = m
	}
	m.Consumables = item.Consumables
	m.GameRoundCount = item.GameRoundCount
}
