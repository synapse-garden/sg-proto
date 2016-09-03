# v0.4.0 ?

- [ ] Versioned REST API?
- [ ] Distributed storage?
- [ ] Cassandra / etc bigtable backend?

# v0.3.0 ?

- [ ] Secure ticket "incept" endpoint (can you just hammer it with UUIDs?)
- [ ] Concurrent store.Wrap?
- [ ] Concurrent store.Wrap with dep chains?
- [ ] Dep chains?
- [ ] DB interface + cache?
- [ ] Optimize buckets / transactions in packages?  Pass needed behaviors
   through store package?  NewTickets, etc. inefficient
- [ ] Store.WrapBucket(store.Bucket(...), ...Transact)
- [ ] Decide about capnproto / protobuf for Bolt

# v0.2.0 ?

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

# v0.1.0

## Bugs

- [ ] Users can't understand missing session Error() string since it's bytes

## Unorganized

- [x] Standardize on JSON camelCase vs snake_case etc
- [ ] Organize TODOs
- [ ] Poms / some kind of work measure
- [ ] Some kind of psych features
- [ ] Make a decision on Rust
- [ ] Switch to encoding/gob instead of JSON on the backend and benchmark it
- [ ] Store tests
- [ ] Make a call about frontend hashing.  Do we really want to?
      Not really secure unless salted, and even then it's "just another password".
- [ ] https://cdn.jsdelivr.net/sjcl/1.0.4/sjcl.js for browser
      http://jsfiddle.net/kRcNK/40/
- [ ] Scour for cases where Put or Marshal could fail and return credentials
- [ ] Return x.ErrMissing, not store.ErrMissing, in Unmarshal cases
- [ ] User logout by uname + pwhash (DELETE /tokens ?)
  - [ ] Lookup from username to session

# v0.0.1

## Bugs

- [x] Fix Windows timestamp UUID generation (use uuid.NewV4)
- [x] Fix Windows startup BoltDB panic (nil transaction or db?)
- [x] body of POST to /incept/:ticket must include pwhash field
- [x] AuthAdmin expects base64 hashed sha256 of auth.Token (uuid Bytes)
- [x] Admin API key stored insecurely, must hash + salt first
  - [x] Report base64 encoded value
- [x] Can't log out because session is not URL-encoded

## Admin API

- [x] AuthAdmin middleware
- [x] Create ticket
- [x] Delete ticket
- [x] Master API key printed on startup?
  - [x] Use own API key via config?
  - [x] Fix admin key nonsense

## Code quality / package sanitation

- [ ] Comment all exported functions, types, methods, and constants
- [ ] Make sure not just anyone can get a refresh token

## Dev mode

- [ ] Specify me
- [ ] No database?
- [ ] Use given mock file as initial store?
- [ ] Expose endpoints without sessions?
- [ ] Default superuser logins given?

## GPL

- [x] Host own source code under /source or some such.

## Login / Session / Logout

- [x] auth.Session API
- [ ] auth.Login tests
- [ ] Invalidate / reissue auth token after refresh
- [x] Delineate split between account (users.User) and auth.Login
- [x] Session auth middleware
- [ ] Test HandleDeleteToken (URL encoding, etc.)

## Account

- [ ] incept.PunchTicket
- [x] Ticket API
- [x] Password hash
- [x] Create user
- [x] Log in
- [x] Log out
- [ ] Have bounty
- [x] Test rest.Incept auth.Login creation

## Chat

- [ ] Chat between two users (websocket)

## Todo

- [ ] Create item with bounty and due date
- [ ] Complete item before due date, receive bounty
