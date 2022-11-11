package db

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"time"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

// ():正分,负分
func GetMemberScore(clubID int32, playerIDArr []int64) (int64, int64, error) {
	collPlayerInfo := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo)

	ctx := context.Background()
	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"club_info": 1})
	cur, err := collPlayerInfo.Find(ctx, bson.M{"uid": bson.M{"$in": playerIDArr}}, &opt)
	if err != nil {
		return 0, 0, err
	}
	defer cur.Close(ctx)

	var gte0, less0 int64
	for cur.Next(ctx) {
		t_ := collPlayer.PlayerInfo{}
		err = cur.Decode(&t_)
		if err != nil {
			return 0, 0, err
		}
		for i, _ := range t_.ClubData {
			if t_.ClubData[i].ClubID == clubID {
				if t_.ClubData[i].Score < 0 {
					less0 += t_.ClubData[i].Score
				}
				gte0 += t_.ClubData[i].Score
			}
		}
	}
	return gte0, less0, nil
}

type StocktakingInfo struct {
	MemberTotalScore      int64
	MemberTotalUnuseScore int64

	AllSubordinateScore      int64
	AllSubordinateUnuseScore int64
}

func SetClubTotalScoreUnuseScore(vMap map[int32]*StocktakingInfo) error {
	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	collClub_ := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo, wcMajorityCollectionOpts)

	ctx := context.Background()

	session, outSideErr := mongoDBClient.StartSession()
	if outSideErr != nil {
		return outSideErr
	}
	defer session.EndSession(ctx)

	_, outSideErr = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {

		for k, v := range vMap {
			_, err := collClub_.UpdateOne(sctx, bson.M{"clubID": k}, bson.M{"$set": bson.M{"score_count": v.AllSubordinateScore, "unusable_score": v.AllSubordinateUnuseScore}})
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return outSideErr
}
