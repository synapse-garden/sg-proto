# v0.3.0 ?

- [ ] Concurrent store.Wrap?
- [ ] Concurrent store.Wrap with dep chains?
- [ ] Dep chains?
- [ ] DB interface?
- [ ] Optimize buckets / transactions in packages?  Pass needed behaviors
   through store package?
- [ ] Store.WrapBucket(store.Bucket(...), ...Transact)

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

# v0.0.1

## Bugs

- [ ] Fix Windows timestamp UUID generation
- [ ] Fix Windows startup BoltDB panic (nil transaction or db?)
- [ ] Fix panic in rest.Incept

## Account

- [ ] Ticket API
- [ ] Password hash
- [ ] Secure ticket "incept" endpoint
- [ ] Create user
- [ ] Log in
- [ ] Log out
- [ ] Have bounty

## Chat

- [ ] Chat between two users (websocket)

## Todo

- [ ] Create item with bounty and due date
- [ ] Complete item before due date, receive bounty
