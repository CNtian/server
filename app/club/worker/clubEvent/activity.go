package clubEvent

import (
	"encoding/json"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/mongo"
	"sort"
	"strconv"
	"time"
	"unsafe"
	"vvService/appClub/db"
	clubProto "vvService/appClub/protoDefine"
	commonDef "vvService/commonPackge"
	"vvService/commonPackge/errorCodeDef"
	"vvService/commonPackge/mateProto"
	collClub "vvService/dbCollectionDefine/club"
)

type ActivityData struct {
	UID         int64  `json:"uid"`
	Score       int64  `json:"-"`
	ScoreString string `json:"score"`
	RoundCount  int    `json:"rounds"`

	R_Index int `json:"rIndex"`
	S_Index int `json:"sIndex"`
}

type ActivityRoundSort []*ActivityData

func (s ActivityRoundSort) Len() int { return len(s) }
func (s ActivityRoundSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
	s[i].R_Index, s[j].R_Index = i, j
}
func (s ActivityRoundSort) Less(i, j int) bool {
	if s[i].RoundCount > s[j].RoundCount {
		return true
	}
	if s[i].RoundCount == s[j].RoundCount {
		if s[i].Score > s[j].Score {
			return true
		}
	}
	return false
}

type ActivityScoreSort []*ActivityData

func (s ActivityScoreSort) Len() int { return len(s) }
func (s ActivityScoreSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
	s[i].S_Index, s[j].S_Index = i, j
}
func (s ActivityScoreSort) Less(i, j int) bool {
	return s[i].Score > s[j].Score
}

// 正在进行的
var (
	_activityMap      = make(map[int64]*ActivityData) // key:uid value:sort
	_activityRoundArr ActivityRoundSort
	_activityScoreArr ActivityScoreSort
	_lastSortTime     time.Time
	_isChanged        bool
	_activityST       int64
	_activityRule     *collClub.DBClubActivity

	_blackListChanged = true
	_blackListMap     = make(map[int64]*collClub.DBBlacklistItem) // key:uid
	_blackListArr     = []*collClub.DBBlacklistItem{}
)

// 上次的
var ()

const (
	typeRound = 1
	typeScore = 2
)

func onGetClubActivity(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_GetActivity{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	cv, err := loadClubData(req.ClubID)
	if err != nil {
		glog.Warning(err.Error(), ",clubID:=", req.ClubID)
		return rsp
	}
	if _, ok := cv.MemberMap[msg.SenderID]; ok == false {
		rsp.Status = errorCodeDef.ErrClubNotMember
		return rsp
	}

	mzClubData := cv
	if cv.MengZhuClubID > 0 {
		mzClubData, err = loadClubData(cv.MengZhuClubID)
		if err != nil {
			glog.Warning(err.Error(), ",clubID:=", cv.MengZhuClubID)
			return rsp
		}
	}

	//if mzClubData.Activity != nil {
	//	tt := time.Now()
	//	if tt.Sub(mzClubData.Activity.Rule.MatchE) > 0 {
	//		return rsp
	//	}
	//}

	rsp.Data = mzClubData.Activity

	return rsp
}

func onActivityGameData(msg *mateProto.MessageMaTe) {
	req := mateProto.SS_ActivityGameData{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		glog.Warning("onActivityGameData. ", err.Error())
		return
	}

	_, ok := _activityIng.Load(req.AcID)
	if ok == false {
		// 前开启后,数据很快过来,误判,导致 提前结束活动
		//noticeDBClubActivityStatus(v.(*collClub.DBClubActivity), false)
		return
	}

	if len(req.PlayerGameScore) < 1 {
		glog.Warning("len(req.PlayerGameScore) < 1")
		return
	}

	for _, v := range req.PlayerGameScore {
		t, ok := _activityMap[v.UID]
		if ok == false {
			t = &ActivityData{UID: v.UID}
		}
		t.Score += v.Score
		t.RoundCount += 1
		_activityMap[v.UID] = t
	}
	_isChanged = true
}

// 新活动开启,数据恢复
func onActivityStatus(msg *mateProto.MessageMaTe) {
	req := mateProto.SS_NoticeClubActivity{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		glog.Warning("onActivityStatus()   ", string(msg.Data), ",", err.Error())
		return
	}

	glog.Warning("onActivityStatus()   ", req.ClubID, ",", req.IsOpen)

	if req.IsOpen == true {
		v, ok := _activityWaitOpen.Load(req.AcID)
		if ok == false {
			glog.Warning("not find activity ID :=", req.AcID.Hex())
			return
		}
		_v := v.(*collClub.DBClubActivity)

		_activityMap = make(map[int64]*ActivityData)
		_activityRoundArr = nil
		_activityScoreArr = nil
		_lastSortTime = time.Time{}
		_isChanged = false
		_activityRule = _v

		_activityWaitOpen.Delete(req.AcID)
		_activityST = _v.Rule.MatchS.Unix()
		_activityIng.Store(req.AcID, _v)

		err = db.DelMengZHuActivityAward(req.ClubID)
		if err != nil {
			glog.Warning("DelMengZHuActivityAward() err.", err.Error())
		}

		err = db.WriteOrDelClubActivityPlayer(req.ClubID, "")
		if err != nil {
			glog.Warning("WriteOrDelClubActivityPlayer() err.", err.Error())
		}
		return
	}

	// 活动结束
	v, ok := _activityIng.Load(req.AcID)
	if ok == false {
		glog.Warning("not find activity ID :=", req.AcID.Hex())
		return
	}
	_v := v.(*collClub.DBClubActivity)
	_activityIng.Delete(req.AcID)

	mzClubData, err := loadClubData(_v.ClubID)
	if err != nil {
		glog.Warning(err.Error(), ",clubID:=", _v.ClubID)
		return
	}

	tt := time.Now()
	sortAc(&tt)

	_t := collClub.DBActivityAwardList{MZClubID: _v.ClubID, CreateTime: time.Now()}
	for i, v := range _activityScoreArr {
		if i >= len(_v.Rule.ScoreAward) {
			break
		}
		_t.IsGet = mzClubData.CreatorID == v.UID
		_t.UID = v.UID
		_t.Category, _t.Rank = typeScore, i
		_t.AwardValue = _v.Rule.ScoreAward[i]

		err = db.PutActivityAwardList(&_t)
		if err != nil {
			glog.Warning("PutActivityAward err.err:=", err.Error(), ",", _t)
		}
	}

	for i, v := range _activityRoundArr {
		if i >= len(_v.Rule.RoundAward) {
			break
		}
		_t.IsGet = mzClubData.CreatorID == v.UID
		_t.UID = v.UID
		_t.Category, _t.Rank = typeRound, i
		_t.AwardValue = _v.Rule.RoundAward[i]

		err = db.PutActivityAwardList(&_t)
		if err != nil {
			glog.Warning("PutActivityAward err.err:=", err.Error(), ",", _t)
		}
	}

	// 存入 redis 活动记录
	acJson, err := json.Marshal(_activityMap)
	if err != nil {
		glog.Warning("Marshal(_activityMap) err.err:=", err.Error())
	} else {
		err = db.WriteOrDelClubActivityPlayer(_v.ClubID, *(*string)(unsafe.Pointer(&acJson)))
		if err != nil {
			glog.Warning("WriteOrDelClubActivityPlayer() err.", err.Error())
		}
	}
}

func onActivitySort(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_GetActivitySort{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	cv, err := loadClubData(req.ClubID)
	if err != nil {
		glog.Warning(err.Error(), ",clubID:=", req.ClubID)
		return rsp
	}
	if _, ok := cv.MemberMap[msg.SenderID]; ok == false {
		rsp.Status = errorCodeDef.ErrClubNotMember
		return rsp
	}
	mzClubData := cv
	if cv.MengZhuClubID > 0 {
		mzClubData, err = loadClubData(cv.MengZhuClubID)
		if err != nil {
			glog.Warning(err.Error(), ",clubID:=", cv.MengZhuClubID)
			rsp.Status = errorCodeDef.Err_Failed
			return rsp
		}
	}

	tt := time.Now()
	if tt.Sub(_lastSortTime) > time.Second && _isChanged == true {
		sortAc(&tt)
		_isChanged = false
	}

	rspBody := &clubProto.SC_GetActivitySort{AcStartTime: _activityST, Self: -1, Value: 0}
	rsp.Data = rspBody

	if req.CateGory == typeRound {
		sIndex := req.CurPage * req.PageSize
		if sIndex >= len(_activityRoundArr) || sIndex < 0 {
			return rsp
		}
		t := sIndex + req.PageSize
		if t >= len(_activityRoundArr) {
			rspBody.Item = _activityRoundArr[sIndex:]
		} else {
			rspBody.Item = _activityRoundArr[sIndex : sIndex+req.PageSize]
		}
		if f, ok := _activityMap[msg.SenderID]; ok == true {
			rspBody.Self = f.R_Index
			rspBody.Value = f.RoundCount
		}
	} else {
		sIndex := req.CurPage * req.PageSize
		if sIndex >= len(_activityScoreArr) || sIndex < 0 {
			return rsp
		}
		t := sIndex + req.PageSize
		if t >= len(_activityScoreArr) {
			rspBody.Item = _activityScoreArr[sIndex:]
		} else {
			rspBody.Item = _activityScoreArr[sIndex : sIndex+req.PageSize]
		}
		for _, v := range rspBody.Item.(ActivityScoreSort) {
			v.ScoreString = commonDef.ScoreToClient(v.Score)
		}

		if f, ok := _activityMap[msg.SenderID]; ok == true {
			rspBody.Self = f.S_Index
			rspBody.Value = commonDef.ScoreToClient(f.Score)
		}
	}

	if mzClubData.Activity != nil && _activityRule != nil {
		if mzClubData.Activity.Rule.ActivityID != _activityRule.Rule.ActivityID {
			rspBody.LastAcRule = _activityRule
		}
	}

	return rsp
}

func sortAc(tt *time.Time) {
	_activityRoundArr = make(ActivityRoundSort, 0, len(_activityMap))
	_activityScoreArr = make(ActivityScoreSort, 0, len(_activityMap))

	i := 0
	for _, v := range _activityMap {
		if _, ok := _blackListMap[v.UID]; ok == true {
			continue
		}
		v.R_Index, v.S_Index = i, i
		_activityRoundArr = append(_activityRoundArr, v)
		_activityScoreArr = append(_activityScoreArr, v)
		i++
	}

	sort.Sort(_activityScoreArr)

	//for _, v := range _activityScoreArr {
	//	fmt.Println("uid:=", v.UID, "  score:=", v.Score, "   sIndex:=", v.S_Index)
	//}

	sort.Sort(_activityRoundArr)

	_lastSortTime = *tt
}

func onReceiveActivityAward(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_GetActivityAward{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	cv, err := loadClubData(req.ClubID)
	if err != nil {
		glog.Warning(err.Error(), ",clubID:=", req.ClubID)
		return rsp
	}
	if _, ok := cv.MemberMap[msg.SenderID]; ok == false {
		rsp.Status = errorCodeDef.ErrClubNotMember
		return rsp
	}

	mzClubData := cv
	if cv.MengZhuClubID > 0 {
		mzClubData, err = loadClubData(cv.MengZhuClubID)
		if err != nil {
			glog.Warning(err.Error(), ",clubID:=", cv.MengZhuClubID)
			return rsp
		}
	}

	award_ := collClub.DBActivityAwardList{}
	err = db.GetMyActivityAward(msg.SenderID, mzClubData.ClubID, req.Category, &award_)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.Err_Not_Find_Item
			return rsp
		}
		glog.Warning("GetActivityAward() err.", err.Error(), ",uid:=", msg.SenderID)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	if award_.IsGet == true {
		rsp.Status = errorCodeDef.Err_Not_Find_Item
		return rsp
	}

	awardValue, _ := strconv.Atoi(award_.AwardValue)

	log := db.UpdateMemberScoreParam{
		A_UID:      mzClubData.CreatorID,
		A_ClubName: mzClubData.Name,
		A_ClubID:   mzClubData.ClubID,
	}
	log.A_Nick = LoadPlayerNick_Name(mzClubData.CreatorID).Nick

	if _, ok := mzClubData.MemberMap[msg.SenderID]; ok == true {
		log.OperationRelation = db.OR__MengZhu_To_ChengYun
	} else {
		log.OperationRelation = db.OR__MengZhu_To_XiaJiChengYun
	}

	log.B_ClubID = req.ClubID
	log.B_ClubName = cv.Name
	log.B_UID = msg.SenderID
	log.B_Nick = LoadPlayerNick_Name(msg.SenderID).Nick
	log.Value = int64(awardValue * commonDef.SR)

	rsp.Status, err = db.ActivityAward(&log, req.Category, &struct {
		Rank     int `json:"rank" bson:"rank"`
		Category int `json:"caty" bson:"caty"`
	}{award_.Rank, award_.Category})
	if err != nil {
		glog.Warning("db.ActivityAward() ", err.Error())
		rsp.Status = errorCodeDef.Err_Failed
	} else {
		noticePlayerChangedScore(log.A_ClubID, log.A_UID, log.Retrun_A_Score)
		noticePlayerChangedScore(log.B_ClubID, log.B_UID, log.Retrun_B_Score)
	}

	return rsp
}

func onGetActivityAwardList(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_GetActivityAwardList{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	cv, err := loadClubData(req.ClubID)
	if err != nil {
		glog.Warning(err.Error(), ",clubID:=", req.ClubID)
		return rsp
	}
	if _, ok := cv.MemberMap[msg.SenderID]; ok == false {
		rsp.Status = errorCodeDef.ErrClubNotMember
		return rsp
	}

	mzClubData := cv
	if cv.MengZhuClubID > 0 {
		mzClubData, err = loadClubData(cv.MengZhuClubID)
		if err != nil {
			glog.Warning(err.Error(), ",clubID:=", cv.MengZhuClubID)
			return rsp
		}
	}

	awardArr := []collClub.DBActivityAwardList{}
	err = db.GetActivityAwardList(mzClubData.ClubID, &awardArr)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			glog.Warning("GetActivityAwardList()", err.Error())
			rsp.Status = errorCodeDef.Err_Failed
		}
		return rsp
	}

	rsp.Data = awardArr
	return rsp
}

func onActivityBlackList(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_ActivityBlackList{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	cv, err := loadClubData(req.ClubID)
	if err != nil {
		glog.Warning(err.Error(), ",clubID:=", req.ClubID)
		return rsp
	}
	if cv.MengZhuClubID > 0 {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	switch req.SubID {
	case 1:
		rsp.Status = blackListUpdateOne(cv, req.Param)
	case 2:
		rsp.Status = blackListDeleteOne(req.Param)
	case 3:
		blackListGetAll()
		rsp.Data = _blackListArr
	default:

	}

	return rsp
}

func blackListUpdateOne(clubData *collClub.DBClubData, data []byte) int32 {
	req := clubProto.BlackListUpdateOne{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return errorCodeDef.Err_Param
	}

	var uidClubData *collClub.DBClubData
	if _, ok := clubData.MemberMap[req.UID]; ok == true {
		uidClubData = clubData
	} else {
		for _, v := range clubData.Subordinates {
			cv, err := loadClubData(v)
			if err != nil {
				glog.Warning("clubID :=", v, ",", err.Error())
				continue
			}
			if _, ok := cv.MemberMap[req.UID]; ok == true {
				uidClubData = cv
				break
			}
		}
	}
	if uidClubData == nil {
		return errorCodeDef.Err_Not_Find_Item
	}

	blItem := collClub.DBBlacklistItem{ClubID: uidClubData.ClubID,
		Uid: req.UID, CreateTime: time.Now(),
		ClubName: clubData.Name}
	p := LoadPlayerNick_Name(req.UID)
	blItem.UidName, blItem.HeadURL = p.Nick, p.HeadURL

	_blackListMap[req.UID] = &blItem
	_blackListChanged = true

	return 0
}

func blackListDeleteOne(data []byte) int32 {
	req := clubProto.BlackListUpdateOne{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return errorCodeDef.Err_Param
	}

	delete(_blackListMap, req.UID)
	_blackListChanged = true

	return 0
}

func blackListGetAll() {
	if _blackListChanged == false {
		return
	}
	_blackListArr = make([]*collClub.DBBlacklistItem, 0, len(_blackListMap))
	for _, v := range _blackListMap {
		_blackListArr = append(_blackListArr, v)
	}
	_blackListChanged = false
}
