package clubMGR

import (
	"encoding/json"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/mongo"
	"vvService/appClub/db"
	collClub "vvService/dbCollectionDefine/club"
)

func _loadClubData(clubID int32) (*collClub.DBClubData, error) {
	var err error

	v, ok := _clubMap[clubID]
	if ok == true {
		return v, nil
	}

	v, err = reloadClubData(clubID)

	return v, err
}

func _delLocalClubData(clubID int32) {
	delete(_clubMap, clubID)
}

func reloadClubData(clubID int32) (*collClub.DBClubData, error) {
	var (
		clubData     *collClub.DBClubData
		err          error
		ok           bool
		tempClubData *collClub.DBClubData
	)

	clubData, ok = _clubMap[clubID]
	if !ok {
		clubData, err = db.LoadClub(clubID)
		if err != nil {
			if err != mongo.ErrNoDocuments {
				glog.Warning("db.LoadClub() err. err:=", err.Error(), ", clubID:=", clubID)
			}
			return nil, err
		}
		_clubMap[clubID] = clubData

		//glog.Warning("_loadClubData clubID:=", clubID, ",", clubData.ProxyUp)

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
			tempClubData, err = reloadClubData(tempClubData.DirectSupervisor.ClubID)
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
