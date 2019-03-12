package collector

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
)

// NewStorage creates a new storage backend
func NewStorage(uri string) (*Storage, error) {
	client, err := mongo.NewClient(uri)
	if err != nil {
		return nil, err
	}

	return &Storage{
		uri:    uri,
		Client: client,
	}, nil
}

// Storage interface implementation for badger
type Storage struct {
	Client *mongo.Client
	uri    string
	opts   []clientopt.Option
}

// Init checks that the store is usable
func (s *Storage) Init() error {
	err := s.Client.Connect(context.TODO())
	if err != nil {
		return err
	}
	return nil
}

// Close the database cleanly
func (s *Storage) Close() error {
	return s.Client.Disconnect(context.Background())
}
