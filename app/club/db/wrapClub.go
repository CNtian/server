package db

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"time"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

// 合并俱乐部
func MergeClub(applyClubID, targetClubID int32) (int32, error) {
	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	ctx := context.Background()

	session, err := mongoDBClient.StartSession()
	if err != nil {
		return 0, err
	}
	defer session.EndSession(ctx)

	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo, wcMajorityCollectionOpts)

	_, err = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {

		var applyClubData collClub.DBClubData
		// 修改 申请者的 直属上级俱乐部
		filter := bson.M{"club_id": applyClubID, "d_superior.club_id": 0}
		update := bson.M{"$set": bson.M{"d_superior.club_id": targetClubID, "d_superior.per": 0, "d_superior.real_per": 0}}

		opt := options.FindOneAndUpdateOptions{}
		opt.SetProjection(bson.M{"subordinates": 1, "d_superior": 1})
		err1 := coll.FindOneAndUpdate(sctx, filter, update).Decode(&applyClubData)
		if err1 != nil {
			return nil, err1
		}

		// 修改 合并者 的 直属下级
		filter = bson.M{"club_id": targetClubID}
		update = bson.M{"$push": bson.M{"d_subordinate": collClub.DBClubMerge{ClubID: applyClubID, ShowPercentage: 0, RealPercentage: 0}}}
		result, err2 := coll.UpdateOne(sctx, filter, update)
		if err2 != nil {
			return nil, err2
		}
		if result.ModifiedCount < 1 {
			glog.Warning("MergeClub() applyClubID:=", applyClubID, ",targetClubID", targetClubID)
			return nil, fmt.Errorf("ModifiedCount<1 ")
		}

		subordinatesArr := applyClubData.Subordinates
		subordinatesArr = append(subordinatesArr, applyClubID)

		// 更新所有上级的  下属俱乐部
		tempClubID := targetClubID
		for tempClubID > 0 {
			var clubData collClub.DBClubData
			filter = bson.M{"club_id": tempClubID}
			update = bson.M{"$push": bson.M{"subordinates": bson.M{"$each": subordinatesArr}}}
			err1 = coll.FindOneAndUpdate(sctx, filter, update, &opt).Decode(&clubData)
			if err1 != nil {
				return nil, err
			}

			tempClubID = clubData.DirectSupervisor.ClubID
		}
		return nil, nil
	})
	return 0, err
}

func MergeClub_(applyPlayerID int64, applyClubID, targetClubID int32, clubScore int64, remark string) (int32, error) {
	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	ctx := context.Background()

	session, err := mongoDBClient.StartSession()
	if err != nil {
		return 0, err
	}
	defer session.EndSession(ctx)

	collClub_ := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo, wcMajorityCollectionOpts)
	collPlayer_ := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo, wcMajorityCollectionOpts)

	_, err = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {

		var applyClubData collClub.DBClubData
		// 修改 申请者的 直属上级俱乐部
		filter := bson.M{"club_id": applyClubID, "d_superior.club_id": 0}
		update := bson.M{"$set": bson.M{"d_superior.club_id": targetClubID, "d_superior.per": 0, "d_superior.real_per": 0, "score_count": clubScore}}

		opt := options.FindOneAndUpdateOptions{}
		opt.SetProjection(bson.M{"subordinates": 1, "d_superior": 1})
		sessionErr := collClub_.FindOneAndUpdate(sctx, filter, update).Decode(&applyClubData)
		if sessionErr != nil {
			return nil, sessionErr
		}

		// 修改 俱乐部分的 归属  1-删掉自己的俱乐部
		filter = bson.M{"uid": applyPlayerID, "club_info.clubID": applyClubID}
		update = bson.M{"$pull": bson.M{"club_info": bson.M{"clubID": applyClubID}}}
		updateRes, sessionErr := collPlayer_.UpdateOne(sctx, filter, update)
		if sessionErr != nil {
			return nil, sessionErr
		}
		if updateRes.ModifiedCount < 1 {
			return nil, fmt.Errorf("not match %d  %d", applyClubID, applyPlayerID)
		}
		// 2-修改 为自己的俱乐部
		filter = bson.M{"uid": applyPlayerID, "club_info.clubID": targetClubID}
		update = bson.M{"$set": bson.M{"club_info.$.clubID": applyClubID}}
		updateRes, sessionErr = collPlayer_.UpdateOne(sctx, filter, update)
		if sessionErr != nil {
			return nil, sessionErr
		}
		if updateRes.ModifiedCount < 1 {
			return nil, fmt.Errorf("not match %d  %d", applyClubID, applyPlayerID)
		}

		// 修改 合并者 的 直属下级
		filter = bson.M{"club_id": targetClubID}
		update = bson.M{"$push": bson.M{"d_subordinate": collClub.DBClubMerge{ClubID: applyClubID, PlayerID: applyPlayerID, ShowPercentage: 0, RealPercentage: 0, Remark: remark}}}
		update["$pull"] = bson.M{"members": bson.M{"uid": applyPlayerID}}
		updateRes, sessionErr = collClub_.UpdateOne(sctx, filter, update)
		if sessionErr != nil {
			return nil, sessionErr
		}
		if updateRes.ModifiedCount < 1 {
			glog.Warning("MergeClub() applyClubID:=", applyClubID, ",targetClubID", targetClubID)
			return nil, fmt.Errorf("ModifiedCount<1 ")
		}

		subordinatesArr := applyClubData.Subordinates
		subordinatesArr = append(subordinatesArr, applyClubID)

		// 更新所有上级的  下属俱乐部
		tempClubID := targetClubID
		for tempClubID > 0 {
			var clubData collClub.DBClubData
			filter = bson.M{"club_id": tempClubID}
			update = bson.M{"$push": bson.M{"subordinates": bson.M{"$each": subordinatesArr}}}
			sessionErr = collClub_.FindOneAndUpdate(sctx, filter, update, &opt).Decode(&clubData)
			if sessionErr != nil {
				return nil, sessionErr
			}

			tempClubID = clubData.DirectSupervisor.ClubID
		}
		return nil, sessionErr
	})
	return 0, err
}
