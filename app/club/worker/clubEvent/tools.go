package clubEvent

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/golang/glog"
	"time"
	"vvService/appClub/db"
	clubProto "vvService/appClub/protoDefine"
	"vvService/commonPackge/errorCodeDef"
	collClub "vvService/dbCollectionDefine/club"
)

func stocktaking(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {

	rspCode, clubData := IsClubCreator(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}
	if clubData.IsOpen == true {
		return errorCodeDef.ErrNotClose, nil
	}
	if _curTableCount > 0 {
		return errorCodeDef.ErrTableExist, nil
	}

	now__ := time.Now()
	curHour := now__.Hour()
	// [2,6]点
	if curHour < 2 || curHour > 6 {
		return errorCodeDef.ErrNotInTime, nil
	}

	lastTime, oErr := db.GetLastStocktaking(clubData.ClubID)
	if oErr != nil {
		if oErr != redis.Nil {
			return rspCode, oErr
		}
		lastTime = 0
	}
	// 一天只能一次
	if lastTime != 0 {
		lastT := time.Unix(lastTime, 0)
		if lastT.Day() == now__.Day() {
			return errorCodeDef.ErrClubRepeatOperation, nil
		}
	}

	oErr = db.WriteLastStocktaking(clubData.ClubID, now__.Unix())
	if oErr != nil {
		glog.Warning("WriteLastStocktaking ", clubData.ClubID, "  ", oErr.Error())
	}

	// 该俱乐部下面的所有子俱乐部 更改为盘点
	allClubID_ := clubData.Subordinates
	allClubID_ = append(allClubID_, clubData.ClubID)
	{
		for _, v := range allClubID_ {
			var club_ *collClub.DBClubData
			club_, oErr = loadClubData(v)
			if oErr != nil {
				glog.Warning("WriteLastStocktaking ", clubData.ClubID, "  ", oErr.Error())
				continue
			}
			club_.IsStocking = true
		}
		rspCode, oErr = db.SetClubStocking(allClubID_, true)
		if oErr != nil {
			glog.Warning("WriteLastStocktaking ", clubData.ClubID, "  ", oErr.Error())
		}
		if rspCode != 0 {
			glog.Warning("WriteLastStocktaking ", clubData.ClubID, "  ", rspCode)
		}
	}

	go func() {
		var err error
		time.Sleep(time.Minute * 1)
		defer func() {
			db.SetClubStocking(allClubID_, false)

			for _, v := range allClubID_ {
				var club_ *collClub.DBClubData
				club_, err = loadClubData(v)
				if oErr != nil {
					glog.Warning(v, ",", err.Error())
					continue
				}
				club_.IsStocking = false
			}

			if err != nil {
				glog.Error("stocktaking-------", clubData.ClubID, "     ", err.Error())
				NoticeClubStockingFinish(clubData.ClubID, clubData.CreatorID, errorCodeDef.Err_Failed)
			} else {
				NoticeClubStockingFinish(clubData.ClubID, clubData.CreatorID, 0)
			}
		}()

		// 统计所有俱乐部成员的分
		allClubIDMap := make(map[int32]*db.StocktakingInfo)
		for _, v := range allClubID_ {
			var club_ *collClub.DBClubData
			club_, err = loadClubData(v)
			if err != nil {
				return
			}
			allMemberArr := make([]int64, len(club_.MemberArr))
			for i, _ := range club_.MemberArr {
				allMemberArr[i] = club_.MemberArr[i].ID
			}

			var gte0, less0 int64
			gte0, less0, err = db.GetMemberScore(v, allMemberArr)
			if err != nil {
				return
			}

			allClubIDMap[v] = &db.StocktakingInfo{MemberTotalScore: gte0, MemberTotalUnuseScore: less0}
		}

		// 按树形统计
		_, _, err = recursionTotal(param.OperationClubID, allClubIDMap)
		if err != nil {
			return
		}

		//for k, v := range allClubIDMap {
		//	glog.Warning("clubID:=", k, " score:=", v.AllSubordinateScore, "    unuseScore:=", v.AllSubordinateUnuseScore)
		//}

		err = db.SetClubTotalScoreUnuseScore(allClubIDMap)
	}()

	return 0, nil
}

// ():正分,负分
func recursionTotal(clubID int32, vMap map[int32]*db.StocktakingInfo) (int64, int64, error) {
	club, err := loadClubData(clubID)
	if err != nil {
		return 0, 0, err
	}

	var (
		allSubordinateScore      int64
		allSubordinateUnuseScore int64
	)
	for _, v := range club.DirectSubordinate {
		totalScore_, totalUnuseScore_, err := recursionTotal(v.ClubID, vMap)
		if err != nil {
			return 0, 0, err
		}
		allSubordinateScore += totalScore_
		allSubordinateUnuseScore += totalUnuseScore_
	}
	v, ok := vMap[clubID]
	if ok == false {
		return 0, 0, fmt.Errorf("not find clubID %d", clubID)
	}
	v.AllSubordinateScore = allSubordinateScore + v.MemberTotalScore
	v.AllSubordinateUnuseScore = allSubordinateUnuseScore + v.MemberTotalUnuseScore

	return v.AllSubordinateScore, v.AllSubordinateUnuseScore, nil
}

func getAllMember(club *collClub.DBClubData) []*collClub.DBClubMember {
	v, ok := _clubAllMember.Load(club.ClubID)
	if ok == false {
		temp_ := make([]*collClub.DBClubMember, 0, 1000)
		recursionAllMember(club, &temp_)
		_clubAllMember.Store(club.ClubID, temp_)
		return temp_
	}
	return v.([]*collClub.DBClubMember)
}

func deleteAllMember(club *collClub.DBClubData) {
	_clubAllMember.Delete(club.ClubID)
	club_, err := loadClubData(club.ClubID)
	if err != nil {
		glog.Warning("deleteAllMember := ", club.ClubID)
		return
	}
	if club_.DirectSupervisor.ClubID < 1 {
		return
	}
	club_, err = loadClubData(club.DirectSupervisor.ClubID)
	if err != nil {
		glog.Warning("deleteAllMember := ", club.DirectSupervisor.ClubID)
		return
	}

	deleteAllMember(club_)
}

func recursionAllMember(club *collClub.DBClubData, allMemberArr *[]*collClub.DBClubMember) {
	*allMemberArr = append(*allMemberArr, club.MemberArr...)

	// 处理 下一级
	for i, _ := range club.DirectSubordinate {
		clubDataTemp_, err := loadClubData(club.DirectSubordinate[i].ClubID)
		if err != nil {
			glog.Warning(err.Error(), ",data:=", club.DirectSubordinate[i].ClubID)
			continue
		}
		recursionAllMember(clubDataTemp_, allMemberArr)
	}
}
