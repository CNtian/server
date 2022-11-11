package clubEvent

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"strconv"
	"time"
	"vvService/appClub/db"
	clubProto "vvService/appClub/protoDefine"
	commonDef "vvService/commonPackge"
	"vvService/commonPackge/errorCodeDef"
	"vvService/commonPackge/mateProto"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
)

//type WrapPlayerTotalGameItem struct {
//	Item           *dbCollectionDefine.DBClubPlayerTotal
//	GameRoundIndex int
//	GameScoreIndex int
//}
//
//type PlayerGameRoundTotalSortItem []*WrapPlayerTotalGameItem
//
//func (s PlayerGameRoundTotalSortItem) Len() int { return len(s) }
//func (s PlayerGameRoundTotalSortItem) Swap(i, j int) {
//	s[i], s[j] = s[j], s[i]
//	s[i].GameRoundIndex, s[j].GameRoundIndex = i, j
//}
//func (s PlayerGameRoundTotalSortItem) Less(i, j int) bool {
//	if s[i].Item.GameCount > s[j].Item.GameCount {
//		return true
//	}
//	return false
//}
//
//type PlayerGameScoredTotalSortItem []*WrapPlayerTotalGameItem
//
//func (s PlayerGameScoredTotalSortItem) Len() int { return len(s) }
//func (s PlayerGameScoredTotalSortItem) Swap(i, j int) {
//	s[i], s[j] = s[j], s[i]
//	s[i].GameScoreIndex, s[j].GameScoreIndex = i, j
//}
//func (s PlayerGameScoredTotalSortItem) Less(i, j int) bool {
//	if s[i].Item.GameScore > s[j].Item.GameScore {
//		return true
//	}
//	return false
//}
//
//var (
//	lock_gsSort      sync.Mutex
//	gameScoreSortArr PlayerGameRoundTotalSortItem
//
//	gameRoundSortArr PlayerGameScoredTotalSortItem
//)

const (
	qSubID_GameScore       = 1 // 得分排行
	qSubID_GameRound       = 2 // 局数排行
	qSubID_GameRecord      = 3 // 战绩记录
	qSubID_Captain_Total   = 4 // 队长管理-队长统计
	qSubID_Captain_My      = 5 // 队长管理-我的队长
	qSubID_Captain_Member  = 6 // 队长管理-我的玩家
	qSubID_Captain_SafeBox = 7 // 队长管理-保险箱
	qSubID_Captain_All     = 8 // 队长管理-队长成员

	qSubID_SafeBox_JL_Detail   = 9  // 队长管理-保险箱 奖励 详情
	qSubID_SafeBox_BD_Detail   = 10 // 队长管理-保险箱 保底 详情
	qSubID_SafeBox_JL_Received = 11 // 队长管理-保险箱 奖励 领取
	qSubID_SafeBox_BD_Received = 12 // 队长管理-保险箱 保底 领取

	qSubID_GameTotal          = 20 // 个人数据-游戏统计
	qSubID_PersionalClubTotal = 21 // 个人数据-亲友圈统计

	qSubID_QueryTongZhuo = 22 // 个人数据-同桌统计
)

func onQueryTotal(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {

	req := clubProto.CS_QueryTotal{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		return &mateProto.JsonResponse{Status: errorCodeDef.Err_Illegal_JSON}
	}
	rsp := &mateProto.JsonResponse{SubID: req.SubID}

	//{  //测试
	//	req.SubID = qSubID_GameTotal
	//	d, _ := json.Marshal(&ReqGameTotal{OperationClubID: 800096, QueryClubID: 800096, QueryUID: 107, PageSize: 10, CurPage: 0})
	//	req.Data = string(d)
	//}

	switch req.SubID {
	case qSubID_GameScore:
		onGetGameScoreRank(msg.SenderID, &req, rsp)
	case qSubID_GameRound:
		onGameRoundRank(msg.SenderID, &req, rsp)
	case qSubID_GameRecord:
		onGameRecord(msg.SenderID, &req, rsp)
	case qSubID_Captain_Total:
		onCaptainTotal(msg.SenderID, &req, rsp)
	case qSubID_Captain_My:
		onMyCaptain(msg.SenderID, &req, rsp)
	case qSubID_Captain_Member:
		onCaptainMember(msg.SenderID, &req, rsp)
	case qSubID_Captain_SafeBox:
		onCaptainSafeBox(msg.SenderID, &req, rsp)
	case qSubID_Captain_All:
		onCaptainAll(msg.SenderID, &req, rsp)

	case qSubID_SafeBox_JL_Detail:
		onSafeBoxJL_Detail(msg.SenderID, &req, rsp)
	case qSubID_SafeBox_BD_Detail:
		onSafeBoxBD_Detail(msg.SenderID, &req, rsp)
	case qSubID_SafeBox_JL_Received:
		onReceivedJiangLi(msg.SenderID, &req, rsp)
	case qSubID_SafeBox_BD_Received:
		onReceivedBaoDi(msg.SenderID, &req, rsp)

	case qSubID_GameTotal:
		onGameTotal(msg.SenderID, &req, rsp)
	case qSubID_PersionalClubTotal:
		onPersonalGameTotal(msg.SenderID, &req, rsp)
	case qSubID_QueryTongZhuo:
		onQueryTongZhuo(msg.MZID, msg.SenderID, &req, rsp)
	default:

	}

	return rsp
}

type ReqRankList struct {
	QueryClubID  int32 `json:"curClubID"` // 操作人 所属俱乐部ID
	TargetClubID int32 `json:"toClubID"`  // 俱乐部ID
	Date         int   `json:"date"`      // 指定日期
	PageSize     int   `json:"pageSize"`
	CurPage      int   `json:"curPage"`
}

func onGetGameScoreRank(senderID int64, req *clubProto.CS_QueryTotal, rsp *mateProto.JsonResponse) {

	type RspGetGameScoreRank struct {
		Arr      []dbCollectionDefine.DBClubPlayerTotal `json:"item"`
		Self     dbCollectionDefine.DBClubPlayerTotal   `json:"self"`
		SelfRank int                                    `json:"rank"`
	}

	rspData := RspGetGameScoreRank{}
	rspData.Self.PlayerID = senderID
	rsp.Data = &rspData

	param := ReqRankList{}
	err := json.Unmarshal([]byte(req.Data), &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return
	}
	if param.CurPage > 100 || param.CurPage < 0 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}
	if param.PageSize > 10 || param.PageSize < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}

	clubData, err := loadClubData(param.QueryClubID)
	if rsp.Status != 0 {
		return
	}
	if _, ok := clubData.MemberMap[senderID]; ok == false {
		rsp.Status = errorCodeDef.ErrClubNotMember
		return
	}
	if clubData.IsShowRankList == false {
		if clubData.DirectSupervisor.ClubID < 1 && clubData.CreatorID == senderID {

		} else {
			rsp.Status = errorCodeDef.ErrPowerNotEnough
			return
		}
	}

	mzCID := clubData.ClubID
	if clubData.MengZhuClubID > 0 {
		mzCID = clubData.MengZhuClubID
	}

	type MGGetGameScoreRank struct {
		Value []dbCollectionDefine.DBClubPlayerTotal `bson:"value"`
		Page  []dbCollectionDefine.DBClubPlayerTotal `bson:"page"`
		Rank  []struct {
			Index int `bson:"index"`
		} `bson:"rank"`
	}
	tempRsp := &MGGetGameScoreRank{}
	err = db.GetGameScoreRank(param.PageSize, param.CurPage, param.Date, mzCID, func(cursor *mongo.Cursor) error {
		return cursor.Decode(tempRsp)
	}, senderID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return
		}
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning(err.Error(), ", req:=", *req)
		return
	}
	rspData.Arr = tempRsp.Page
	if len(tempRsp.Value) > 0 {
		rspData.Self.GameScore = tempRsp.Value[0].GameScore
	}
	if len(tempRsp.Rank) > 0 {
		rspData.SelfRank = tempRsp.Rank[0].Index
	}

	for i, _ := range rspData.Arr {
		rspData.Arr[i].GameScoreText = commonDef.ScoreToClient(rspData.Arr[i].GameScore)
	}
	rspData.Self.GameScoreText = commonDef.ScoreToClient(rspData.Self.GameScore)

}

func onGameRoundRank(senderID int64, req *clubProto.CS_QueryTotal, rsp *mateProto.JsonResponse) {

	type RspGameRoundRank struct {
		Arr      []dbCollectionDefine.DBClubPlayerTotal `json:"item"`
		Self     dbCollectionDefine.DBClubPlayerTotal   `json:"self"`
		SelfRank int                                    `json:"rank"`
	}

	rspData := RspGameRoundRank{}
	rspData.Self.PlayerID = senderID
	rsp.Data = &rspData

	param := ReqRankList{}
	err := json.Unmarshal([]byte(req.Data), &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return
	}
	if param.CurPage > 100 || param.CurPage < 0 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}
	if param.PageSize > 10 || param.PageSize < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}

	clubData, err := loadClubData(param.QueryClubID)
	if rsp.Status != 0 {
		return
	}
	if _, ok := clubData.MemberMap[senderID]; ok == false {
		rsp.Status = errorCodeDef.ErrClubNotMember
		return
	}
	if clubData.IsShowRankList == false {
		if clubData.DirectSupervisor.ClubID < 1 && clubData.CreatorID == senderID {

		} else {
			rsp.Status = errorCodeDef.ErrPowerNotEnough
			return
		}
	}

	mzCID := clubData.ClubID
	if clubData.MengZhuClubID > 0 {
		mzCID = clubData.MengZhuClubID
	}

	type MGGetGameScoreRank struct {
		Value []dbCollectionDefine.DBClubPlayerTotal `bson:"value"`
		Page  []dbCollectionDefine.DBClubPlayerTotal `bson:"page"`
		Rank  []struct {
			Index int `bson:"index"`
		} `bson:"rank"`
	}
	tempRsp := &MGGetGameScoreRank{}
	err = db.GetGameRoundRank(param.PageSize, param.CurPage, param.Date, mzCID, func(cursor *mongo.Cursor) error {
		return cursor.Decode(tempRsp)
	}, senderID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return
		}
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning(err.Error(), ", req:=", *req)
		return
	}
	rspData.Arr = tempRsp.Page
	if len(tempRsp.Value) > 0 {
		rspData.Self.GameCount = tempRsp.Value[0].GameCount
	}
	if len(tempRsp.Rank) > 0 {
		rspData.SelfRank = tempRsp.Rank[0].Index
	}
}

type ReqGameRecord struct {
	OperationClubID int32 `json:"curClubID"` // 操作人 所属俱乐部ID
	TargetClubID    int32 `json:"toClubID"`  // 俱乐部ID

	Date       int   `json:"date"`       // 指定日期
	ClubPlayID int64 `json:"clubPlayID"` // 俱乐部玩法ID
	PlayerID   int64 `json:"playerID"`   // 玩家ID
	IsCaptain  bool  `json:"captain"`    // 查队长
	TableID    int   `json:"tableID"`    // 桌子ID
	PageSize   int   `json:"pageSize"`
	CurPage    int   `json:"curPage"`
	QueryType  int   `json:"queryT"` // 查询类型 0:我的  1:小队  2:全队
}

func onGameRecord(senderID int64, req *clubProto.CS_QueryTotal, rsp *mateProto.JsonResponse) {
	type RspGameRecord struct {
		Arr []dbCollectionDefine.DBGameOverRecord `json:"item"`
	}

	rspData := RspGameRecord{}
	rsp.Data = &rspData

	param := ReqGameRecord{}
	err := json.Unmarshal([]byte(req.Data), &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return
	}
	if param.CurPage > 100 || param.CurPage < 0 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}
	if param.PageSize > 10 || param.PageSize < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}

	defer func() {
		for i, _ := range rspData.Arr {
			for _, v := range rspData.Arr[i].PlayerScore {
				v.ScoreText = commonDef.ScoreToClient(v.SScore)
			}
		}
	}()

	var (
		filter bson.D
	)

	if param.QueryType == 0 {
		if param.TableID != 0 {
			filter = append(filter, bson.E{"table_id", param.TableID})
		}
		if param.ClubPlayID != 0 {
			filter = append(filter, bson.E{"club_play_id", param.ClubPlayID})
		}
		filter = append(filter, bson.E{"players.uid", senderID})
		err = db.GetGameRecord(param.PageSize, param.CurPage, param.Date, filter, &rspData.Arr)
		if err != nil && err != mongo.ErrNilDocument {
			rsp.Status = errorCodeDef.Err_Failed
			glog.Warning(err.Error(), ", req:=", *req)
		}
		return
	}
	if param.QueryType == 1 {

		var clubData *collClub.DBClubData
		rsp.Status, clubData = checkClubPower(param.OperationClubID, senderID)
		if rsp.Status != 0 {
			return
		}
		if param.TableID != 0 {
			filter = append(filter, bson.E{"table_id", param.TableID})
		}
		if param.ClubPlayID != 0 {
			filter = append(filter, bson.E{"club_play_id", param.ClubPlayID})
		}
		if param.PlayerID != 0 && param.IsCaptain == false {
			filter = append(filter, bson.E{"players.uid", param.PlayerID})
		}
		if param.IsCaptain && param.PlayerID > 0 {
			clubID := int32(0)
			for _, pv := range getAllMember(clubData) {
				if pv.ID == param.PlayerID {
					clubID = pv.OfClubID
					break
				}
			}
			filter = append(filter, bson.E{"players.clubID", clubID})
		}
		if param.TargetClubID == 0 && param.IsCaptain == false {
			filter = append(filter, bson.E{"players.clubID", clubData.ClubID})
		} else if param.TargetClubID > 0 {
			if _, ok := clubData.SubordinatesMap[param.TargetClubID]; ok == false {
				rsp.Status = errorCodeDef.ErrClubRelation
				return
			}
			filter = append(filter, bson.E{"players.clubID", param.TargetClubID})
		}
		err = db.GetGameRecord(param.PageSize, param.CurPage, param.Date, filter, &rspData.Arr)
		if err != nil && err != mongo.ErrNilDocument {
			rsp.Status = errorCodeDef.Err_Failed
			glog.Warning(err.Error(), ", req:=", *req)
		}
		return
	}
	if param.QueryType == 2 {
		var clubData *collClub.DBClubData
		rsp.Status, clubData = checkClubPower(param.OperationClubID, senderID)
		if rsp.Status != 0 {
			return
		}
		if param.TableID != 0 {
			filter = append(filter, bson.E{"table_id", param.TableID})
		}
		if param.ClubPlayID != 0 {
			filter = append(filter, bson.E{"club_play_id", param.ClubPlayID})
		}
		if param.PlayerID != 0 && param.IsCaptain == false {
			filter = append(filter, bson.E{"players.uid", param.PlayerID})
		}
		if param.IsCaptain && param.PlayerID > 0 {
			clubID := int32(0)
			for _, pv := range getAllMember(clubData) {
				if pv.ID == param.PlayerID {
					clubID = pv.OfClubID
					break
				}
			}
			filter = append(filter, bson.E{"players.clubID", clubID})
		}
		if param.TargetClubID == 0 && param.IsCaptain == false {
			sub := clubData.Subordinates
			sub = append(sub, clubData.ClubID)
			filter = append(filter, bson.E{"players.clubID", bson.M{"$in": sub}})
		} else if param.TargetClubID > 0 {
			if _, ok := clubData.SubordinatesMap[param.TargetClubID]; ok == false {
				rsp.Status = errorCodeDef.ErrClubRelation
				return
			}
			filter = append(filter, bson.E{"players.clubID", param.TargetClubID})
		}
		err = db.GetGameRecord(param.PageSize, param.CurPage, param.Date, filter, &rspData.Arr)
		if err != nil && err != mongo.ErrNilDocument {
			rsp.Status = errorCodeDef.Err_Failed
			glog.Warning(err.Error(), ", req:=", *req)
		}
		return
	}
}

type ReqCaptainTotal struct {
	OperationClubID int32 `json:"curClubID"` // 操作人 所属俱乐部ID
}

// 队长管理-队长统计
func onCaptainTotal(senderID int64, req *clubProto.CS_QueryTotal, rsp *mateProto.JsonResponse) {

	type RspCaptainTotal struct {
		TodayGX     string `json:"todayGX"`
		YesterdayGX string `json:"yesGX"`

		TodaySY     string `json:"todaySY"`
		YesterdaySY string `json:"yesSY"`

		TotalScore string `json:"totalScore"`
		SYBL       int32  `json:"syBL"`
		AllMember  int    `json:"allMember"`
	}

	rspData := RspCaptainTotal{}
	rsp.Data = &rspData

	param := ReqCaptainTotal{}
	err := json.Unmarshal([]byte(req.Data), &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.OperationClubID, senderID)
	if rsp.Status != 0 {
		return
	}

	now := time.Now()
	y, m, d := now.Date()
	today_, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", y, m, d))

	td, _ := time.ParseDuration("-24h")
	y, m, d = time.Now().Add(td).Date()
	yesterday_, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", y, m, d))

	arr := []dbCollectionDefine.DBClubTotal{}
	err = db.JZGetClubTotal(today_, yesterday_, clubData.ClubID, &arr)
	if err != nil && err != mongo.ErrNilDocument {
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning(err.Error(), ", req:=", *req)
		return
	}

	for i := 0; i < len(arr); i++ {
		if arr[i].Date == int32(today_) {
			rspData.TodayGX = commonDef.ScoreToClient(arr[i].XiaoHaoCount)
			rspData.TodaySY = commonDef.ScoreToClient(arr[i].JiangLi)
		} else if arr[i].Date == int32(yesterday_) {
			rspData.YesterdayGX = commonDef.ScoreToClient(arr[i].XiaoHaoCount)
			rspData.YesterdaySY = commonDef.ScoreToClient(arr[i].JiangLi)
		}
	}
	scoreTotal, err := db.GetClubCountScore(clubData.ClubID)
	if err != nil {
		glog.Warning("err := ", err.Error(), ", clubID := ", clubData.ClubID)
	} else {
		rspData.TotalScore = commonDef.ScoreToClient(scoreTotal)
	}

	if clubData.BiLiShowWay == 0 {
		rspData.SYBL = clubData.DirectSupervisor.ShowPercentage
	} else {
		rspData.SYBL = clubData.DirectSupervisor.RealPercentage
	}

	rspData.AllMember = len(getAllMember(clubData))
}

type ReqMyCaptain struct {
	OperationClubID int32  `json:"curClubID"` // 操作人 所属俱乐部ID
	QueryUID        int64  `json:"qUID"`
	QueryName       string `json:"qName"`

	Date     int `json:"date"` // 指定日期
	PageSize int `json:"pageSize"`
	CurPage  int `json:"curPage"`
}

// 队长管理-我的队长
func onMyCaptain(senderID int64, req *clubProto.CS_QueryTotal, rsp *mateProto.JsonResponse) {
	type MyCaptainItem struct {
		UID    int64 `json:"uid"`
		ClubID int32 `json:"clubID"`

		TodayGX string `json:"todayGX"`
		TodaySY string `json:"todaySY"`

		TotalScore  string `json:"clubScore"`
		PlayerScore string `json:"score"`

		JuShu  int32  `json:"juShu"`
		FangKa string `json:"fangka"`

		JingJie             string `json:"jingJie"`       // 警戒
		FuFen               string `json:"fuFen"`         // 负分
		IsOpen              bool   `json:"open"`          // 状态 0:正常 1:打烊
		IsFrozen            bool   `json:"frozen"`        // 状态 0:正常 1:冻结
		IsKickOutMember     bool   `json:"kickOutMember"` // 俱乐部是否可以踢出成员
		IsKickOutLeague     bool   `json:"kickOutLeague"` // 俱乐部是否可以踢出
		IsDirectSubordinate bool   `json:"isDire"`        // 是否是直属
	}
	type RspMyCaptain struct {
		Item []MyCaptainItem `json:"item"`
	}

	rspData := RspMyCaptain{}
	rsp.Data = &rspData

	param := ReqMyCaptain{}
	err := json.Unmarshal([]byte(req.Data), &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.OperationClubID, senderID)
	if rsp.Status != 0 {
		return
	}

	//now := time.Now()
	//y, m, d := now.Date()
	//today_, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", y, m, d))

	if len(param.QueryName) > 0 {

		return
	}
	if param.QueryUID != 0 {
		findIndex := 0
		var (
			subClubData      *collClub.DBClubData
			isDirsubClubData bool
		)
		for ; findIndex < len(clubData.DirectSubordinate); findIndex++ {
			if clubData.DirectSubordinate[findIndex].PlayerID != param.QueryUID {
				continue
			}
			subClubData, err = loadClubData(clubData.DirectSubordinate[findIndex].ClubID)
			if err != nil {
				glog.Warning(err.Error(), ",", clubData.DirectSubordinate[findIndex].ClubID)
				rsp.Status = errorCodeDef.Err_Failed
				return
			}
			isDirsubClubData = true
			break
		}
		if subClubData == nil {
			for _, v := range clubData.Subordinates {
				subClubData, err = loadClubData(v)
				if err != nil {
					glog.Warning(err.Error(), ",", v)
					rsp.Status = errorCodeDef.Err_Failed
					return
				}
				if subClubData.CreatorID == param.QueryUID {
					break
				}
				subClubData = nil
			}
		}
		if subClubData == nil {
			return
		}

		rspData.Item = make([]MyCaptainItem, 1)
		rspData.Item[0].ClubID = subClubData.ClubID
		rspData.Item[0].UID = subClubData.CreatorID
		rspData.Item[0].IsDirectSubordinate = isDirsubClubData

		clubTotalMap, err := db.GetClubTotal([]int32{subClubData.ClubID}, param.Date)
		if err != nil {
			glog.Warning(err.Error(), ", ", subClubData.ClubID)
		} else {
			total_, ok := clubTotalMap[subClubData.ClubID]
			if ok == true {
				rspData.Item[0].TodayGX = commonDef.ScoreToClient(total_.GongXian)
				rspData.Item[0].TodaySY = commonDef.ScoreToClient(total_.JiangLi)

				rspData.Item[0].FangKa = commonDef.ScoreToClient(total_.HaoKa)
				rspData.Item[0].JuShu = total_.GameCount
			}
		}

		temp_, _ := db.GetPlayerClubScore(subClubData.ClubID, subClubData.CreatorID)
		rspData.Item[0].PlayerScore = commonDef.ScoreToClient(temp_)

		score_, err := db.GetClubCountScore(subClubData.ClubID)
		if err != nil {
			glog.Warning(err.Error(), ", ", subClubData.ClubID)
		} else {
			rspData.Item[0].TotalScore = commonDef.ScoreToClient(score_)
		}

		rspData.Item[0].JingJie = commonDef.ScoreToClient(subClubData.BaoDiScore)
		rspData.Item[0].IsOpen = subClubData.IsOpen
		rspData.Item[0].IsFrozen = subClubData.IsFrozen
		rspData.Item[0].IsKickOutMember = subClubData.IsKickOutMember
		rspData.Item[0].IsKickOutLeague = subClubData.IsKickOutLeague
		//rspData.Item[0].Remark = subClubData.Remark

		return
	}

	if param.CurPage > 100 || param.CurPage < 0 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}
	if param.PageSize > 10 || param.PageSize < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}

	directSubIDArr := make([]int32, 0, len(clubData.DirectSubordinate))
	if param.CurPage == 0 {
		directSubIDArr = append(directSubIDArr, clubData.ClubID)
	}

	for i := param.CurPage * param.PageSize; i < len(clubData.DirectSubordinate) && len(directSubIDArr) < param.PageSize; i++ {
		directSubIDArr = append(directSubIDArr, clubData.DirectSubordinate[i].ClubID)
	}

	clubTotalMap, err := db.GetClubTotal(directSubIDArr, param.Date)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning(err.Error(), ",  ", directSubIDArr)
		return
	}
	clubCountScoreMap, err := db.GetXClubCountScore(directSubIDArr)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning(err.Error(), ",  ", directSubIDArr)
		return
	}

	rspData.Item = make([]MyCaptainItem, param.PageSize)
	itemIndex := 0

	// 添加自己
	if param.CurPage == 0 {
		rspData.Item[itemIndex].ClubID = clubData.ClubID
		rspData.Item[itemIndex].UID = clubData.CreatorID

		total_, ok := clubTotalMap[clubData.ClubID]
		if ok == true {
			rspData.Item[itemIndex].TodayGX = commonDef.ScoreToClient(total_.XiaoHaoCount)
			rspData.Item[itemIndex].TodaySY = commonDef.ScoreToClient(total_.JiangLi)
			rspData.Item[itemIndex].FangKa = commonDef.ScoreToClient(total_.HaoKa)
			rspData.Item[itemIndex].JuShu = total_.GameCount
		}

		temp_, _ := db.GetPlayerClubScore(clubData.ClubID, clubData.CreatorID)
		rspData.Item[itemIndex].PlayerScore = commonDef.ScoreToClient(temp_)

		score_, ok := clubCountScoreMap[clubData.ClubID]
		if ok == true {
			rspData.Item[itemIndex].TotalScore = commonDef.ScoreToClient(score_)
		}

		rspData.Item[itemIndex].JingJie = commonDef.ScoreToClient(clubData.BaoDiScore)
		rspData.Item[itemIndex].IsOpen = clubData.IsOpen
		rspData.Item[itemIndex].IsFrozen = clubData.IsFrozen
		rspData.Item[itemIndex].IsKickOutMember = clubData.IsKickOutMember
		rspData.Item[itemIndex].IsKickOutLeague = clubData.IsKickOutLeague

		itemIndex++
	}

	for i := param.CurPage * param.PageSize; i < len(clubData.DirectSubordinate) && itemIndex < param.PageSize; i++ {
		rspData.Item[itemIndex].ClubID = clubData.DirectSubordinate[i].ClubID
		rspData.Item[itemIndex].UID = clubData.DirectSubordinate[i].PlayerID
		subClubData, err := loadClubData(clubData.DirectSubordinate[i].ClubID)
		if err != nil {
			rsp.Status = errorCodeDef.Err_Failed
			glog.Warning(err.Error(), ",  ", clubData.DirectSubordinate[i].ClubID)
			return
		}

		total_, ok := clubTotalMap[clubData.DirectSubordinate[i].ClubID]
		if ok == true {
			rspData.Item[itemIndex].TodayGX = commonDef.ScoreToClient(total_.XiaoHaoCount)
			rspData.Item[itemIndex].TodaySY = commonDef.ScoreToClient(total_.JiangLi)

			rspData.Item[itemIndex].FangKa = commonDef.ScoreToClient(total_.HaoKa)
			rspData.Item[itemIndex].JuShu = total_.GameCount
		}

		temp_, _ := db.GetPlayerClubScore(clubData.DirectSubordinate[i].ClubID, clubData.DirectSubordinate[i].PlayerID)
		rspData.Item[itemIndex].PlayerScore = commonDef.ScoreToClient(temp_)

		score_, ok := clubCountScoreMap[clubData.DirectSubordinate[i].ClubID]
		if ok == true {
			rspData.Item[itemIndex].TotalScore = commonDef.ScoreToClient(score_)
		}

		rspData.Item[itemIndex].JingJie = commonDef.ScoreToClient(subClubData.BaoDiScore)
		rspData.Item[itemIndex].IsOpen = subClubData.IsOpen
		rspData.Item[itemIndex].IsFrozen = subClubData.IsFrozen
		rspData.Item[itemIndex].IsKickOutMember = subClubData.IsKickOutMember
		rspData.Item[itemIndex].IsKickOutLeague = subClubData.IsKickOutLeague
		rspData.Item[itemIndex].IsDirectSubordinate = true

		itemIndex++
	}
	if itemIndex < len(rspData.Item) {
		rspData.Item = rspData.Item[:itemIndex]
	}
}

type ReqCaptainMember struct {
	OperationClubID int32  `json:"curClubID"` // 操作人 所属俱乐部ID
	QueryUID        int64  `json:"qUID"`
	QueryName       string `json:"qName"`

	Date     int `json:"date"` // 指定日期
	PageSize int `json:"pageSize"`
	CurPage  int `json:"curPage"`
}

type CaptainMemberItem struct {
	UID    int64 `json:"uid"`
	ClubID int32 `json:"clubID"`

	PlayerScore    string `json:"score"`
	GameScore      string `json:"gScore"`
	GameRoundCount int32  `json:"grCount"`

	TodayGX string `json:"GX"`
	TodaySY string `json:"SY"`

	Role    int32  `json:"role"`    // 0:成员  1:管理员  2:圈主
	Status  int32  `json:"status"`  // 0:正常  1:冻结
	IsStop3 bool   `json:"isStop3"` // 是否 禁止 玩3人局
	IsStop4 bool   `json:"isStop4"` // 是否 禁止 玩4人局
	Remark  string `json:"remark"`  //
	IsRobot int    `json:"rt"`      //

	Robot []dbCollectionDefine.ClubPlayItemCfg `json:"r"`
}
type RspCaptainMember struct {
	Item []CaptainMemberItem `json:"item"`
}

// 队长管理-我的玩家
func onCaptainMember(senderID int64, req *clubProto.CS_QueryTotal, rsp *mateProto.JsonResponse) {

	rspData := RspCaptainMember{}
	rsp.Data = &rspData

	param := ReqCaptainMember{}
	err := json.Unmarshal([]byte(req.Data), &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.OperationClubID, senderID)
	if rsp.Status != 0 {
		return
	}

	//now := time.Now()
	//y, m, d := now.Date()
	//today_, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", y, m, d))

	isRobotMap := map[int64]bool{}

	if len(param.QueryName) > 0 {
		return
	}
	if param.QueryUID != 0 {
		player, ok := clubData.MemberMap[param.QueryUID]
		if ok == false {

			tempClubID := int32(0)
			for _, v := range getAllMember(clubData) {
				if v.ID == param.QueryUID {
					ok = true
					clubData, err = loadClubData(v.OfClubID)
					if err != nil {
						rsp.Status = errorCodeDef.Err_Failed
						glog.Warning(err.Error(), ",", v.OfClubID)
						return
					}
					tempClubID = v.OfClubID
					break
				}
			}
			if tempClubID == 0 {
				return
			}
			player, ok = clubData.MemberMap[param.QueryUID]
		}

		rspData.Item = make([]CaptainMemberItem, 1)
		rspData.Item[0].UID = param.QueryUID
		rspData.Item[0].ClubID = clubData.ClubID

		playerTotalMap_ := map[int64]dbCollectionDefine.DBClubPlayerTotal{}
		err = db.GetPlayerTotal(clubData.ClubID, []int64{param.QueryUID}, param.Date, &playerTotalMap_)
		if err != nil {
			glog.Warning(err.Error())
			rsp.Status = errorCodeDef.Err_Failed
			return
		}
		if v, ok := playerTotalMap_[param.QueryUID]; ok == true {
			rspData.Item[0].TodayGX = commonDef.ScoreToClient(v.XiaoHaoScore)
			rspData.Item[0].TodaySY = commonDef.ScoreToClient(v.GongXian)
			//for i, _ := range v.ClubJiangLi {
			//	if v.ClubJiangLi[i].ClubID == clubData.ClubID {
			//		rspData.Item[0].TodaySY = commonDef.ScoreToClient(v.ClubJiangLi[i].JiangLi)
			//		break
			//	}
			//}
			rspData.Item[0].GameScore = commonDef.ScoreToClient(v.GameScore)
			rspData.Item[0].GameRoundCount = v.GameCount
		}
		if clubData.DirectSupervisor.ClubID < 1 {
			err = db.GetIsRobot(clubData.ClubID, []int64{param.QueryUID}, &isRobotMap)
			if err != nil {
				rsp.Status = errorCodeDef.Err_Failed
				glog.Warning(err.Error(), ",  ", []int64{param.QueryUID}, ", ", param.Date, ",", clubData.ClubID)
				return
			}
		} else {
			err = db.GetIsRobot(clubData.MengZhuClubID, []int64{param.QueryUID}, &isRobotMap)
			if err != nil {
				rsp.Status = errorCodeDef.Err_Failed
				glog.Warning(err.Error(), ",  ", []int64{param.QueryUID}, ", ", param.Date, ",", clubData.ClubID)
				return
			}
		}

		temp_, _ := db.GetPlayerClubScore(clubData.ClubID, param.QueryUID)
		rspData.Item[0].PlayerScore = commonDef.ScoreToClient(temp_)

		if rspData.Item[0].UID == clubData.CreatorID {
			rspData.Item[0].Role = 2
		} else if player.IsAdmin {
			rspData.Item[0].Role = 1
		}
		if player.Frozen.IsFrozen {
			rspData.Item[0].Status = 1
		}
		rspData.Item[0].IsStop3 = player.IsNo3
		rspData.Item[0].IsStop4 = player.IsNo4
		rspData.Item[0].Remark = player.Remark
		if _, ok := isRobotMap[player.ID]; ok == true {
			rspData.Item[0].IsRobot = 1
		}
		return
	}

	if param.CurPage > 100 || param.CurPage < 0 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}
	if param.PageSize > 10 || param.PageSize < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}

	memberIDArr := make([]int64, 0, len(clubData.MemberArr))
	for i := param.CurPage * param.PageSize; i < len(clubData.MemberArr); i++ {
		memberIDArr = append(memberIDArr, clubData.MemberArr[i].ID)
	}
	playerTotalMap_ := map[int64]dbCollectionDefine.DBClubPlayerTotal{}
	err = db.GetPlayerTotal(clubData.ClubID, memberIDArr, param.Date, &playerTotalMap_)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning(err.Error(), ",  ", memberIDArr, ", ", param.Date, ",", clubData.ClubID)
		return
	}

	if clubData.DirectSupervisor.ClubID < 1 {
		err = db.GetIsRobot(clubData.ClubID, memberIDArr, &isRobotMap)
		if err != nil {
			rsp.Status = errorCodeDef.Err_Failed
			glog.Warning(err.Error(), ",  ", memberIDArr, ", ", param.Date, ",", clubData.ClubID)
			return
		}
	} else {
		err = db.GetIsRobot(clubData.MengZhuClubID, memberIDArr, &isRobotMap)
		if err != nil {
			rsp.Status = errorCodeDef.Err_Failed
			glog.Warning(err.Error(), ",  ", memberIDArr, ", ", param.Date, ",", clubData.ClubID)
			return
		}
	}

	rspData.Item = make([]CaptainMemberItem, param.PageSize)
	itemIndex := 0
	for i := param.CurPage * param.PageSize; i < len(clubData.MemberArr) && itemIndex < param.PageSize; i++ {
		rspData.Item[itemIndex].UID = clubData.MemberArr[i].ID
		rspData.Item[itemIndex].ClubID = clubData.ClubID

		if v, ok := playerTotalMap_[clubData.MemberArr[i].ID]; ok == true {
			rspData.Item[itemIndex].TodayGX = commonDef.ScoreToClient(v.XiaoHaoScore)
			rspData.Item[itemIndex].TodaySY = commonDef.ScoreToClient(v.GongXian)
			//for jli, _ := range v.ClubJiangLi {
			//	if v.ClubJiangLi[jli].ClubID == clubData.ClubID {
			//		rspData.Item[itemIndex].TodaySY = commonDef.ScoreToClient(v.ClubJiangLi[jli].JiangLi)
			//		break
			//	}
			//}
			rspData.Item[itemIndex].GameScore = commonDef.ScoreToClient(v.GameScore)
			rspData.Item[itemIndex].GameRoundCount = v.GameCount
		}

		temp_, _ := db.GetPlayerClubScore(clubData.ClubID, clubData.MemberArr[i].ID)
		rspData.Item[itemIndex].PlayerScore = commonDef.ScoreToClient(temp_)

		if rspData.Item[itemIndex].UID == clubData.CreatorID {
			rspData.Item[itemIndex].Role = 2
		} else if clubData.MemberArr[i].IsAdmin {
			rspData.Item[itemIndex].Role = 1
		}
		if clubData.MemberArr[i].Frozen.IsFrozen {
			rspData.Item[itemIndex].Status = 1
		}
		rspData.Item[itemIndex].IsStop3 = clubData.MemberArr[i].IsNo3
		rspData.Item[itemIndex].IsStop4 = clubData.MemberArr[i].IsNo4
		rspData.Item[itemIndex].Remark = clubData.MemberArr[i].Remark
		if _, ok := isRobotMap[clubData.MemberArr[i].ID]; ok == true {
			rspData.Item[itemIndex].IsRobot = 1
		}

		itemIndex++
	}
	if itemIndex < len(rspData.Item) {
		rspData.Item = rspData.Item[:itemIndex]
	}
}

// 队长管理-保险箱
func onCaptainSafeBox(senderID int64, req *clubProto.CS_QueryTotal, rsp *mateProto.JsonResponse) {
	param := ReqCaptainMember{}
	err := json.Unmarshal([]byte(req.Data), &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = IsClubCreator(param.OperationClubID, senderID)
	if rsp.Status != 0 {
		return
	}

	arr := []collClub.DBSafeBox{}
	err = db.GetSafeBoxItemList(clubData.ClubID, clubData.CreatorID, &arr)
	if err != nil && err != mongo.ErrNoDocuments {
		glog.Warning(err.Error(), ",", clubData.ClubID)
		return
	}

	for i, _ := range arr {
		arr[i].BaoDiCountStr = commonDef.ScoreToClient(arr[i].BaoDiCount)
		arr[i].JiangLiCountStr = commonDef.ScoreToClient(arr[i].JiangLiCount)
	}

	rsp.Data = arr
}

type ReqSafeBoxJL_Detail struct {
	OperationClubID int32              `json:"curClubID"` // 操作人 所属俱乐部ID
	LogID           primitive.ObjectID `json:"logID"`

	PageSize int `json:"pageSize"`
	CurPage  int `json:"curPage"`
}

// 保险箱 奖励详情
func onSafeBoxJL_Detail(senderID int64, req *clubProto.CS_QueryTotal, rsp *mateProto.JsonResponse) {
	param := ReqSafeBoxJL_Detail{}
	err := json.Unmarshal([]byte(req.Data), &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return
	}

	if param.CurPage > 100 || param.CurPage < 0 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}
	if param.PageSize > 10 || param.PageSize < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = IsClubCreator(param.OperationClubID, senderID)
	if rsp.Status != 0 {
		return
	}

	logItem, err := db.GetSafeBoxJiangLiItemDetail(param.PageSize, param.CurPage, param.LogID)
	if err != nil && err != mongo.ErrNoDocuments {
		glog.Warning(err.Error(), ",", clubData.ClubID)
		return
	}

	rsp.Data = logItem
}

// 保险箱 保底详情
func onSafeBoxBD_Detail(senderID int64, req *clubProto.CS_QueryTotal, rsp *mateProto.JsonResponse) {
	param := ReqSafeBoxJL_Detail{}
	err := json.Unmarshal([]byte(req.Data), &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return
	}

	if param.CurPage > 100 || param.CurPage < 0 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}
	if param.PageSize > 10 || param.PageSize < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = IsClubCreator(param.OperationClubID, senderID)
	if rsp.Status != 0 {
		return
	}

	logItem, err := db.GetSafeBoxBaoDiItemDetail(param.PageSize, param.CurPage, param.LogID)
	if err != nil && err != mongo.ErrNoDocuments {
		glog.Warning(err.Error(), ",", clubData.ClubID)
		return
	}

	rsp.Data = logItem
}

type ReqReceivedJiangLi struct {
	OperationClubID int32              `json:"curClubID"` // 操作人 所属俱乐部ID
	LogID           primitive.ObjectID `json:"logID"`
}

// 保险箱 奖励 领取
func onReceivedJiangLi(senderID int64, req *clubProto.CS_QueryTotal, rsp *mateProto.JsonResponse) {
	param := ReqReceivedJiangLi{}
	err := json.Unmarshal([]byte(req.Data), &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = IsClubCreator(param.OperationClubID, senderID)
	if rsp.Status != 0 {
		return
	}

	now__ := time.Now()
	nowInt, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", now__.Year(), now__.Month(), now__.Day()))

	var safeBox collClub.DBSafeBox
	err = db.ReceivedJiangLi(param.LogID, &safeBox, nowInt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.Err_NotReceived
			return
		}
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning(err.Error(), ",", clubData.ClubID)
		return
	}

	if safeBox.JiangLiCount < 1 || safeBox.ClubID != clubData.ClubID {
		return
	}

	mzClubID := clubData.ClubID
	clubIDArr := make([]int32, 0, 7)
	tempClubData := clubData
	for i := 0; i < 100; i++ {
		clubIDArr = append(clubIDArr, tempClubData.ClubID)
		if tempClubData.DirectSupervisor.ClubID < 1 {
			mzClubID = tempClubData.ClubID
			break
		}
		tempClubData, err = loadClubData(tempClubData.DirectSupervisor.ClubID)
		if err != nil {
			glog.Warning(err.Error(), ",", tempClubData.DirectSupervisor.ClubID)
			return
		}
	}

	now := time.Now()
	y, m, d := now.Date()
	today_, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", y, m, d))

	curScore, err := db.WriteReceivedSafeBoxLog(today_, mzClubID, clubIDArr, collClub.LogReceivedJL, clubData.CreatorID, safeBox.JiangLiCount)
	if err != nil {
		glog.Warning("WriteReceivedSafeBoxLog(). ", err.Error())
		return
	} else {
		noticePlayerChangedScore(param.OperationClubID, senderID, curScore)
	}
}

// 保险箱 保底 领取
func onReceivedBaoDi(senderID int64, req *clubProto.CS_QueryTotal, rsp *mateProto.JsonResponse) {
	param := ReqReceivedJiangLi{}
	err := json.Unmarshal([]byte(req.Data), &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = IsClubCreator(param.OperationClubID, senderID)
	if rsp.Status != 0 {
		return
	}

	now__ := time.Now()
	nowInt, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", now__.Year(), now__.Month(), now__.Day()))

	var safeBox collClub.DBSafeBox
	err = db.ReceivedBaoDi(param.LogID, &safeBox, nowInt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.Err_NotReceived
			return
		}
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning(err.Error(), ",", clubData.ClubID)
		return
	}

	if safeBox.BaoDiCount < 1 || safeBox.ClubID != clubData.ClubID {
		return
	}

	mzClubID := clubData.ClubID
	clubIDArr := make([]int32, 0, 7)
	tempClubData := clubData
	for i := 0; i < 100; i++ {
		clubIDArr = append(clubIDArr, tempClubData.ClubID)
		if tempClubData.DirectSupervisor.ClubID < 1 {
			mzClubID = tempClubData.ClubID
			break
		}
		tempClubData, err = loadClubData(tempClubData.DirectSupervisor.ClubID)
		if err != nil {
			glog.Warning(err.Error(), ",", tempClubData.DirectSupervisor.ClubID)
			return
		}
	}

	now := time.Now()
	y, m, d := now.Date()
	today_, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", y, m, d))

	curScore, err := db.WriteReceivedSafeBoxLog(today_, mzClubID, clubIDArr, collClub.LogReceivedBD, clubData.CreatorID, safeBox.BaoDiCount)
	if err != nil {
		glog.Warning("WriteReceivedSafeBoxLog(). ", err.Error())
		return
	} else {
		noticePlayerChangedScore(param.OperationClubID, senderID, curScore)
	}
}

type ReqCaptainAll struct {
	OperationClubID int32  `json:"curClubID"` // 操作人 所属俱乐部ID
	QueryUID        int64  `json:"qUID"`
	QueryName       string `json:"qName"`

	Date     int `json:"date"` // 指定日期
	PageSize int `json:"pageSize"`
	CurPage  int `json:"curPage"`
}

// 队长管理-战队成员
func onCaptainAll(senderID int64, req *clubProto.CS_QueryTotal, rsp *mateProto.JsonResponse) {
	type CaptainAllItem struct {
		UID int64 `json:"uid"`

		PlayerScore    string `json:"score"`
		GameScore      string `json:"gScore"`
		GameRoundCount int32  `json:"grCount"`

		Supervisor int64 `json:"upID"`
	}
	type RspCaptainMember struct {
		Item []CaptainAllItem `json:"item"`
	}

	rspData := RspCaptainMember{}
	rsp.Data = &rspData

	param := ReqCaptainAll{}
	err := json.Unmarshal([]byte(req.Data), &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.OperationClubID, senderID)
	if rsp.Status != 0 {
		return
	}

	//now := time.Now()
	//y, m, d := now.Date()
	//today_, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", y, m, d))

	if len(param.QueryName) > 0 {

		return
	}
	if param.QueryUID != 0 {
		allMemberArr := getAllMember(clubData)

		ofClubID := int32(0)
		for _, v := range allMemberArr {
			if v.ID == param.QueryUID {
				ofClubID = v.OfClubID
				break
			}
		}
		if ofClubID == 0 {
			return
		}

		rspData.Item = make([]CaptainAllItem, 1)
		rspData.Item[0].UID = param.QueryUID

		playerTotalMap_ := map[int64]dbCollectionDefine.DBClubPlayerTotal{}
		err = db.GetPlayerTotal(clubData.ClubID, []int64{param.QueryUID}, param.Date, &playerTotalMap_)
		if err != nil {
			glog.Warning(err.Error())
			rsp.Status = errorCodeDef.Err_Failed
			return
		}
		if v, ok := playerTotalMap_[param.QueryUID]; ok == true {
			rspData.Item[0].UID = param.QueryUID
			rspData.Item[0].GameScore = commonDef.ScoreToClient(v.GameScore)
			rspData.Item[0].GameRoundCount = v.GameCount
		}

		temp_, _ := db.GetPlayerClubScore(clubData.ClubID, param.QueryUID)
		rspData.Item[0].PlayerScore = commonDef.ScoreToClient(temp_)

		//tempClub, err := loadClubData(ofClubID)
		//if err != nil {
		//	glog.Warning(err.Error(), ",", ofClubID)
		//} else if tempClub.DirectSupervisor.ClubID > 0 {
		//	tempClub, err = loadClubData(tempClub.DirectSupervisor.ClubID)
		//	if err != nil {
		//		glog.Warning(err.Error(), ",", tempClub.ClubID)
		//	} else {
		//		rspData.Item[0].Supervisor = tempClub.CreatorID
		//	}
		//} else {
		//	rspData.Item[0].Supervisor = tempClub.CreatorID
		//}
		tempClub, err := loadClubData(ofClubID)
		if err != nil {
			glog.Warning(err.Error(), ",", ofClubID)
		} else {
			rspData.Item[0].Supervisor = tempClub.CreatorID
		}
		return
	}

	if param.CurPage > 100 || param.CurPage < 0 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}
	if param.PageSize > 10 || param.PageSize < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}
	allMemberArr := getAllMember(clubData)

	memberIDArr := make([]int64, 0, param.PageSize)
	for i := param.CurPage * param.PageSize; i < len(allMemberArr); i++ {
		memberIDArr = append(memberIDArr, allMemberArr[i].ID)
	}

	playerTotalMap_ := map[int64]dbCollectionDefine.DBClubPlayerTotal{}
	err = db.GetPlayerTotal(0, memberIDArr, param.Date, &playerTotalMap_)
	if err != nil {
		glog.Warning(err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return
	}

	rspData.Item = make([]CaptainAllItem, param.PageSize)
	indexItem := 0
	for i := param.CurPage * param.PageSize; i < len(allMemberArr) && indexItem < param.PageSize; i++ {
		rspData.Item[indexItem].UID = allMemberArr[i].ID
		if totalV, ok := playerTotalMap_[allMemberArr[i].ID]; ok == true {
			rspData.Item[indexItem].GameScore = commonDef.ScoreToClient(totalV.GameScore)
			rspData.Item[indexItem].GameRoundCount = totalV.GameCount
		}

		temp_, _ := db.GetPlayerClubScore(allMemberArr[i].OfClubID, allMemberArr[i].ID)
		rspData.Item[indexItem].PlayerScore = commonDef.ScoreToClient(temp_)

		//tempClub, err := loadClubData(allMemberArr[i].OfClubID)
		//if err != nil {
		//	glog.Warning(err.Error(), ",", allMemberArr[i].OfClubID)
		//} else if tempClub.DirectSupervisor.ClubID > 0 {
		//	tempClub, err = loadClubData(tempClub.DirectSupervisor.ClubID)
		//	if err != nil {
		//		glog.Warning(err.Error(), ",", tempClub.ClubID)
		//	} else {
		//		rspData.Item[indexItem].Supervisor = tempClub.CreatorID
		//	}
		//} else {
		//	rspData.Item[indexItem].Supervisor = tempClub.CreatorID
		//}
		tempClub, err := loadClubData(allMemberArr[i].OfClubID)
		if err != nil {
			glog.Warning(err.Error(), ",", allMemberArr[i].OfClubID)
		} else {
			rspData.Item[indexItem].Supervisor = tempClub.CreatorID
		}

		indexItem++
	}
	if indexItem < len(rspData.Item) {
		rspData.Item = rspData.Item[:indexItem]
	}
}

type ReqGameTotal struct {
	OperationClubID int32 `json:"curClubID"` // 操作人 所属俱乐部ID

	QueryClubID int32  `json:"qClubID"`
	QueryUID    int64  `json:"qUID"`
	QueryName   string `json:"qName"`

	Date     int `json:"date"`
	PageSize int `json:"pageSize"`
	CurPage  int `json:"curPage"`
}

// 个人数据-游戏统计
func onGameTotal(senderID int64, req *clubProto.CS_QueryTotal, rsp *mateProto.JsonResponse) {
	//type CaptainAllItem struct {
	//	ClubID int32 `json:"clubID"`
	//	UID    int64 `json:"uid"`
	//}
	//type RspCaptainMember struct {
	//	Item []CaptainAllItem `json:"item"`
	//}

	//rspData := RspCaptainMember{}
	//rsp.Data = &rspData

	param := ReqGameTotal{}
	err := json.Unmarshal([]byte(req.Data), &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.OperationClubID, senderID)
	if rsp.Status != 0 {
		return
	}

	if param.CurPage > 100 || param.CurPage < 0 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}
	if param.PageSize > 10 || param.PageSize < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}
	for _, v := range getAllMember(clubData) {
		if v.ID == param.QueryUID {
			param.QueryClubID = v.OfClubID
			break
		}
	}

	getArr := []interface{}{}
	err = db.GetPlayerTotalForGame(param.PageSize, param.CurPage, param.Date, param.QueryClubID, param.QueryUID, &getArr)
	if err != nil {
		glog.Warning(err.Error(), ",", param)
		rsp.Status = errorCodeDef.Err_Failed
		return
	}
	rsp.Data = getArr
}

type ReqPersonalGameTotal struct {
	OperationClubID int32 `json:"curClubID"` // 操作人 所属俱乐部ID

	QueryClubID int32  `json:"qClubID"`
	QueryUID    int64  `json:"qUID"`
	QueryName   string `json:"qName"`

	PageSize int `json:"pageSize"`
	CurPage  int `json:"curPage"`
}

// 个人数据-游戏统计
func onPersonalGameTotal(senderID int64, req *clubProto.CS_QueryTotal, rsp *mateProto.JsonResponse) {
	type PersonalGameTotalItem struct {
		Date     int    `json:"date"`
		RoundC   int32  `json:"roundC"`
		RoomCard string `json:"roomCard"`
		GongXian string `json:"gx"`
		JiangLi  string `json:"jl"`
	}
	//type RspCaptainMember struct {
	//	Item []CaptainAllItem `json:"item"`
	//}

	//rspData := RspCaptainMember{}
	//rsp.Data = &rspData

	param := ReqPersonalGameTotal{}
	err := json.Unmarshal([]byte(req.Data), &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.OperationClubID, senderID)
	if rsp.Status != 0 {
		return
	}

	if param.CurPage > 100 || param.CurPage < 0 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}
	if param.PageSize > 10 || param.PageSize < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}
	for _, v := range getAllMember(clubData) {
		if v.ID == param.QueryUID {
			param.QueryClubID = v.OfClubID
			break
		}
	}

	getArr := make([]PersonalGameTotalItem, 0, param.PageSize)
	err = db.GetPlayerTotalForClub(param.PageSize, param.CurPage, param.QueryClubID, param.QueryUID, func(item *dbCollectionDefine.DBClubPlayerTotal) {
		getArr = append(getArr, PersonalGameTotalItem{Date: item.Date, RoundC: item.GameCount})
		getArr[len(getArr)-1].GongXian = commonDef.ScoreToClient(item.XiaoHaoScore)
		getArr[len(getArr)-1].RoomCard = commonDef.ScoreToClient(item.HaoKa)
		for i := 0; i < len(item.ClubJiangLi); i++ {
			if item.ClubJiangLi[i].ClubID == param.OperationClubID {
				getArr[len(getArr)-1].JiangLi = commonDef.ScoreToClient(item.ClubJiangLi[i].JiangLi)
				break
			}
		}
	})
	if err != nil {
		glog.Warning(err.Error(), ",", param)
		rsp.Status = errorCodeDef.Err_Failed
		return
	}
	rsp.Data = getArr
}

type ReqQueryTongZhuo struct {
	OperationClubID int32 `json:"curClubID"` // 操作人 所属俱乐部ID

	QueryDate   int    `json:"date"`
	QueryClubID int32  `json:"qClubID"`
	QueryUID    int64  `json:"qUID"`
	QueryName   string `json:"qName"`

	PageSize int `json:"pageSize"`
	CurPage  int `json:"curPage"`
}

// 个人数据-同桌统计
func onQueryTongZhuo(mzIDStr string, senderID int64, req *clubProto.CS_QueryTotal, rsp *mateProto.JsonResponse) {
	//type RspCaptainMember struct {
	//	Item []CaptainAllItem `json:"item"`
	//}

	//rspData := RspCaptainMember{}
	//rsp.Data = &rspData

	param := ReqQueryTongZhuo{}
	err := json.Unmarshal([]byte(req.Data), &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return
	}

	//var clubData *collClub.DBClubData
	rsp.Status, _ = IsClubCreator(param.OperationClubID, senderID)
	if rsp.Status != 0 {
		return
	}

	if param.CurPage > 100 || param.CurPage < 0 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}
	if param.PageSize > 10 || param.PageSize < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return
	}

	mzClubID, _ := strconv.Atoi(mzIDStr)
	getArr := make([]interface{}, 0, param.PageSize)
	err = db.GetTongZhuo(param.PageSize, param.CurPage, param.QueryDate, int32(mzClubID), param.QueryUID, &getArr)
	if err != nil {
		glog.Warning(err.Error(), ",", param)
		rsp.Status = errorCodeDef.Err_Failed
		return
	}
	rsp.Data = getArr
}

func onGetClubRobotCfg(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_GetRobotCfg{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = IsClubCreator(param.ClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	if clubData.DirectSupervisor.ClubID > 0 {
		rsp.Status = errorCodeDef.ErrFindSuperiorExist
		return rsp
	}

	arr := []dbCollectionDefine.DBRobotClubPlayConfig{}
	err = db.GetClubRobotCfg(clubData.ClubID, &arr)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return rsp
		}
		glog.Warning(err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	rsp.Data = arr
	return rsp
}

func onGetClubRobotItemCfg(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_GetRobotItemCfg{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = IsClubCreator(param.ClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	if clubData.DirectSupervisor.ClubID > 0 {
		rsp.Status = errorCodeDef.ErrFindSuperiorExist
		return rsp
	}

	arr := []dbCollectionDefine.DBRobotSingle{}
	err = db.GetRobotItemCfg(clubData.ClubID, param.TargetUID, param.PageSize, param.CurPage, &arr)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return rsp
		}
		glog.Warning(err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	memberIDArr := make([]int64, 0, len(arr))
	for i, _ := range arr {
		memberIDArr = append(memberIDArr, arr[i].ID)
	}

	playerTotalMap_ := map[int64]dbCollectionDefine.DBClubPlayerTotal{}
	err = db.GetPlayerTotal(0, memberIDArr, param.Date, &playerTotalMap_)
	if err != nil {
		glog.Warning(err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	item_ := make([]CaptainMemberItem, len(arr))
	itemIndex := 0
	for i := 0; i < len(arr); i++ {
		item_[itemIndex].UID = arr[i].ID
		item_[itemIndex].ClubID = arr[i].ClubID

		if v, ok := playerTotalMap_[arr[i].ID]; ok == true {
			item_[itemIndex].TodayGX = commonDef.ScoreToClient(v.XiaoHaoScore)
			item_[itemIndex].TodaySY = commonDef.ScoreToClient(v.GongXian)

			item_[itemIndex].GameScore = commonDef.ScoreToClient(v.GameScore)
			item_[itemIndex].GameRoundCount = v.GameCount
		}

		temp_, _ := db.GetPlayerClubScore(arr[i].ClubID, arr[i].ID)
		item_[itemIndex].PlayerScore = commonDef.ScoreToClient(temp_)

		//if item_[itemIndex].UID == clubData.CreatorID {
		//	item_[itemIndex].Role = 2
		//} else if arr[i].IsAdmin {
		//	item_[itemIndex].Role = 1
		//}
		//if arr[i].Frozen.IsFrozen {
		//	item_[itemIndex].Status = 1
		//}
		//item_[itemIndex].IsStop3 = arr[i].IsNo3
		//item_[itemIndex].IsStop4 = arr[i].IsNo4
		//item_[itemIndex].Remark = arr[i].Remark

		itemIndex++
	}

	for i, _ := range arr {
		item_[i].Robot = arr[i].ClubPlayItem
	}

	rsp.Data = item_
	return rsp
}
