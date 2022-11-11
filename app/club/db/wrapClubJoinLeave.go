package db

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"time"
	commonDef "vvService/commonPackge"
	"vvService/commonPackge/errorCodeDef"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

// 直接加入俱乐部
func DragIntoClub(clubID int32, uid int64) (int32, error) {

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	ctx := context.Background()

	session, err := mongoDBClient.StartSession()
	if err != nil {
		return 0, err
	}
	defer session.EndSession(ctx)

	var rspCode int32
	// 加入俱乐部
	collClubInfo := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo, wcMajorityCollectionOpts)
	collPlayerInfo := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo, wcMajorityCollectionOpts)

	_, err = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {

		var (
			userInfo   collPlayer.PlayerInfo
			clubMember = collClub.DBClubMember{
				ID:           uid,
				JoinClubTime: time.Now(),
			}

			updateRes *mongo.UpdateResult
			findCount int64
		)
		err = collPlayerInfo.FindOne(sctx, bson.M{"uid": uid}).Decode(&userInfo)
		if err != nil {
			return nil, err
		}
		if len(userInfo.ClubData) >= 3 {
			rspCode = errorCodeDef.ErrClubJoinMore
			return nil, fmt.Errorf("club count too more.uid:=%d", uid)
		}
		for _, v := range userInfo.ClubData {
			if v.ClubID == clubID {
				rspCode = errorCodeDef.ErrClubAlreadyInMember
				return nil, fmt.Errorf("already in club.uid:=%d ,clubID:=%d", uid, clubID)
			}
		}
		findCount, err = collClubInfo.CountDocuments(sctx, bson.M{"club_id": clubID, "members.uid": uid})
		if err != nil {
			return nil, err
		}
		if findCount > 0 {
			rspCode = errorCodeDef.ErrClubAlreadyInMember
			return nil, fmt.Errorf("already in club.uid:=%d ,clubID:=%d", uid, clubID)
		}

		// 更新俱乐部
		filter := bson.M{"club_id": clubID}
		update := bson.M{"$push": bson.M{"members": clubMember}}
		updateRes, err = collClubInfo.UpdateOne(sctx, filter, update)
		if err != nil {
			return nil, err
		}
		if updateRes.ModifiedCount < 1 && updateRes.MatchedCount < 1 {
			rspCode = errorCodeDef.ErrClubNotExist
			return nil, fmt.Errorf("not match club id:=%d", clubID)
		}

		// 更新个人俱乐部
		clubData := collPlayer.ClubInfo{ClubID: clubID, LastPlayTime: time.Now()}
		filter = bson.M{"uid": uid}
		update = bson.M{"$push": bson.M{"club_info": clubData}}
		updateRes, err = collPlayerInfo.UpdateOne(sctx, filter, update)
		if err != nil {
			return nil, err
		}
		if updateRes.ModifiedCount < 1 && updateRes.MatchedCount < 1 {
			rspCode = errorCodeDef.Err_Not_Find_Item
			return nil, fmt.Errorf("not match uid:=%d", uid)
		}
		return nil, err
	})

	return rspCode, err
}

// 离开俱乐部
//():结果,剩余成员数量,错误
func ClubMemberExit(clubID int32, uid int64, isCompulsively bool) (int32, error) {
	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	ctx := context.Background()

	session, err := mongoDBClient.StartSession()
	if err != nil {
		return 0, err
	}
	defer session.EndSession(ctx)

	var (
		rspCode int32
	)

	collClubInfo := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo, wcMajorityCollectionOpts)
	collPlayerInfo := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo, wcMajorityCollectionOpts)

	_, err = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {

		filter := bson.M{}
		if isCompulsively == false {
			filter = bson.M{"uid": uid, "club_info": bson.M{"$elemMatch": bson.M{"clubID": clubID, "score": bson.M{"$gte": 0, "$lt": commonDef.SR}}}}
		} else {
			filter = bson.M{"uid": uid, "club_info": bson.M{"$elemMatch": bson.M{"clubID": clubID}}}
		}

		update := bson.M{"$pull": bson.M{"club_info": bson.M{"clubID": clubID}}}
		resUpdateResult, err := collPlayerInfo.UpdateOne(sctx, filter, update)
		if err != nil {
			return nil, err
		}
		if resUpdateResult.MatchedCount < 1 && resUpdateResult.ModifiedCount < 1 {
			rspCode = errorCodeDef.ErrClubNotMember
			return nil, fmt.Errorf("not match erorr in player info.clubid:=%d,uid:=%d", clubID, uid)
		}

		filter = bson.M{"club_id": clubID}
		update = bson.M{"$pull": bson.M{"members": bson.M{"uid": uid}}}
		resUpdateResult, err = collClubInfo.UpdateOne(sctx, filter, update)
		if err != nil {
			return nil, err
		}
		if resUpdateResult.MatchedCount < 1 && resUpdateResult.ModifiedCount < 1 {
			rspCode = errorCodeDef.ErrClubNotMember
			return nil, fmt.Errorf("not match erorr in club.clubid:=%d,uid:=%d", clubID, uid)
		}

		return nil, err
	})

	return rspCode, err
}

// 解散俱乐部
func DissolveClub(clubID int32, memberArr []int64, proxyID int64) error {
	ctx := context.Background()

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	collClub := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo, wcMajorityCollectionOpts)
	collPlayer_ := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo, wcMajorityCollectionOpts)

	session, err := mongoDBClient.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {

		// 创建者(移除关系)
		_, err = collClub.UpdateMany(sctx, bson.M{"club_id": clubID},
			bson.M{"$mul": bson.M{"creator_id": -1}})
		if err != nil {
			return nil, err
		}

		// 更新 玩家的所属俱乐部(移除关系)
		updateRes, err := collPlayer_.UpdateMany(sctx, bson.M{"uid": bson.M{"$in": memberArr}},
			bson.M{"$pull": bson.M{"club_info": bson.M{"clubID": clubID}}})
		if err != nil {
			return nil, err
		}

		if updateRes.ModifiedCount < 1 {
			return nil, fmt.Errorf(".not find match clubID:=%d", clubID)
		}

		if proxyID != 0 {
			updateRes, err = collPlayer_.UpdateOne(sctx, bson.M{"uid": proxyID}, bson.M{"$pull": bson.M{"proxy_club": clubID}})
			if err != nil {
				return nil, err
			}

			if updateRes.ModifiedCount < 1 {
				return nil, fmt.Errorf(".not find match clubID:=%d  proxyID:=%d", clubID, proxyID)
			}
		}

		return nil, nil
	})

	return err
}
