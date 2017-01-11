# Task Design

- Task is the minimal viable utility I can immediately gain benefit from
  while still being future-usable; maximally extensible / pluggable
- That means it's completely unintelligent and simple with an easy API
  for internal and external users; if clients want more data, expose it
  later

- Bare minimum:
  - ID string UUID
  - Owner / creator
  - Readers / writers
  - Name
  - Bounty
  - Due
  - Notes (as ref(s?) with own bucket)
  - Children (refs) -- how to avoid loops?  Assume client handles this?
    - "IsLoop" walks task chain?

  - CRUD notifs
  - Bounty / profile / etc notifs
  - Completion saves to "completed" list
  - Global list of tasks, when you search it uses filters similar to
    Streams

- Next steps:
- Think about conditioning focus
- Task accrues points?  User accrues points?  User has some work ledger?
  - When a user wants to work on a task, maybe it clones the task into
    a personal folder; when the task is delivered, then it can update
    the original
- Task is built to support intelligence first: the whole point is
  scheduling.
- Projects are entities in their own right; users come and go.
  - A user has a task view / can mutate their copy, but the owner
    decides what qualifies as delivered.
- Tasks separate from productivity / pom view?
- Pom integration?
- Task dep chains?
  - "Precompile" in scheduler client (MF kernel?)
  - Serialization occurs here so multiple users can add tasks?
    - Scenario: two users try to merge two compatible task trees.
    - Scenario: two users try to merge two incompatible task trees.
      - Deserialize task array into tree?  Array representation of tree?
- Tasks can have bounties.
- Tasks can have due dates.
- Tasks can have manual priority levels.
- Tasks can have calculated "criticality" (value / deps / due date etc.)
- Task API simple; intelligence added behind the scenes
- What happens if updating the task state mutates other states?
  - What states can task changes touch?
    - User bounty
    - Intelligence state
    - Other tasks priority?
      - Maybe priority should be calculated on the fly?  By the client?
      - How would the client know priority?
      - Wouldn't it be easier to precompute on mutation?
      - Could there be an on-the-fly priority estimation?
      - Deferred internal jobs?
      - Ongoing internal jobs?  (Walk priority tree, updating?)
      - Some things can be in flux; other things must be atomic
      - Things which lots of people could interact with should be fluxy
      - Things which MUST be exact should not
    - Other internal API users could use the task API to do their own
      thinking about tasks, then task could just be simple data
      - This meshes well with the kernel queue concept
    - Deferred computation could open a websocket to stream results

## Task scheduler design

- Consider a kernel scheduler.  The scheduler does not create its own
  queue; rather, the user adds jobs to the queue.  It assumes the user
  can schedule its own tasks correctly (i.e. there is no concept of
  task dependency chains.  These are managed by kernel users.)
- The CPU itself also has an intelligent scheduler and a "task" queue;
  the CPU manages its own pipelining.
- Task dependency chains are managed at compile time or by a VM; they
  are either precompiled into sequential instructions, or the VM
  processes them into sequential instructions (e.g. stack-based.)
- Much of this is necessary due to the requirement of immediacy.
- LLVM bytecode can be precompiled or JIT-compiled.
