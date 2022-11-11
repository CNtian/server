package clubEvent

import (
	"encoding/json"
	"github.com/go-redis/redis/v7"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"sync"
	"time"
	"vvService/appClub/clubCommon"
	"vvService/appClub/db"
	clubProto "vvService/appClub/protoDefine"
	"vvService/appClub/wrapMQ"
	commonDef "vvService/commonPackge"
	"vvService/commonPackge/errorCodeDef"
	"vvService/commonPackge/mateProto"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
)

var (
	clubEventChan      = make(chan *mateProto.MessageMaTe, 1024)
	SelfPostEvent      SelfPostEvents
	tableSnapshotEvent clubCommon.PostEvent
	virtualTableEvent  clubCommon.PostEvent

	_activityWaitOpen sync.Map // key:activityID value:
	_activityIng      sync.Map // key:activityID value:

	_clubAllMember sync.Map // key:clubID value:[]*DBClubMember

	_curTableCount int64
)

type SelfPostEvents struct {
}

func (this *SelfPostEvents) PostMaTeEvent(msg *mateProto.MessageMaTe) {
	clubEventChan <- msg
}

func SetTableSnapshotEvent(e clubCommon.PostEvent) {
	tableSnapshotEvent = e
}

func SetVirtualTableEvent(e clubCommon.PostEvent) {
	virtualTableEvent = e
}

func handleClubReadEvent(msg *mateProto.MessageMaTe) bool {

	var rsp *mateProto.JsonResponse

	nowT1 := time.Now()

	defer func() {
		nowT2 := time.Now()
		dur := nowT2.Sub(nowT1)
		if dur.Milliseconds() >= 50 {
			glog.Warning("msg handle time too long. expend ", dur.Milliseconds(), " ms."+
				" msg:=", msg.MessageID, ",sender ID:=", msg.SenderID, ",data:=", len(msg.Data))
		}
	}()

	switch msg.MessageID {
	// 俱乐部 成员
	case clubProto.ID_TableGet, clubProto.ID_PerSeconGetTables:
		rsp = onGetClubTables(msg)
	case clubProto.ID_GetMutexPlayer:
		rsp = onGetMutexGroup(msg)
	case clubProto.ID_GetMemberJudgeLog:
		rsp = onGetMemberJudgeLog(msg)
	case clubProto.ID_GetActivity:
		rsp = onGetClubActivity(msg)

	// 俱乐部
	case clubProto.ID_GetClubData:
		rsp = onGetClubData(msg)
	case clubProto.ID_GetClubMail:
		rsp = onGetClubMail(msg)
	case clubProto.ID_GetClubPlayInfo:
		rsp = onGetClubPlayInfo(msg)
	case clubProto.ID_GetClubPlayList:
		rsp = onGetClubPlay(msg)
	case clubProto.ID_GetGameList:
		rsp = onGetClubGameIDList(msg)
	case clubProto.ID_GetClubIntro:
		rsp = onGetClubIntro(msg)
	case clubProto.ID_ForceDissolveTable:
		rsp = onForceDissolveTable(msg)
	case clubProto.ID_GetClubPlayPercent:
		rsp = onGetClubPlayPercent(msg)
	case clubProto.ID_GetClubMemberRemark:
		rsp = onGetClubMemberRemark(msg)

	case clubProto.ID_GetActivityAward:
		rsp = onReceiveActivityAward(msg)
	case clubProto.ID_GetActivityAwardList:
		rsp = onGetActivityAwardList(msg)
	case clubProto.ID_GetVirtualTableConfigItem:
		rsp = onGetVirtualTableConfig(msg)

	// 俱乐部 查询\统计
	case clubProto.ID_GetClubMember:
		rsp = onGetClubMember(msg)
	case clubProto.ID_GetClubList:
		rsp = onGetClubList(msg)
	case clubProto.ID_GetClubScoreLog:
		rsp = onGetClubScoreLog(msg)
	case clubProto.ID_GetClubOperationLog:
		rsp = onGetClubOperationLog(msg)
	case clubProto.ID_GetClubGameRecord:
		rsp = onGetClubGameRecord(msg)
	case clubProto.ID_GetClubPlayerTotal:
		rsp = onGetClubPlayerTotal(msg)
	case clubProto.ID_GetClubTotal:
		rsp = onGetClubTotal(msg)
	case clubProto.ID_QueryPlayerLeague:
		rsp = onGetPlayerInLeague(msg)
	case clubProto.ID_GetTwoPlayerTogetherData:
		rsp = getTwoPlayerTogetherData(msg)
	case clubProto.ID_QueryTotal:
		rsp = onQueryTotal(msg)

	case mateProto.ID_UpdateClubRobotConfig:
		rsp = onUpdateRobotCfg(msg)
	case mateProto.ID_UpdateRobotItemCfg:
		rsp = onUpdateRobotItemCfg(msg)
	case clubProto.ID_GetRobotCfg:
		rsp = onGetClubRobotCfg(msg)
	case clubProto.ID_GetRobotItemCfg:
		rsp = onGetClubRobotItemCfg(msg)
	default:
		return false
	}

	if rsp == nil {
		return true
	}

	msg.Data, _ = json.Marshal(rsp)
	err := wrapMQ.PublishProto(msg.Source, msg)
	if err != nil {
		glog.Warning("uid:=", msg.SenderID,
			" ,wrapMQ.PublishData err. err:=", err.Error(),
			",msg ID:=", msg.MessageID, " ,data:=", string(msg.Data))
	}

	return true
}

func handleClubWriteEvent(msg *mateProto.MessageMaTe) bool {
	var rsp *mateProto.JsonResponse

	nowT1 := time.Now()

	defer func() {
		nowT2 := time.Now()
		dur := nowT2.Sub(nowT1)
		if dur.Milliseconds() >= 50 {
			glog.Warning("msg handle time too long. expend ", dur.Milliseconds(), " ms."+
				" msg:=", msg.MessageID, ",sender ID:=", msg.SenderID, ",data:=", len(msg.Data))
		}
	}()

	switch msg.MessageID {
	// 俱乐部 成员
	case clubProto.ID_ClubCreateTable:
		rsp = onCreateClubTable(msg)
	case clubProto.ID_ClubJoinTable:
		rsp = onJoinClubTable(msg)
	case clubProto.ID_QuickStart:
		rsp = onQuickStartGame(msg)
	case clubProto.ID_ReplyQuickStart:
		rsp = replyQuickStart(msg)
	case clubProto.ID_DragIntoClub:
		//rsp = onDragIntoClub(msg)
		rsp = onInviteJoinClub(msg)
	case clubProto.ID_ApplyJoinClub:
		rsp = onApplyJoinClub(msg)
	case clubProto.ID_ExitClub:
		rsp = onExitClub(msg)
	case clubProto.ID_PutMutexPlayer:
		rsp = onPutMutexPlayer(msg)
	case clubProto.ID_DeleteMutexPlayerGroup:
		rsp = onDeleteMutexPlayerGroup(msg)
	//case clubProto.ID_PlayerSceneChanged:
	//rsp = onUpdateMemberOnline(msg)
	//case clubProto.ID_GetMzOnlineMember:
	//rsp = onGetMzMember(msg)
	//case clubProto.ID_InviteMzOnlineMember:
	//rsp = onInviteMzOnlineMember(msg)
	case clubProto.ID_HandleInviteJoinClub:
		rsp = onHandleInviteJoinClub(msg)

	case clubProto.ID_GetActivitySort:
		rsp = onActivitySort(msg)
	case mateProto.ID_ActivityGameData:
		onActivityGameData(msg)
	case mateProto.ID_NoticeClubActivity:
		onActivityStatus(msg)
	case clubProto.ID_ActivityBlackList:
		rsp = onActivityBlackList(msg)

	// 俱乐部
	case clubProto.ID_PutClubPlay:
		rsp = onPutClubPlay(msg)
	case clubProto.ID_PutClubPlay_RPC:
		rsp = onPRCPutClubPlay(msg)
	case clubProto.ID_DeleteClubPlay:
		rsp = onDeleteClubPlay(msg)
	case clubProto.ID_MemberOperation:
		rsp = onClubMemberOperation(msg)
	case clubProto.ID_ClubOperation:
		rsp = onClubOperation(msg)
	case clubProto.ID_CheckApplyJoinClub:
		rsp = onCheckApplyJoinClub(msg)
	//case clubProto.ID_CheckApplyMergeClub:
	//rsp = onCheckMergeClub(msg)
	//case clubProto.ID_ApplyMergeClub:
	//rsp = onApplyMergeClub(msg)
	//case clubProto.ID_ApplyExitLeague:
	//rsp = onApplyExitLeague(msg)
	//case clubProto.ID_CheckExitLeague:
	//rsp = onCheckExitLeague(msg)
	case clubProto.ID_CheckExitClub:
		rsp = onCheckApplyExitClub(msg)
	case clubProto.ID_SetClubScore0:
		rsp = onSetClubScore0(msg)
	case clubProto.ID_UpdateClubStatus:
		rsp = onUpdateClubStatus(msg)
	case mateProto.ID_CurrentTableCount:
		_curTableCount = msg.SenderID
	//
	case clubProto.ID_UpdateRobotCfg:
		onUpdateRobotConfig(msg)

	// 游戏服
	case mateProto.ID_GameSignIn:
		rsp = onGameSignIn(msg)
	case mateProto.ID_BroadGameServiceStatus:
		rsp = onGameServiceStatus(msg)
	// login
	case mateProto.ID_PlayerInfoUpdate:
		onUpdatePlayerInfo(msg)
	// hall
	case mateProto.ID_HallUpdateClub:
		delAllClubData(true)
	//case mateProto.ID_PlayerOnline:
	//	onUpdateMemberOffline(msg)
	// 俱乐部管理
	case mateProto.ID_NoticeClubMGRLaunch:
		clubMGRLaunch(msg)
	case clubProto.ID_UpdateClubLevel:
		rsp = onUpdateClubLevel(msg)
	default:
		return false
	}

	if rsp == nil {
		return true
	}

	msg.Data, _ = json.Marshal(rsp)
	err := wrapMQ.PublishProto(msg.Source, msg)
	if err != nil {
		glog.Warning("uid:=", msg.SenderID,
			" ,wrapMQ.PublishData err. err:=", err.Error(),
			",msg ID:=", msg.MessageID, " ,data:=", string(msg.Data))
	}

	return true
}

func HandleClubEvent() {

	commonDef.Wait.Add(1)
	defer commonDef.Wait.Done()

	for msg := range clubEventChan {
		if handleClubWriteEvent(msg) == false {
			go handleClubReadEvent(msg)
		}
		if commonDef.IsRun == false {
			break
		}
	}
}

func ActivityTimer(clubID int32) {
	if clubID < 1 {
		return
	}

	commonDef.Wait.Add(1)
	defer commonDef.Wait.Done()

	// 查看是否有历史活动记录
	acJson, err := db.GetClubActivity(clubID)
	if err != nil {
		if err != redis.Nil {
			glog.Warning("GetClubActivity() err.", err.Error())
		}
	} else if len(acJson) > 0 {
		err = json.Unmarshal(acJson, &_activityMap)
		if err != nil {
			glog.Warning("Unmarshal Activity() err.", err.Error())
		}
		for _, v := range _activityMap {
			v.Score, _ = commonDef.TextScoreToService(v.ScoreString)
		}
	}
	lastAcRuleBytes, err := db.GetLastClubActivityRule(clubID)
	if err != nil {
		if err != redis.Nil {
			glog.Warning("GetLastClubActivityRule  ", err.Error())
		}
	} else {
		_t := collClub.DBClubActivity{}
		err = json.Unmarshal(lastAcRuleBytes, &_t)
		if err != nil {
			glog.Warning("Unmarshal Activity() err.", err.Error())
		} else {
			_activityRule = &_t
		}
	}

	// 读取黑名单
	blacklistItem_ := []collClub.DBBlacklistItem{}
	err = db.GetBlackList(clubID, &blacklistItem_)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			glog.Warning("GetBlackList() err.", err.Error())
		}
	}
	for i, _ := range blacklistItem_ {
		_blackListMap[blacklistItem_[i].Uid] = &blacklistItem_[i]
	}

	acInfo := collClub.DBClubActivity{}
	err = db.GetMengZhuActivity(clubID, &acInfo)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			glog.Fatal(err.Error(), ", clubID:=", clubID)
		}
	} else {
		// 当前和历史是一致的
		if _activityRule != nil && _activityRule.Rule.ActivityID == acInfo.Rule.ActivityID {
			_activityRule = &acInfo
		}

		// 恢复历史活动
		_activityST = acInfo.Rule.MatchS.Unix()
		_isChanged = len(_activityMap) > 0

		var clubData *collClub.DBClubData
		clubData, _ = loadClubData(clubID)
		clubData.Activity = &acInfo

		// 记录未开始的活动
		_activityWaitOpen.Store(acInfo.Rule.ActivityID, &acInfo)
	}

	_, _, lastDay := time.Now().Date()

	for commonDef.IsRun {
		tt := time.Now()
		unixTT := tt.Unix()

		_activityWaitOpen.Range(func(key, value interface{}) bool {
			_id := key.(primitive.ObjectID)
			_v := value.(*collClub.DBClubActivity)
			diff := time.Duration(_v.Rule.MatchS.Unix() - unixTT)
			// 定时发射 [1s - 3m]
			if diff > 0 && diff < 60*3 &&
				_v.NoticeOpenTimer == nil {
				// 提前1秒 告知
				_v.NoticeOpenTimer = time.AfterFunc(time.Second*(diff-1), func() {

					// 通知 DB/自己 活动开始
					clubEventChan <- noticeDBClubActivityStatus(_v, true)
				})
			} else if diff < -60 {
				_activityWaitOpen.Delete(_id)
			}
			return true
		})

		_activityIng.Range(func(key, value interface{}) bool {
			_id := key.(primitive.ObjectID)
			_v := value.(*collClub.DBClubActivity)
			diff := time.Duration(_v.Rule.MatchE.Unix() - unixTT)
			// 定时发射 [1s - 3m]
			if diff >= 0 && diff < 60*3 &&
				_v.NoticeCloseTimer == nil {

				// 定时 告知结束
				_v.NoticeCloseTimer = time.AfterFunc(time.Second*diff, func() {
					// 通知 DB/自己 活动结束
					clubEventChan <- noticeDBClubActivityStatus(_v, false)
				})
			} else if diff < -60 {
				_activityIng.Delete(_id)
			}
			return true
		})

		// 删除 积分日志
		_, _, tDay := tt.Date()
		if tDay != lastDay {
			db.DeleteClubScoreLog(clubID)
		}

		time.Sleep(time.Minute)
	}

	if len(_blackListMap) > 0 {
		blackListGetAll()
		err = db.PutBlackList(clubID, _blackListArr)
		if err != nil {
			glog.Warning("db.PutBlackList() err. ", err.Error())
		}
	}
}

func noticeDBServiceClubChanged(clubID int32) {
	msg := mateProto.MessageMaTe{MessageID: mateProto.ID_UpdateClubData}
	msg.Data, _ = json.Marshal(&mateProto.SS_UpdateClubData{ClubID: clubID})
	err := wrapMQ.PublishProto("db", &msg)
	if err != nil {
		glog.Warning("uid:=", msg.SenderID,
			" ,wrapMQ.PublishData err. err:=", err.Error(),
			",msg ID:=", msg.MessageID, " ,data:=", string(msg.Data))
	}

	err = wrapMQ.PublishProto("club", &msg)
	if err != nil {
		glog.Warning("uid:=", msg.SenderID,
			" ,wrapMQ.PublishData err. err:=", err.Error(),
			",msg ID:=", msg.MessageID, " ,data:=", string(msg.Data))
	}
}

func noticeClubMGRClubChanged(clubID int32) {
	msg := mateProto.MessageMaTe{MessageID: mateProto.ID_UpdateClubData}
	msg.Data, _ = json.Marshal(&mateProto.SS_UpdateClubData{ClubID: clubID})
	err := wrapMQ.PublishProto("club", &msg)
	if err != nil {
		glog.Warning("uid:=", msg.SenderID,
			" ,wrapMQ.PublishData err. err:=", err.Error(),
			",msg ID:=", msg.MessageID, " ,data:=", string(msg.Data))
	}
}

func noticeDBServiceClubDeleteTotal(superArr []int32, clubID int32, sub []int32) {
	msg := mateProto.MessageMaTe{MessageID: mateProto.ID_DeleteClubTotal}
	msg.Data, _ = json.Marshal(&mateProto.SS_DeleteClubTotal{
		SuperiorClubID:    superArr,
		ClubID:            clubID,
		SubordinateClubID: sub,
	})
	err := wrapMQ.PublishProto("db", &msg)
	if err != nil {
		glog.Warning("uid:=", msg.SenderID,
			" ,wrapMQ.PublishData err. err:=", err.Error(),
			",msg ID:=", msg.MessageID, " ,data:=", string(msg.Data))
	}
}

func onGetClubTables(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_GetTable{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}
	if param.BeginIndex < 0 || param.BeginIndex > 1000 ||
		param.StopIndex < 1 || param.StopIndex > 1000 {
		rsp.Status, rsp.Detail = errorCodeDef.Err_Param, "page"
		return rsp
	}

	var clubData *collClub.DBClubData
	clubData, err = loadClubData(param.ClubID)
	if err != nil {
		rsp.Status = errorCodeDef.ErrClubNotExist
		return rsp
	}
	if _, ok := clubData.MemberMap[msg.SenderID]; ok == false {
		rsp.Status = errorCodeDef.ErrClubNotMember
		return rsp
	}

	param.ClubVersionNumber = clubData.ClubVerNum

	if clubData.MengZhuClubID > 0 {
		param.ClubID = clubData.MengZhuClubID
		clubData, err = loadClubData(clubData.MengZhuClubID)
		if err != nil {
			rsp.Status = errorCodeDef.ErrClubNotExist
			return rsp
		}
	}

	param.CurVersionNum = clubData.PlayVersionNum
	msg.MsgBody = &param

	tableSnapshotEvent.PostMaTeEvent(msg)
	return nil
}

func noticeDBClubActivityStatus(param *collClub.DBClubActivity, isOpen bool) *mateProto.MessageMaTe {
	noticeMsg := mateProto.MessageMaTe{MessageID: mateProto.ID_NoticeClubActivity}
	noticeMsg.Data, _ = json.Marshal(&mateProto.SS_NoticeClubActivity{
		ClubID:  param.ClubID,
		EndTime: param.Rule.MatchE.Unix(),
		AcID:    param.Rule.ActivityID,
		IsOpen:  isOpen,
	})
	err := wrapMQ.PublishProto("db", &noticeMsg)
	if err != nil {
		glog.Warning(" ,wrapMQ.PublishData err. err:=", err.Error(),
			",msg ID:=", noticeMsg.MessageID, " ,data:=", string(noticeMsg.Data))
	}
	return &noticeMsg
}

func onUpdateRobotCfg(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := mateProto.CS_UpdateClubRobotConfig{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	clubData, err = loadClubData(param.MZClubID)
	if err != nil {
		rsp.Status = errorCodeDef.ErrClubNotExist
		return rsp
	}
	if clubData.DirectSupervisor.ClubID > 0 {
		rsp.Status = errorCodeDef.ErrFindSuperiorExist
		return rsp
	}
	v, ok := clubData.PlayIDMap[param.ClubPlayID]
	if ok == false {
		rsp.Status = errorCodeDef.ErrClubNotFindPlayID
		return rsp
	}
	param.GameID = v.GameID

	msg.Data, _ = json.Marshal(&param)
	data, _ := json.Marshal(msg)
	wrapMQ.ForwardTo("robot", &data)

	return nil
}

func onUpdateRobotConfig(msg *mateProto.MessageMaTe) {

	param := dbCollectionDefine.DBRobotClubPlayConfig{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		return
	}

	var clubData *collClub.DBClubData
	clubData, err = loadClubData(param.MZClubID)
	if err != nil {
		return
	}
	v, ok := clubData.PlayIDMap[param.ClubPlayID]
	if ok == false {
		return
	}
	v.RobotJoinPlaying = param.ClubPlayItem.JoinPlayingTable
	v.RobotJoinReady = param.ClubPlayItem.JoinReadyTable
	v.RobotInviteTimer = param.ClubPlayItem.CheckTime
	v.RobotOpen = param.ClubPlayItem.Open
}

func onUpdateRobotItemCfg(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := mateProto.CS_UpdateRobotItemCfg{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	switch param.WinLevel {
	case 10, 5, 1, 0:
	default:
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	var clubData *collClub.DBClubData
	clubData, err = loadClubData(param.MZClubID)
	if err != nil {
		rsp.Status = errorCodeDef.ErrClubNotExist
		return rsp
	}
	if clubData.DirectSupervisor.ClubID > 0 {
		rsp.Status = errorCodeDef.ErrFindSuperiorExist
		return rsp
	}

	//if _, ok := clubData.PlayIDMap[param.ClubPlayID]; ok == false {
	//	rsp.Status = errorCodeDef.ErrClubNotFindPlayID
	//	return rsp
	//}
	if clubData.ClubID != param.ClubID {
		subClubData, err := loadClubData(param.ClubID)
		if err != nil {
			rsp.Status = errorCodeDef.Err_Failed
			return rsp
		}
		if _, ok := subClubData.MemberMap[param.UID]; ok == false {
			rsp.Status = errorCodeDef.ErrClubNotMember
			return rsp
		}
	} else {
		if _, ok := clubData.MemberMap[param.UID]; ok == false {
			rsp.Status = errorCodeDef.ErrClubNotMember
			return rsp
		}
	}

	data, _ := json.Marshal(msg)
	wrapMQ.ForwardTo("robot", &data)

	return rsp
}

func noticeRobotDeleteClubPlay(mzClubID int32, playID int64) {
	noticeMsg := mateProto.MessageMaTe{MessageID: mateProto.ID_NoticeRobotDeleteClubPlay}
	noticeMsg.Data, _ = json.Marshal(&mateProto.SS_NoticeRobotDeleteClubPlay{
		MZClubID:   mzClubID,
		ClubPlayID: playID,
	})
	err := wrapMQ.PublishProto("robot", &noticeMsg)
	if err != nil {
		glog.Warning(" ,wrapMQ.PublishData err. err:=", err.Error(),
			",msg ID:=", noticeMsg.MessageID, " ,data:=", string(noticeMsg.Data))
	}
}
