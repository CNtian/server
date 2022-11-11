package clubEvent

import (
	"encoding/json"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/mongo"
	"sync"
	"time"
	"vvService/appClub/db"
	"vvService/appClub/localConfig"
	clubProto "vvService/appClub/protoDefine"
	"vvService/appClub/wrapMQ"
	commonDB "vvService/commonPackge/db"
	"vvService/commonPackge/mateProto"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
)

type PlayerBaseInfo struct {
	Nick    string
	HeadURL string
}

type mzMemberOnline struct {
	uid            []int64
	count          int
	lastUpdateTime time.Time
}

var (
	mutexClubDataMap sync.RWMutex
	clubDataMap      = make(map[int32]*collClub.DBClubData)

	mutexPlayerMap sync.RWMutex
	playerMap      = make(map[int64]PlayerBaseInfo)

	mzClubPlayerOnlineMap = make(map[int32]*mzMemberOnline) // key:俱乐部盟ID value:玩家id
	mzMemberMap           = make(map[int64]int32)           // key:玩家ID value:盟主ID
)

func ReloadClubData(clubID int32) (*collClub.DBClubData, error) {
	var (
		clubData     *collClub.DBClubData
		err          error
		ok           bool
		tempClubData *collClub.DBClubData
	)

	clubData, ok = clubDataMap[clubID]
	if !ok {
		clubData, err = db.LoadClub(clubID)
		if err != nil {
			if err != mongo.ErrNoDocuments {
				glog.Warning("db.LoadClub() err. err:=", err.Error(), ", clubID:=", clubID)
			}
			return nil, err
		}
		clubDataMap[clubID] = clubData

		clubData.MemberMap = make(map[int64]*collClub.DBClubMember)
		clubData.AdminMemberMap = make(map[int64]struct{})
		for i, v := range clubData.MemberArr {
			v.OfClubID = clubData.ClubID
			clubData.MemberMap[v.ID] = clubData.MemberArr[i]

			if clubData.MemberArr[i].IsAdmin == true {
				clubData.AdminMemberMap[v.ID] = struct{}{}
			}
		}

		clubData.GameIDArr = make([]int32, 0, 4)
		tempGameIDMap := make(map[int32]int32)
		clubData.PlayIDMap = make(map[int64]*collClub.DBClubPlay)
		clubData.SimplePlayIDArr = make([]*collClub.DBClubPlay, 0, 60)
		for i, v := range clubData.PlayArr {
			tempClubRule := collClub.DBClubRule{}
			json.Unmarshal([]byte(v.ClubCfg), &tempClubRule)
			clubData.PlayArr[i].ClubRule = &tempClubRule
			clubData.PlayArr[i].ClubRule.GetTextVale()
			clubData.PlayIDMap[v.ID] = clubData.PlayArr[i]

			if v.IsDelete == false && v.IsHide == false {
				if _, ok := tempGameIDMap[v.GameID]; ok == false {
					tempGameIDMap[v.GameID] = v.GameID
					clubData.GameIDArr = append(clubData.GameIDArr, v.GameID)
				}

				temp_clubPlay := *v
				temp_clubPlay.TableCfg = ""
				temp_clubPlay.ClubCfg = ""
				temp_clubPlay.ClubRule = nil
				clubData.SimplePlayIDArr = append(clubData.SimplePlayIDArr, &temp_clubPlay)
			}
		}

		clubData.SubordinatesMap = make(map[int32]int32)
		for _, v := range clubData.Subordinates {
			clubData.SubordinatesMap[v] = v
		}
	}
	tempClubData = clubData

	if clubData.DirectSupervisor.ClubID > 0 {
		listIndex := int32(0)
		for tempClubData.DirectSupervisor.ClubID > 0 {
			clubID := tempClubData.DirectSupervisor.ClubID
			tempClubData, err = ReloadClubData(tempClubData.DirectSupervisor.ClubID)
			if err != nil {
				glog.Warning("reloadClubData() err.clubID:=", clubID,
					",err:=", err.Error())
				break
			}
			listIndex += 1
			if tempClubData.DirectSupervisor.ClubID < 1 {
				clubData.MengZhuClubID = tempClubData.ClubID
				err = db.WriteClubMengZhuID(clubData.ClubID, clubData.MengZhuClubID)
				if err != nil {
					glog.Warning("WriteClubMengZhuID :=", clubData.ClubID, ",err:=", err.Error())
				}
				break
			}
		}
		clubData.ListIndex = listIndex
	}

	return clubData, nil
}

func loadClubData(clubID int32) (*collClub.DBClubData, error) {
	var err error
	mutexClubDataMap.RLock()
	v, ok := clubDataMap[clubID]
	mutexClubDataMap.RUnlock()
	if ok == true {
		return v, nil
	}

	mutexClubDataMap.Lock()
	defer mutexClubDataMap.Unlock()
	v, err = ReloadClubData(clubID)
	if v != nil && err == nil {
		if x, _ := db.CheckNewMail(v.ClubID); x > 0 {
			v.IsHadNewMail = true
		} else {
			v.IsHadNewMail = false
		}
		//if v.DirectSupervisor.ClubID == 0 {
		//	v.IsKickOutMember = true
		//	v.IsKickOutLeague = true
		//}
	}
	if v == nil {
		glog.Warning("clubID :=", clubID)
		return v, err
	}
	if v.MengZhuClubID < 1 {
		v.Activity = &collClub.DBClubActivity{}
		err = db.GetMengZhuActivity(v.ClubID, v.Activity)
		if err != nil {
			if err != mongo.ErrNoDocuments {
				glog.Warning("GetMengZhuActivity() club_id:=", v.ClubID, ",err := ", err.Error())
			}
			v.Activity = nil
		}
		err = nil
	}

	if v.MengZhuClubID < 1 {
		arr := []dbCollectionDefine.DBRobotClubPlayConfig{}
		err = db.GetClubRobotCfg(v.ClubID, &arr)
		if err != nil {
			glog.Warning(err.Error())
		}
		for i, _ := range arr {
			playRobotRule, ok := v.PlayIDMap[arr[i].ClubPlayID]
			if ok == false {
				continue
			}
			playRobotRule.RobotJoinPlaying = arr[i].ClubPlayItem.JoinPlayingTable
			playRobotRule.RobotJoinReady = arr[i].ClubPlayItem.JoinReadyTable
			playRobotRule.RobotInviteTimer = arr[i].ClubPlayItem.CheckTime
			playRobotRule.RobotOpen = arr[i].ClubPlayItem.Open
		}
	}

	return v, err
}

func delLocalClubData(clubID int32) {
	mutexClubDataMap.Lock()
	delete(clubDataMap, clubID)
	mutexClubDataMap.Unlock()
}

func delAllClubData(noticeDB bool) {
	mutexClubDataMap.Lock()
	clubDataMap = make(map[int32]*collClub.DBClubData)
	mutexClubDataMap.Unlock()

	if noticeDB == true {
		noticeDBServiceClubChanged(0)
	}
}

func clubMGRLaunch(msg *mateProto.MessageMaTe) {
	msg.MZID = localConfig.GetConfig().ID
	err := wrapMQ.SendToSource(msg, nil)
	if err != nil {
		glog.Warning("clubMGRLaunch(). ", err.Error())
	}
}

func LoadPlayerNick_Name(uid int64) *PlayerBaseInfo {
	mutexPlayerMap.RLock()
	v, ok := playerMap[uid]
	mutexPlayerMap.RUnlock()
	if ok == true {
		return &v
	}

	mutexPlayerMap.Lock()
	if len(playerMap) > 10000 {
		playerMap = make(map[int64]PlayerBaseInfo)
	}

	temp := PlayerBaseInfo{}
	temp.HeadURL, temp.Nick = loadPlayerDataFromRedis(uid)
	playerMap[uid] = temp
	mutexPlayerMap.Unlock()
	return &temp

}

func loadPlayerDataFromRedis(uid int64) (string, string) {
	headDomain, headPath, nick, err := db.LoadPlayerHead(uid)
	if err != nil {
		glog.Warning("loadPlayerDataFromRedis() err.", err.Error())
	}
	return headDomain + headPath, nick
}

func onUpdatePlayerInfo(msg *mateProto.MessageMaTe) {
	msgBody := mateProto.SS_PlayerInfoUpdate{}

	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		return
	}

	mutexPlayerMap.Lock()
	delete(playerMap, msgBody.UID)
	mutexPlayerMap.Unlock()
}

func noticeClubNewMail(mailRecvClubID int32) {

	clubData, err := loadClubData(mailRecvClubID)
	if err != nil {
		glog.Warning("noticeNewMail() err. err:=", err.Error())
		return
	}
	clubData.IsHadNewMail = true

	noticePlayerArr := make([]int64, 0, len(clubData.AdminMemberMap)+1)
	for k, _ := range clubData.AdminMemberMap {
		noticePlayerArr = append(noticePlayerArr, k)
	}
	noticePlayerArr = append(noticePlayerArr, clubData.CreatorID)

	msg := mateProto.MessageMaTe{MessageID: clubProto.ID_NewClubMail}
	msg.Data, _ = json.Marshal(&clubProto.CS_NewClubMail{ClubID: mailRecvClubID})

	go func() {
		var source string
		for _, v := range noticePlayerArr {
			source, _ = commonDB.LoadGatewayServiceID(db.PersonalRedisClient, v)
			if len(source) < 1 {
				continue
			}
			msg.SenderID = v
			err = wrapMQ.PublishProto(source, &msg)
			if err != nil {
				glog.Warning("uid:=", msg.SenderID,
					" ,wrapMQ.PublishData err. err:=", err.Error(),
					",msg ID:=", msg.MessageID, " ,data:=", len(msg.Data))
			}
		}
	}()
}

func NoticePlayerNewMail(uid int64) {

	//commonDB.SetPlayerReadEmailStatus(db.PersonalRedisClient, uid, 1)

	gateWayID, _ := commonDB.LoadGatewayServiceID(db.PersonalRedisClient, uid)
	if len(gateWayID) < 1 {
		return
	}

	noticePlayerNewEmail := mateProto.MessageMaTe{To: gateWayID, SenderID: uid, MessageID: clubProto.ID_NoticePlayerNewEmail}
	wrapMQ.PublishProto(gateWayID, &noticePlayerNewEmail)
}

func updatePlayerOnline(mzClubID int32, uid int64, isOnline bool) {
	if len(mzClubPlayerOnlineMap) > 1000 {
		now := time.Now()
		for k, v := range mzClubPlayerOnlineMap {
			if v.count < 1 {
				delete(mzClubPlayerOnlineMap, k)
			} else if now.Sub(v.lastUpdateTime) > time.Hour*24 {
				delete(mzClubPlayerOnlineMap, k)
			}
		}
	}
	memberData, ok := mzClubPlayerOnlineMap[mzClubID]
	if ok == false {
		memberData = &mzMemberOnline{}
		mzClubPlayerOnlineMap[mzClubID] = memberData
	}
	if memberData.count < 1 {
		memberData.uid = make([]int64, 0, 100)
	}
	memberData.lastUpdateTime = time.Now()

	if isOnline == true {
		for i := 0; i < len(memberData.uid); i++ {
			if memberData.uid[i] < 1 {
				memberData.uid[i] = uid
				memberData.count += 1
				mzMemberMap[uid] = mzClubID
				break
			}
		}
	} else {
		for i := 0; i < len(memberData.uid); i++ {
			if memberData.uid[i] == uid {
				memberData.uid[i] = 0
				memberData.count -= 1
				delete(mzMemberMap, uid)
				break
			}
		}
	}
}
