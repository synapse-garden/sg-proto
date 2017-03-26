package river_test

import . "gopkg.in/check.v1"

func (s *RiverSuite) TestCreateHook(c *C) {
	// When the Stream is first created, check the hooks bucket for
	// for that Stream's ID.
	//
	// Can we just always do this for certain kinds of Streams?  The
	// "hook" is just a per-package implementation which is called
	// first.  If some condition is satisfied, the package can
	// decide to do other things.
	//
	// The problem is that when the last Stream disconnects, whoever
	// is dealing with a Terminate hook needs to manage that too.
	// If they are not in the same context the first hook caller
	// was, they might not be able to clean up.  Remember, this kind
	// of thing would be better if it was stateless.
	//
	// You could use a special Bind.  Close over the Read func.
	// But that is still a memory state which can't be shared.
	// Share via message-passing, not via memory.
	//
	// Use-case:
	//   - User wants to create some Stream.
	//   - First, View some bucket I own.  Does it already have a
	//     River token saved?
	//     > If so, just connect to the Stream however you like.
	//     > Otherwise, do an Update, check again.
	//     > If still first Stream client, insert token, and do
	//       whatever your hook is.  (Spawn a Scribe?)
	//   - When leaving the Stream, View to check whether there is
	//     anyone else's token still left in the Stream.  If there
	//     is not, then run your Terminate hook.
}

func (s *RiverSuite) TestTerminateHook(c *C) {

}
