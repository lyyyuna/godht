package mongointegr

import (
	"context"
	"encoding/hex"
	"fmt"
	"godht/pkg/dht"
	"time"

	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// PersistStore structure of persist layer
type PersistStore struct {
	mclient       *mongo.Client
	mgetpeerquery *mongo.Collection
	mannouncement *mongo.Collection
}

// NewMongoClient create a new mongo client
func NewMongoClient(addr string) (*PersistStore, error) {
	mongoserver := fmt.Sprintf("mongodb://%s", addr)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoserver))
	if err != nil {
		return nil, err
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}

	collection := client.Database("test").Collection("infohash")
	collection2 := client.Database("test").Collection("announcement")

	return &PersistStore{
		mclient:       client,
		mgetpeerquery: collection,
		mannouncement: collection2,
	}, nil
}

// InsertOneInfoHash to insert one infohash from get_peer query to mongo server
func (ps *PersistStore) InsertOneInfoHash(peerQuery *dht.GetPeersQuery) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	infohash := hex.EncodeToString([]byte(peerQuery.Infohash))
	addr := peerQuery.Src.String()

	opup := options.Update()
	opup.SetUpsert(true)
	_, err := ps.mgetpeerquery.UpdateOne(
		ctx,
		bson.D{
			{"infohash", infohash},
		},
		bson.D{{
			"$set",
			bson.D{{"addr", addr}},
		}},
		opup,
	)
	if err != nil {
		glog.Errorf("Insert error: %v", err)
	}
}

func (ps *PersistStore) InsertOneAnnouncement(announce *dht.Announcement) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	infohash := hex.EncodeToString([]byte(announce.Infohash))
	nodeaddr := announce.Src.String()
	peeraddr := announce.Peer.String()

	opup := options.Update()
	opup.SetUpsert(true)
	_, err := ps.mannouncement.UpdateOne(
		ctx,
		bson.D{
			{"infohash", infohash},
		},
		bson.D{{
			"$set",
			bson.D{{"nodeaddr", nodeaddr}, {"peeraddr", peeraddr}},
		}},
		opup,
	)
	if err != nil {
		glog.Errorf("Insert error: %v", err)
	}
}
