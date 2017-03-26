package store

import "github.com/boltdb/bolt"

type View func(*bolt.Tx) error

type Mutation func(*bolt.Tx) error
