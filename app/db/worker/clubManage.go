package worker

import (
	"encoding/json"
	"github.com/golang/glog"
	"vvService/appDB/db"
	collClub "vvService/dbCollectionDefine/club"
)

var clubDataMap = make(map[int32]*collClub.DBClubData)

func loadClubData(clubID int32) (*collClub.DBClubData, error) {
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
			return nil, err
		}
		clubDataMap[clubID] = clubData

		clubData.MemberMap = make(map[int64]*collClub.DBClubMember)
		for i, v := range clubData.MemberArr {
			clubData.MemberMap[v.ID] = clubData.MemberArr[i]
		}

		clubData.PlayIDMap = make(map[int64]*collClub.DBClubPlay)
		for i, v := range clubData.PlayArr {
			tempClubRule := collClub.DBClubRule{}
			json.Unmarshal([]byte(v.ClubCfg), &tempClubRule)
			clubData.PlayArr[i].ClubRule = &tempClubRule
			clubData.PlayArr[i].ClubRule.GetTextVale()
			clubData.PlayIDMap[v.ID] = clubData.PlayArr[i]
		}

		clubData.SubordinatesMap = make(map[int32]int32)
		for _, v := range clubData.Subordinates {
			clubData.SubordinatesMap[v] = v
		}
	}
	tempClubData = clubData

	if clubData.DirectSupervisor.ClubID > 0 {
		for tempClubData.DirectSupervisor.ClubID > 0 {
			tempClubData, err = loadClubData(tempClubData.DirectSupervisor.ClubID)
			if err != nil {
				glog.Warning("loadClubData() err.clubID:=", tempClubData.DirectSupervisor.ClubID,
					",err:=", err.Error())
				break
			}
			if tempClubData.DirectSupervisor.ClubID < 1 {
				clubData.MengZhuClubID = tempClubData.ClubID
				break
			}
		}
	}

	return clubData, nil
}

//func loadPlayerDataFromRedis(uid int64) (string, string) {
//	head, nick, err := db.LoadPlayerHead(uid)
//	if err != nil {
//		glog.Warning("loadPlayerDataFromRedis() err.", err.Error())
//	}
//	return head, nick
//}
