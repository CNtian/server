package db

import (
	"context"
	"errors"
	"github.com/bwmarrin/snowflake"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"time"
	collClub "vvService/dbCollectionDefine/club"
)

func NewClubPlay(clubID int32, clubPlay *collClub.DBClubPlay) (int32, error) {

	if snowflakeNode == nil {
		node, err := snowflake.NewNode(1)
		if err != nil {
			return 0, err
		}
		snowflakeNode = node
	}

	// Generate a snowflake ID.
	clubPlay.ID = int64(snowflakeNode.Generate())
	clubPlay.ID /= 100000

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	ctx := context.Background()

	session, err := mongoDBClient.StartSession()
	if err != nil {
		return 0, err
	}
	defer session.EndSession(ctx)

	collClub_ := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo, wcMajorityCollectionOpts)
	collPercentage := mongoDBClient.Database(databaseName).Collection(collClub.CollClubPlayPercentage, wcMajorityCollectionOpts)

	_, err = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {
		res, session_err := collClub_.UpdateOne(sctx,
			bson.M{"club_id": clubID},
			bson.M{"$push": bson.M{"plays": clubPlay}, "$inc": bson.M{"play_version_num": 1}})
		if session_err != nil {
			return 0, session_err
		}
		if res.ModifiedCount < 1 {
			return 0, errors.New("not match club play")
		}

		_, session_err = collPercentage.InsertOne(sctx,
			&collClub.DBClubPlayPercentage{MZClubID: clubID, PlayID: clubPlay.ID, ClubID: clubID,
				ShowPercentage: 100, RealPercentage: 100, RealBaoDiPer: 100, ShowBaoDiPer: 100})
		if session_err != nil {
			return 0, session_err
		}
		return 0, session_err
	})

	return 0, err
}

func UpdateClubPlay(clubID int32, clubPlayID int64, clubPlay *collClub.DBClubPlay) error {

	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	res, err := coll.UpdateOne(
		ctx,
		bson.M{"club_id": clubID, "plays.id": clubPlayID},
		bson.M{"$set": bson.M{"plays.$": clubPlay}, "$inc": bson.M{"play_version_num": 1}})
	if err != nil {
		return err
	}
	if res.ModifiedCount > 0 || res.MatchedCount > 0 {
		return nil
	}
	return err
}

func DeleteClubPlay(clubID int32, clubPlayID int64) error {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID, "plays": bson.M{"$elemMatch": bson.M{"id": clubPlayID}}}
	update := bson.M{"$set": bson.M{"plays.$.del": true, "plays.$.delete_time": time.Now()}, "$inc": bson.M{"play_version_num": 1}}
	return coll.FindOneAndUpdate(ctx, filter, update).Err()
}

func HideClubPlay(clubID int32, clubPlayID int64, value bool) error {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID, "plays": bson.M{"$elemMatch": bson.M{"id": clubPlayID}}}
	update := bson.M{"$set": bson.M{"plays.$.is_hide": value}, "$inc": bson.M{"play_version_num": 1}}
	return coll.FindOneAndUpdate(ctx, filter, update).Err()
}
