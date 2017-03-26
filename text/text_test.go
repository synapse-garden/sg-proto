package text_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/synapse-garden/sg-proto/store"
	sgt "github.com/synapse-garden/sg-proto/testing"
	"github.com/synapse-garden/sg-proto/text"

	"github.com/boltdb/bolt"
	uuid "github.com/satori/go.uuid"
	. "gopkg.in/check.v1"
)

type TextSuite struct {
	*bolt.DB

	tmpDir string
}

var _ = Suite(&TextSuite{})
var _ = store.LoadStorer(text.ID(uuid.Nil))

func Test(t *testing.T) { TestingT(t) }

func (s *TextSuite) SetUpTest(c *C) {
	db, tmpDir, err := sgt.TempDB("sg-stream-test")
	c.Assert(err, IsNil)
	c.Assert(db.Update(store.Wrap(
		store.Migrate(store.VerCurrent),
		store.SetupBuckets(text.TextBucket),
	)), IsNil)
	s.DB, s.tmpDir = db, tmpDir
}

func (s *TextSuite) TearDownTest(c *C) {
	c.Assert(sgt.CleanupDB(s.DB), IsNil)
	c.Assert(os.Remove(s.tmpDir), IsNil)
}

func (s *TextSuite) TestLoadStore(c *C) {
	foo := struct{}{}
	c.Check(s.View(
		text.ID(uuid.Nil).Load(foo),
	), ErrorMatches, "unexpected Load argument of type struct {}")

	txt := "hello there"
	into := new(string)
	id := text.ID(uuid.NewV4())
	c.Assert(s.Update(id.Store(txt)), IsNil)
	c.Assert(s.View(id.Load(into)), IsNil)
	c.Check(*into, Equals, txt)

	more := []string{"hi", "how are you", "I'm well thanks"}
	intos := []*string{new(string), new(string), new(string)}
	ids := []uuid.UUID{uuid.NewV4(), uuid.NewV4(), uuid.NewV4()}
	c.Assert(s.Update(store.Wrap(
		text.ID(ids[0]).Store(more[0]),
		text.ID(ids[1]).Store(more[1]),
		text.ID(ids[2]).Store(more[2]),
	)), IsNil)

	c.Assert(s.View(store.Wrap(
		text.ID(ids[0]).Load(intos[0]),
		text.ID(ids[1]).Load(intos[1]),
		text.ID(ids[2]).Load(intos[2]),
	)), IsNil)

	for i := 0; i < 3; i++ {
		c.Check(*intos[i], Equals, more[i])
	}

	c.Assert(s.Update(id.Delete), IsNil)

	err := s.View(id.Load(into))
	c.Check(
		reflect.TypeOf(err),
		DeepEquals,
		reflect.TypeOf(new(store.MissingError)),
	)

	c.Check(err.Error(), Equals, fmt.Sprintf(
		"key %q not in bucket `text`", id,
	))

}
