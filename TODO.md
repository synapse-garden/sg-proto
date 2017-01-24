# v0.5.0+ ?

- [ ] MF task kernel integration
- [ ] Cascading store.Delete / store.Update?
- [ ] Map TODOs to Github Issues?
- [ ] Map TODOs to SG streams / items?
- [ ] Multi-part Stream receives

# v0.4.0 ?

- [ ] Versioned REST API?
- [ ] Apple / etc Push API?
- [ ] Distributed storage?
  - [ ] If so, MUST revise Surv / Resp rivers!!!
- [ ] Cassandra / etc bigtable backend?
- [ ] Offer River connections besides Websocket?
- [ ] Consider package-global caching optimizations to things such as
      user ID hashes

# v0.3.0 ?

- [ ] Refactor Streams database funcs
  - [ ] Marshal, Unmarshal, Delete, Get etc. take multi bolt Buckets
- [ ] Secure ticket "incept" endpoint (can you just hammer it with UUIDs?)
- [ ] Concurrent store.Wrap?
- [ ] Concurrent store.Wrap with dep chains?
- [ ] Dep chains?
- [ ] DB interface + cache?
- [ ] Optimize buckets / transactions in packages?  Pass needed behaviors
   through store package?  NewTickets, etc. inefficient
- [ ] Store.WrapBucket(store.Bucket(...), ...Transact)
- [ ] Decide about capnproto / protobuf for Bolt / Rivers
- [ ] Read-only Streams
- [ ] Finer-grained read authorization, public / private / circles?
- [ ] Only one notification stream exists per user
- [ ] Package user can specify how it works
- [ ] Consider using a salted hash for stream topics
- [ ] Consider globally buffering all streams

# v0.2.0 ?

- [ ] Notif global Topic
- [ ] Decide whether auth.Refresh should delete and exchange the given refresh token
- [ ] "Friendly UUIDs" -- map 4-bit chunks to phonemes or small words?
- [ ] "HTTP Errors" -- this is really two problems.
  - [ ] 1. JSON-serialized form errors that can be used to indicate problems
       in a context-friendly way
  - [ ] 2. HTTP error codes which have some relevance to the API user to help
       clarify what went wrong without passing forward sensitive data.
- [ ] Better database testing -- maybe a memory mapped file or some other
      option so our setups / teardowns don't have to thrash the filesystem.
- [ ] Testable rest.Bind
- [ ] Maybe a database mock?
- [ ] Caching database wrapper
- [ ] Use bolt batch
- [ ] Bucket threading
- [ ] Transaction type to replace func tx blah
- [ ] Better ErrorMissing / ErrorExists context messages
  - [ ] HTTP / Display Error interface which has Code and Message suitable for users
  - [ ] ValidationError interface which tells the client what keys / etc are wrong?
- [ ] Figure out whether we can find a logical mapping for UUID / base64
      shasum strings
- [ ] Notification topic performance
  - [ ] Individual publishers for GUIDs instead of API-global
    - [ ] Group.Hash -> GUID unique by users (?)
	- [ ] Create and destroy Group pub rivers on Group update
	- [ ] API pub river publishes to GUID subscribers, GUIDs publish to
	      their subscribers (???)
	- [ ] Clarify this API / sketch up some tests
- [ ] Refactor websocket Connect REST methods into nested testable steps

# v0.1.0

## Bugs

- [ ] Users can't understand missing session Error() string since it's bytes
  - [ ] Configure error output to match expected values: base64 shasums or
        UUID strings
- [ ] Invalidate / reissue auth token after refresh
  - [ ] Figure out how to thread session context through this

## Dev mode

- [ ] Specify me
- [ ] No database?
- [ ] Use given mock file as initial store?
- [ ] Expose endpoints without sessions?
- [ ] Default superuser logins given?

## Errors

- [ ] An interface is provided for API-serializable errors
- [ ] An interface is provided for non-lethal errors
  - [ ] IsLethal
  - [ ] Stop returning bools from websocket Conn reader

## Unorganized

- [ ] Organize TODOs

- [ ] "Resource" interface to make CRUD much, much simpler
- [x] The design of Rivers must support a future implementation which
      permits the API to use req/rep Rivers to control the behavior of
	  receivers.
- [x] Reorganize streams package with more abstraction
- [x] Standardize on JSON camelCase vs snake_case etc
- [ ] All the database backend stuff is a spaghettified nightmare since
      each package manages its own database behaviors.
- [ ] Tighten down and offer Streams as a CLI option.
- [ ] Remove self from stream / convo if you don't want to be in it,
      even if you don't own it.
- [ ] Redesign streams / hangup event chains
- [ ] Don't encode the same resource over and over for notifs.
- [ ] No way of cleaning up failed Scribes
- [ ] REST resources as interface / code gen?
- [ ] Tighten up convo message funcs
- [ ] Make convo message blocks
- [ ] Test convo message db funcs
- [ ] Messages keyed by ID instead of date
  - [ ] Migration for this?
- [ ] Test notif hangups
- [ ] Make a helper function to make hangups easier to use.
- [ ] ws.HangupSender a horrible mess.  Do something better, for the
      love of God!
- [ ] REST Stream tests brittle
- [ ] Survey response errors need a useful error implementation.
- [ ] Swagger HTTP API doc
- [ ] Poms / some kind of work measure
- [ ] Some kind of psych features
- [ ] Make a decision on Rust
- [ ] Switch to encoding/gob instead of JSON on the backend and benchmark it
  - [ ] Why not protobuf, msgpack, colfer, capnproto?
  - [ ] Some other dynamic schema?
  - [ ] Make a simple call and defer this decision.
- [ ] "store" package tests
- [ ] Make a call about frontend hashing.  Do we really want to?
      Not really secure unless salted, and even then it's "just another password".
- [ ] https://cdn.jsdelivr.net/sjcl/1.0.4/sjcl.js for browser
      http://jsfiddle.net/kRcNK/40/
- [ ] Scour for cases where Put or Marshal could fail and return credentials
  - [ ] ???
- [ ] Return x.ErrMissing, not store.ErrMissing, in Unmarshal cases
- [ ] User logout by uname + pwhash (DELETE /tokens ?)
  - [x] Lookup from username to session
- [ ] Users can GET /streams with search parameters
- [ ] More streams abstraction (better Filters, IsMember, etc. moved into package API)
- [ ] Make plan to reduce / eliminate rest.Stream API redundancy
  - [ ] ???
- [ ] Make plan to reduce all rest redundancy
- [ ] Chat endpoint which uses rest.ConnectStream with a river.Messager under the hood!
- [ ] Sanely handle stream errors
- [ ] Thoroughly test ws package
- [ ] client package uses custom HTTP client instead of global

# v0.0.1

## Bugs

- [x] Fix Windows timestamp UUID generation (use uuid.NewV4)
- [x] Fix Windows startup BoltDB panic (nil transaction or db?)
- [x] body of POST to /incept/:ticket must include pwhash field
- [x] AuthAdmin expects base64 hashed sha256 of auth.Token (uuid Bytes)
- [x] Admin API key stored insecurely, must hash + salt first
  - [x] Report base64 encoded value
- [x] Can't log out because session is not URL-encoded
- [x] River Bind never returns, so River is never cleaned up
- [x] Fix failing or blocking tests
- [x] Bus and Sub Rivers must never overwrite existing IDs
- [x] river.Surveyor and river.Respondent require a slight pause between
      Dial and usage.  Data race found due to mangos Init!
	  https://github.com/go-mangos/mangos/issues/236
- [x] Convo message parse errors are NOT sent to the websocket!
- [x] Convo Delete does not do anything if no errors.
- [x] Stream Delete does not do anything if no errors.
- [x] Convo connect/disconnect notifs don't specify convo ID.
- [x] Stream connect/disconnect notifs don't specify stream ID.
- [x] Stream Put does not require the user to own the Stream.
- [x] Convo Put does not require the user to own the Convo.
- [x] Convo Messages GET has incorrect range
- [x] Convo delete notif should not use stream delete
- [x] 404 on empty messages, should be populated on convo create.
- [x] Convo Delete should hang up Scribe and users.
- [x] Stream Delete should hang up users.
- [x] Convo Delete should remove Scribe checkins bucket.
- [x] Convo Delete should remove convo's messages bucket.
- [x] Convo Delete should have correct auth error message on DELETE.
- [x] Convo.Bind should never silently drop an error on NewPub.
- [x] convo.Scribe.DeleteCheckins should not panic if the Checkins
      bucket is missing; this is normal and means no checkins exist yet.
- [x] If user tries to check out of deleted convo (i.e. closes websocket), fatal error occurs:
  > failed to check out of convo: no such bucket `81c4b367-7cd0-46a1-90d0-618fb5c790b8`
- [x] Empty GET on /convos should return [], not null
- [x] Notifs arrive with contents base64-encoded
- [x] Fatal race in convo Scribe hangup on DELETE
- [x] Convo / Stream PUT which removes users must also hang them up.
- [x] Race / 500 on convo websocket close / convo delete.
- [x] Race in Convo PUT due to hangup
- [x] Race in Scribe hangup / Convo Delete.
- [x] scribe DeleteCheckins fails if the Scribe had no checkins.
- [x] Notif hangup Recv never finishes if the websocket is closed
- [ ] No auth timeout / river / notifs closure
- [ ] Tokens don't refresh on activity
  - [ ] Add this to token middleware?
- [ ] Old Bus buckets should be deleted after the convo or stream is deleted.
- [ ] Diagnose occasional test failures in RiverSuite.TestNewBus
- [ ] Better testing of REST resource security.
- [ ] Deleting the user's profile doesn't close his Streams.
- [o] Surveyor / Respondent don't keep track of who's still alive.  If a
      Responder removes itself from its bucket, the Survey will fail.
  - STATUS: "Solved" by post-check if some didn't respond.
- [ ] If a survey has a problem, responders are in an unknown state.
- [ ] Refresh tokens must be concatenated to auth tokens in header
- [ ] Refresh tokens must not zombify expired auth tokens, instead
      create new tokens
- [ ] Refresh tokens must be able to be invalidated
- [ ] User auth should return same token if not expired? (TODO: understand this)
- [ ] Performance is terrible (~30ms on GET on /source???)
  - [ ] Is it just Postman?
  - [ ] Benchmarking?
  - [ ] Bolt?
    - [ ] Configure cache settings?
- [ ] Deleting the user's profile doesn't eliminate his owned objects.
- [ ] Bad usernames cannot be looked up for expired Sessions

## Admin API

- [x] AuthAdmin middleware
- [x] Create ticket
- [ ] GET /tickets?per_page=n&page=m
- [x] Delete ticket(s)
- [x] Master API key printed on startup?
  - [x] Use own API key via config?
  - [x] Fix admin key nonsense

## Code quality / package sanitation

- [x] Split Streams and Rivers
- [ ] Make sure all empty GETs return [], not null
- [ ] Tighten up Convo REST API, add defered cleanups
- [ ] Update README.md and CONTRIBUTING.md, clean up 0.0.1 TODO
- [ ] Comment all exported functions, types, methods, and constants
- [ ] Make sure not just anyone can get a refresh token
- [ ] Log ERROR statements on all unexpected internal errors
- [ ] Update store.Version

## GPL

- [x] Host own source code under /source or some such.

## Login / Session / Logout

- [x] auth.Session API
- [ ] auth.Login tests
- [x] Delineate split between account (users.User) and auth.Login
- [x] Session auth middleware
- [ ] Test HandleDeleteToken (URL encoding, etc.)
- [x] Session key => session context (user ID, etc.) lookup

## Account

- [ ] incept.PunchTicket
- [x] Ticket API
- [x] Password hash
- [x] Create user
- [x] Log in
- [x] Log out
- [x] Test rest.Incept auth.Login creation

## Profile

- [x] GET /profile
- [x] DELETE /profile

- [ ] Have bounty
- [ ] User is notified when profile changes (e.g. bounty increase)
- [ ] Update with new password

## Ledger?

## Streams

- [x] Stream has multiple Rivers
- [x] Rivers can be created and deleted, and dial one another using
      Mangos inproc Bus protocol
- [x] Rivers can Send() and Recv() and Close()
- [x] Rivers close endpoints when told
- [x] ClearRivers (eliminates river cache on startup)
- [x] Stream REST API
- [x] Users can GET /streams they belong to, not just Streams they own
- [x] SSL "wss" works correctly
- [x] Multiple Bus Rivers per Stream per User
- [x] User is notified when added to a Stream
- [x] Stream members are notified when a user connects to a Stream
- [x] Stream members are notified when a user leaves a Stream
- [ ] Delete meta buckets on Convo close.
- [ ] Close running stream from API (use Survey/Resp)
- [ ] Removing a user from a Stream hangs up the user's Stream bindings
- [ ] Use https://golang.org/pkg/net/http/httptrace/ for REST test
- [ ] Inactive Rivers eventually time out

## Notifications

- [x] notif.MakeUserTopic returns a notif.UserTopic generated uniquely
      using "USER"+BLAKE2(id).
- [x] User can connect to ws to subscribe to notifs on topic uniquely
      generated from username.
- [x] APIs publish notifs to each affected user
  - [x] A user cannot spoof the topic by making their username something
        colliding with another user's topic.  ("john" vs "johndoe")
  - [x] Use u/BLAKE2 hash of username.
  - [x] Notif package generates a 64-byte unique ID to prepend the User's
        topics with.
- [x] Pub topics are the output of some function, the API does not use
      its own topics.
- [x] The user switches on the prefix to the topic in order to subslice
      the message, removing the topic slice.
- [x] Messages sent by the user on the websocket do nothing.
- [x] Only an authenticated user can obtain a sub River.
- [x] An authenticated user can obtain more than one sub River at once.
- [x] Topics are loaded by the sub river from a user bucket in streams.
      I.e., at an API level, the notification rivers belonging to the
	  user are interfaced via a single Stream having the user's ID.
- [x] Notifs can be hung up.

## Chat

- [x] Convos are Streams with a REST interface
- [x] Convo websocket interactions are well-tested
- [x] Removing a user from a convo hangs up their convos
- [x] Everything is identical to Bus rivers but:
  - [x] convos have their own bucket
  - [x] the SocketReader wraps messages with username and timestamp
  - [x] a Scribe connection for the convo is requested by the first to
        join the convo, and deleted by the last to leave
	- [ ] Future: Scribe is a single Sub Recver which is created on init
	      and cannot be hung up, senders double-post to it
	- [x] Present: Scribe is an orphan Bus Recver which doesn't send, is
	      created if not present by first person to join convo, and is
		  hung up by last person to leave
  - [ ] Convo connections time out when inactive for a while (15 min?)
  - [ ] Convos time out when login times out

- [x] GET /convos/<id>/messages? ( start/end/etc )
  - [ ] More filters

- [x] Chat between two or more users (on top of streams API)
- [x] Chat messages stored
- [x] Chat messages queryable (backward?) by timestamp and paginated
- [x] User sends {"content":"string"} which gets bound with username
- [x] On malformed client message, error message is written to websocket
- [x] Unregister reader on close

- [ ] Filters on GET
  - [ ] Sender: &sender=<userID str>
  - [ ] Date: &begin=<RFC3999>, &end=<RFC3999>
  - [ ] Max: &num=<int>
  - [ ] Paginate: &per_page=<int>, &page=<int>
  - [ ] Filter by max messages
    - [x] Default to last 50
- [x] Notify user when someone creates a convo with them
- [x] Notify user when they are added to a convo
- [x] Notify user when they are removed from convo
- [x] Notify user when someone connects to convo
- [x] Notify user when someone leaves convo
- [x] Handle errors sanely
- [x] Test what happens when one or more users hang up, etc

## Todo

- [ ] Create item with bounty and due date
- [ ] Complete item before due date, receive bounty
- [ ] Notifications
  - [ ] Notify on CRUD
