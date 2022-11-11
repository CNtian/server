package db

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	collClub "vvService/dbCollectionDefine/club"
)

func GetVirtualTableConfig(clubID int32, arr *[]collClub.VirtualTableConfigItem) error {
	collVirtualTable := mongoDBClient.Database(databaseName).Collection(collClub.CollVirtualTable)
	ctx := context.Background()

	cur, err := collVirtualTable.Find(ctx, bson.M{"club_id": clubID})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	tableCfg := collClub.DBVirtualTableConfig{ConfigItem: []collClub.VirtualTableConfigItem{}}
	for cur.Next(ctx) {
		err = cur.Decode(&tableCfg)
		if err != nil {
			break
		}
		*arr = tableCfg.ConfigItem
		break
	}

	return err
}

// ():新增,更新,错误
func UpdateVirtualTableConfig(clubID int32, item *collClub.VirtualTableConfigItem) (bool, bool, error) {
	collVirtualTable := mongoDBClient.Database(databaseName).Collection(collClub.CollVirtualTable)

	var (
		res *mongo.UpdateResult
		err error
		//opt options.UpdateOptions
	)

	//opt.SetUpsert(true)
	//opt.SetArrayFilters(options.ArrayFilters{Filters: bson.A{bson.M{"config_item.play_id": item.PlayID}}})

	virtualTable := collClub.DBVirtualTableConfig{}
	sr := collVirtualTable.FindOne(nil, bson.M{"club_id": clubID})
	err = sr.Decode(&virtualTable)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			virtualTable.ClubID = clubID
			virtualTable.ConfigItem = []collClub.VirtualTableConfigItem{*item}
			_, err = collVirtualTable.InsertOne(nil, virtualTable)
			if err == nil {
				return true, false, nil
			}
		}
		return false, false, err
	}

	find := false
	for i, _ := range virtualTable.ConfigItem {
		if virtualTable.ConfigItem[i].PlayID == item.PlayID {
			res, err = collVirtualTable.UpdateOne(nil, bson.M{"club_id": clubID, "config_item.play_id": item.PlayID},
				bson.M{"$set": bson.M{"config_item.$": *item}})
			find = true
			break
		}
	}
	if find == false {
		res, err = collVirtualTable.UpdateOne(nil, bson.M{"club_id": clubID},
			bson.M{"$push": bson.M{"config_item": *item}})
	}

	if res.ModifiedCount > 0 {
		return false, true, nil
	} else if res.UpsertedCount < 1 {
		return true, false, nil
	}

	return false, false, err
}

func DeleteVirtualTableConfigItem(clubID int32, playID int64) error {
	collVirtualTable := mongoDBClient.Database(databaseName).Collection(collClub.CollVirtualTable)

	_, err := collVirtualTable.UpdateOne(nil, bson.M{"club_id": clubID},
		bson.M{"$pull": bson.M{"config_item": bson.M{"play_id": playID}}})

	return err
}
