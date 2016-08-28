# v0.4.0 ?

- [ ] Distributed storage?
- [ ] Cassandra / etc bigtable backend?

# v0.3.0 ?

- [ ] Secure ticket "incept" endpoint (can you just hammer it with UUIDs?)
- [ ] Concurrent store.Wrap?
- [ ] Concurrent store.Wrap with dep chains?
- [ ] Dep chains?
- [ ] DB interface?
- [ ] Optimize buckets / transactions in packages?  Pass needed behaviors
   through store package?  NewTickets, etc. inefficient
- [ ] Store.WrapBucket(store.Bucket(...), ...Transact)
- [ ] Decide about capnproto / protobuf for Bolt

# v0.2.0 ?

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
- [ ] Use bolt batch
- [ ] Bucket threading
- [ ] Transaction type to replace func tx blah
- [ ] Better ErrorMissing / ErrorExists context messages

# v0.1.0

- [ ] Poms / some kind of work measure
- [ ] Some kind of psych features
- [ ] Make a decision on Rust
- [ ] Switch to encoding/gob instead of JSON on the backend
- [ ] Store tests
- [ ] Make a call about frontend hashing.  Do we really want to?
      Not really secure unless salted, and even then it's "just another password".
- [ ] https://cdn.jsdelivr.net/sjcl/1.0.4/sjcl.js for browser
      http://jsfiddle.net/kRcNK/40/

# v0.0.1

## Dev mode

- [ ] Specify me
- [ ] No database?
- [ ] Use given mock file as initial store?
- [ ] Expose endpoints without sessions?
- [ ] Default superuser logins given?

## GPL

- [x] Host own source code under /source or some such.

## Bugs

- [x] Fix Windows timestamp UUID generation (use uuid.NewV4)
- [x] Fix Windows startup BoltDB panic (nil transaction or db?)
- [x] body of POST to /incept/:ticket must include pwhash field

## Login / Session

- [ ] auth.Session API?
- [ ] auth.Login tests
- [x] Delineate split between account (users.User) and auth.Login

## Account

- [ ] incept.PunchTicket
- [x] Ticket API
- [x] Password hash
- [x] Create user
- [ ] Log in
- [ ] Log out
- [ ] Have bounty
- [ ] Test rest.Incept auth.Login creation

## Chat

- [ ] Chat between two users (websocket)

## Todo

- [ ] Create item with bounty and due date
- [ ] Complete item before due date, receive bounty
