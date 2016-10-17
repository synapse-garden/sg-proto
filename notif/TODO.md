# Notif Ideation

Pub/sub for backend API pieces to communicate
Implemented by River where stream ID is API name and only API points can
  join

API flow:
  - API which wants to send notifs binds to "notif/_apiName_" Pub river
    - Topic is byte mapped from _apiName_ package const
	- API can use stream's River features immediately
  - user logs in
  - authed user makes GET on wss://.../notif
    - "notif" bucket river created for user
	- river dials other bus endpoints in stream
  - some other API endpoint is called
  - the relevant users are updated with the new object

P = Pub
S = Sub
x = Same user websocket conn Bind

 x-S  S  S
    \ | /
x-S--(P)--S
    / | \
 x-S  S  S
 
THE DUMB WAY:
- Just use the UserID as the topic.

ANOTHER DUMB WAY:
- Could it be even dumber without pub/sub if users join a notif stream
  which they update one another with?  Then you have to trust the user
  (could hash the message on the server?) but the user can define their
  own update mechanism.

Notif authorization:
- [ ] The end-user only gets notifications on things he has at least
      read privilege for.
  - [ ] When the end-user's privileges change, he must stop or start
        receiving updates on the changed resources.
    - [ ] This would be easier to manage if privilege was described by a
	      single collection.
	- [ ] A user subscribes to the ID of a resource he has read priv on.
	- [ ] When he loses privilege, the topic changes at the pub.
	- [ ] When a privilege changes, the publisher must notify all users
	      (who are still privileged) about the new token, with req/rep.
	- [ ] So: req/rep is how privilege updates have to happen on rivers.
  - [ ] A notification river is a topical subscriber to a notif pub.
  - [ ] Either the subscriber must change, or the publisher must change.
  - [ ] The publisher has knowledge of the updated resource.
  - [ ] The subscriber only has knowledge of the topic and the pub addr.

- [ ] Topics in /users are User IDs.
  - [ ] What about blocked Users?  Can anyone see when a user's bounty
        increases?
  - [ ] When a group notif token is updated, it will notify all users in
        the new group's authorized users of the new token, and their
		notif river will add the new topic and remove the old.
- [ ] Topics in /groups are Group IDs.  Can everyone see who is in a
      group?

AUTH PROBLEMS:
  1. Have to confirm users have unsubscribed before finishing
    1a. Some users might be offline.  Don't know how req/rep works.
  2. Have to either mutate or reconnect.
    2a. If you mutate, you need a mutable subscriber.
	2b. While you mutate or reconnect, you may miss updates.  Should it
	    wait for confirmations before posting new updates?  What does it
		do while they pile up?  Where does it buffer them?
  3. If users get a new token, they might miss some updates.  Then diffs
     can't work.
    3a. You could send diffs with IDs.  If the ID is wrong, you know you
	    need to refresh your cache of the object and catch up.
  4. This all leads me to think the complexity is higher than necessary.

Notif API:
- [ ] Subscribe to all publishers in /rivers/notifs
- [ ] Topics loaded for user from /streams/users/:user_id/
- [ ] Something with a notif publisher is created or updated
  - [ ] It calls streams.UpdateToken (returns old token if any)
    - [ ] The new Token is a BLAKE2 of the entity ID plus a random salt.
    - [ ] /streams/topics/:entity_id=>token is Put
	- [ ] /streams/users/:user_id/:old_token is replaced with :new_token
	      for each authorized user.
	- [ ] De-authorized Users have their Tokens deleted from their user
	      topics (/streams/users/:user_id/:old_token) so that new
		  subscriptions will not have the old topic.
  - [ ] The API notifies each authorized User on topic userID of the new
        Token and expired old token if any.
  - [ ] The API notifies any removed Users that the token has expired.
- [ ] A notif subscriber receives a Token update on his User topic.
  - [ ] The subscriber removes the old Token topic and creates a new one
  - [ ] The update is also sent over the websocket if group removal.
- [ ] When the API wants to publish a notification, it looks up the
      Token from /streams/topics/:entity_id.
- [ ] Notifications are specified by resource path (/things/:id)
- [ ] Notifications can contain diffs or whole values
- [ ] Notifications specify a type constant which tells them how to
      unmarshal
- [ ] Notifications offer a handle for REST API packages to share
  - [ ] Sender only does anything if user connected
  - [ ] Multiple connections per user OK
  - [ ] Sender duplicates message to user's connected sessions
  - [ ] Notifs use mangos Socket backend
- [ ] Notifications are well-tested
