package task

import (
	"encoding/json"
	"time"

	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/text"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
)

// TaskBucket is the Bucket for Tasks.
var TaskBucket = store.Bucket("tasks")

// Task is a job or task which can be handled by someone.
type Task struct {
	users.Group

	ID   ID     `json:"id"`
	Name string `json:"name"`

	Bounty int64 `json:"bounty,omitempty"`

	Completed   bool       `json:"completed"`
	CompletedBy string     `json:"completedBy,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	Due *time.Time `json:"due,omitempty"`

	// Resources are foreign keys into the Text bucket.  They should
	// never be exposed to the user.
	Resources []text.ID `json:"resources,omitempty"`

	// Notes are actual string resources passed with the Task.
	// They are not stored in the task's Bucket.
	Notes []string `json:"notes,omitempty"`
}

// GetAll returns a function which unmarshals all tasks for which the
// user has ownership.  If Filters are passed, only tasks for which
// filter.Member(task) == true will be returned.  Note that the order of
// the returned slice is determined by the IDs of the Tasks, which are
// random UUIDs.
func GetAll(user string, filters ...Filter) func(*bolt.Tx) ([]*Task, error) {
	var result []*Task

	defaultFilter := MultiOr{
		ByOwner(user),
		ByReader(user),
		ByWriter(user),
	}
	otherFilters := MultiAnd(filters)

	return func(tx *bolt.Tx) ([]*Task, error) {
		b := tx.Bucket(TaskBucket)
		// TODO: channel producer / consumer to speed this up
		// TODO: Other ways to improve this so users aren't
		//       constantly hammering the database
		// TODO: Benchmark test
		err := b.ForEach(func(k, v []byte) error {
			next := new(Task)
			if err := json.Unmarshal(v, next); err != nil {
				return err
			}

			switch {
			case !defaultFilter.Member(next):
				return nil
			case !otherFilters.Member(next):
				return nil
			}

			if err := next.ID.Load(next)(tx); err != nil {
				return err
			}

			result = append(result, next)
			return nil
		})

		return result, err
	}
}

// ID is a store.LoadStorer for Tasks.
type ID store.ID

// Store implements store.Storer on ID.
func (i ID) Store(what interface{}) func(*bolt.Tx) error {
	tsk, ok := what.(*Task)
	if !ok {
		return store.Errorf("unexpected Store arg of type %T", what)
	}

	notes := tsk.Notes
	tsk.ID, tsk.Notes = i, nil
	old := new(Task)
	idBytes := i[:]
	return store.Wrap(
		store.View(store.Unmarshal(TaskBucket, old, idBytes)).OrMissing,
		tsk.StoreResources(store.ID(i), notes, old.Resources),
		store.Marshal(TaskBucket, tsk, idBytes),
	)
}

// Load implements store.Loader on TaskID.  It also loads all text.
func (i ID) Load(into interface{}) func(*bolt.Tx) error {
	tsk, ok := into.(*Task)
	if !ok {
		return store.Errorf("unexpected Load arg of type %T", into)
	}

	return func(tx *bolt.Tx) error {
		err := store.Unmarshal(TaskBucket, tsk, i[:])(tx)
		if err != nil {
			return err
		}

		lRes := len(tsk.Resources)
		if lRes == 0 {
			return nil
		}

		resources := make(store.Loaders, lRes)
		texts := make([]interface{}, lRes)
		notes := make([]string, lRes)

		for i, r := range tsk.Resources {
			resources[i] = r
			texts[i] = new(string)
		}

		if err := resources.LoadAll(texts...)(tx); err != nil {
			return err
		}

		for i, t := range texts {
			notes[i] = *(t.(*string))
		}

		tsk.Notes = notes
		tsk.Resources = nil

		return nil
	}
}

// Delete deletes the task with the given ID.
func (i ID) Delete(tx *bolt.Tx) error {
	// Note that since all Resource IDs are hashes with the Task's
	// ID (which is unique), the chance of collision is nearly zero.
	var (
		tsk     = new(Task)
		idBytes = i[:]
	)
	return store.Wrap(
		store.View(store.Unmarshal(TaskBucket, tsk, idBytes)).OrMissing,
		DeleteResources(tsk.Resources),
		store.Delete(TaskBucket, i[:]),
	)(tx)
}

// StoreResources uses the given ID to check for new and deleted Text.
// Then the deleted Texts are removed from the database, while new ones
// are stored.
func (t *Task) StoreResources(
	hasher store.ID,
	what []string,
	old []text.ID,
) func(*bolt.Tx) error {
	// In this case, we know all our resources are actually
	// IDs generated from the hash of some contents.  If those
	// hashes no longer exist, we should delete them from the
	// database.  If they already exist, we don't need to store them
	// again.
	exist := make(map[text.ID]bool)
	for _, o := range old {
		exist[o] = true
	}

	var storers store.Storers
	var toStore []interface{}
	var toDelete store.Deleters
	for j, note := range what {
		id := text.ID(hasher.HashWith(note))
		if !exist[id] {
			storers = append(storers, id)
			toStore = append(toStore, what[j])
		} else {
			toDelete = append(toDelete, id)
		}
	}

	ids := make([]text.ID, len(storers))
	for i, s := range storers {
		ids[i] = s.(text.ID)
	}
	t.Resources = ids

	return store.Wrap(
		storers.StoreAll(toStore...),
		toDelete.DeleteAll,
	)
}

func DeleteResources(rs []text.ID) store.Mutation {
	resources := make(store.Deleters, len(rs))
	for i, r := range rs {
		resources[i] = r
	}
	return resources.DeleteAll
}
