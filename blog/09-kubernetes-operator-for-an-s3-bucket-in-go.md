# A Kubernetes Operator for an S3 Bucket in Go

A Kubernetes operator is a program that keeps something outside the cluster equal to an object inside it. In this
article, the object is a `ManagedBucket`, the thing outside is an S3 bucket, and the program is written in Go with
Operator SDK.

The happy path is short: create the object, then create the bucket. The real work is in the cases that go wrong. The
operator can crash in the middle of a step. Someone can change a bucket by hand. A bucket can already exist, or be full
of objects. The whole cluster can be lost. This article shows how the operator handles each case, and which questions
are still open.

## What This Article Covers

The operator is small on purpose, so every decision can be explained. It has one resource, one controller and one
storage service. It runs locally without an account or a paid service, because the S3 side is an open-source emulator.

The article is about design. It makes no claim about load or capacity, because none was measured.

## The Resource

A user writes this:

```yaml
apiVersion: storage.example.com/v1alpha1
kind: ManagedBucket
metadata:
  name: managedbucket-sample
spec:
  bucketName: managedbucket-sample-0001
  region: eu-central-1
  versioning: Enabled
  tags:
    team: platform
```

The name and the region cannot change. S3 cannot move a bucket, so a change would mean a different bucket. A CEL rule
on the CRD rejects such an edit at the API server, before the operator sees it. The versioning is `Enabled` or
`Suspended`, and `Suspended` is the default.

The CRD allows up to 50 tags. S3 allows 50 tags on a bucket, and the operator adds two of its own, so 48 are left for the
user.

The status has two conditions. `Ready` says whether the bucket matches the spec. `Deleting` says that removal is
running. The reason names the cause (`InSync`, `DriftCorrected`, `NotOwned`, `NotEmpty`, `AWSError`), and the message
carries the text of the error.

## Reconcile From the State, Not From the Event

A controller is called when something changes, but it should not care what changed. `Reconcile` reads the resource,
reads the bucket, and fixes only what differs. It never asks what happened before.

This one rule gives most of the reliability. If the operator crashes and restarts, it reconciles every resource again
and catches up. If a call is repeated, the second pass finds nothing to do. If an event is lost, the next pass still
sees the right picture. The operator keeps no state of its own.

The steps are these:

1. A resource that is being deleted goes to the deletion path. Any other resource gets a finalizer.
2. The bucket is read. A missing bucket is created, with its versioning and tags.
3. If the bucket exists and differs from the spec, the versioning and tags are written again.
4. After an interval, five minutes by default, the resource is reconciled again.

The last step is how drift is found. S3 has no change feed that fits a controller, so the operator asks. The interval is
a trade-off. A short interval means more calls to the storage service. A long interval means a change made by hand
lasts longer.

A pass is cheap when nothing changed: it makes three read calls. A pass that has to write reads the bucket again first,
and its own status update starts one more pass.

The drift was tested on a local kind cluster, with the interval set to 10 seconds. The versioning of a bucket was
changed directly in the emulator while the operator was running, and the operator set it back.

## The Operator Deletes Only What It Marked

The most dangerous thing an operator can do is delete a real resource that it does not own. So ownership is explicit.

When the operator creates a bucket, it adds two tags. One says which operator made it. The other holds the namespace and
name of the resource. The operator changes or deletes only a bucket that has both tags.

Sometimes a bucket with the requested name already exists in the same account, without these tags. The operator leaves
it alone, reports `Ready=False` with the reason `NotOwned`, and checks again later.

A bucket name is global in S3. If the name belongs to another account, S3 answers "forbidden". The operator reports this
as a storage error and retries with backoff. It does not report `NotOwned`.

The tags are a marker, not a proof. Anyone who can write the tags of a bucket can make the operator adopt it, and later
delete it. The tags also say nothing about the cluster. Two clusters in one account that use the same namespace and name
would fight over one bucket. The operator is meant to run in an account where its own permissions are the only ones that
write these tags, and one cluster uses one account.

Another way is to adopt an existing bucket. It is more convenient, but much less safe, because the operator cannot know
whether something else uses the bucket. If adoption is wanted, it should be an explicit annotation on the resource.

The tags have one more useful property: they live in the storage service, not in the cluster. When the cluster is lost,
the ownership survives. This comes up again below.

## Deletion With a Finalizer

When a user deletes the resource, Kubernetes removes it at once. The operator would never know that a bucket is left
behind. A finalizer fixes this. The operator puts its name on the resource, and Kubernetes keeps the resource until that
name is removed. The operator removes it only after the bucket is gone.

The deletion path has three rules:

- A bucket that is already gone is not an error. The resource is released.
- A bucket that the operator did not make is not touched. The resource is released.
- A bucket that still holds objects is **never emptied**. The operator reports `NotEmpty` and tries again every 30
  seconds.

The last rule is a choice. Emptying a bucket destroys data, and a user cannot undo a mistake in a delete loop. A stuck
deletion is visible in the status and in an event, and the user decides what to do. It is annoying, but it is safe.

## Failures Are Visible and Retried

Every error from the storage service does three things. It is written to the `Ready` condition, it becomes a Kubernetes
event, and it is returned from `Reconcile`.

Returning the error makes controller-runtime retry with exponential backoff. The wait starts at 5 milliseconds and
doubles, up to about 17 minutes. The S3 client library also retries a throttled or failed call by itself, up to three
attempts. Each step compares first, so a step that already worked is not repeated.

Counters show what happens over time. `managedbucket_outcomes_total` counts what each pass did: created, drift
corrected, not owned, not empty, deleted, storage error. They sit next to the standard controller-runtime metrics (queue
depth, reconcile time, errors) on the same endpoint. The endpoint is served over HTTPS with authentication.

## Ready for More Than One Replica

The deployment runs two replicas. Leader election, through a Kubernetes lease, lets only one of them work at a time. If
the leader dies, the other takes over in about 15 seconds, or a little more. On the local cluster it took 16 seconds.

The pods run as non-root with all capabilities dropped. The probes are `/healthz` and `/readyz`.

The Kubernetes permissions are generated from markers in the code, so they cannot drift away from what the code needs:
the resource, its status and finalizers, and events. On the storage side, the README lists the eight S3 actions to
allow, on the buckets that should be managed.

## A Small Package for S3

The reconciler does not import the S3 client library. It uses one small interface, with three methods, in a package
called `s3bucket`. One method reads a bucket, one writes it, and one deletes it. In the tests, a fake with a few lines of
code stands behind the interface. The SDK stays in one file.

The tests have three layers:

- The reconciler runs against a real API server (envtest) with the fake store. The cases are the ones in this article:
  creating, matching, correcting drift, a bucket that is not its own, deleting, a bucket that is not empty, a failure of
  the storage service and a retry after it.
- The client runs against a local HTTP server that answers with S3 errors. This checks how it reads "not found", "not
  empty" and "slow down".
- The client runs against a real emulator, Floci, in a container. The same test runs in the CI pipeline as a service.

Floci passes the test, is MIT licensed and needs no account, so the tests use it, with the image pinned by digest.

`make local-up` builds the image, creates a kind cluster, loads the image and deploys the operator with the emulator. It
needs Go, Docker, `kubectl` and kind, and nothing else.

## Versioning

The version is `v1alpha1`. Alpha means the fields may still change.

A change that only adds an optional field with a default can stay in the same version. A change that renames, removes or
re-types a field needs a new version, and that needs a conversion webhook. The webhook is described in the README and is
not built.

The CRD is never deleted with the operator, because deleting a CRD deletes every resource of it.

## Disaster Recovery

- **The operator is lost.** The buckets stay and keep working. New and changed resources wait. Deleted resources wait
  too, because their finalizer holds them. After a restart, the first pass over all resources catches up.
- **The cluster is lost.** The buckets stay in the storage service. Deleting a cluster does not delete its resources one
  by one, so no finalizer runs. In a new cluster, apply the same manifests from Git. A resource with the same namespace
  and name finds its bucket by the tags and takes it over. A resource with another name sees `NotOwned`, which protects
  the bucket from a mistake.
- **A bucket is lost or changed outside the cluster.** The next check creates it again, or restores the versioning and
  tags. The objects in a lost bucket come back only from versioning or replication, which are outside the operator.

The state to back up is the manifests, not the cluster.

## The Known Weakness

A bucket is created and tagged in two separate calls. If the operator crashes between them, the bucket exists without
tags. The operator refuses to touch it, because it cannot tell it from a stranger's bucket. The result is safe, but it
needs a manual step.

The first fix to build is a pending marker. The operator writes the marker into the status before the call. Then an
untagged bucket with a matching marker is known to be the operator's own.

This fix is in the README's list of next steps, with a few more: adoption by annotation, more settings such as
encryption and lifecycle rules, the second CRD version, and a run against a real S3 service.

## Three Things to Remember

1. **Reconcile from the state.** A pass that looks only at the resource and the world is safe to repeat, to lose and to
   restart.
2. **Make ownership explicit.** An operator should change only what it has marked as its own, know how strong that mark
   is, and say so when it refuses.
3. **Prefer a stuck deletion to a lost bucket.** Make the unsafe case visible and leave the decision to a person.

In short, the operator does little, and it can do it twice without harm. When it is unsure, it stops and says why.
