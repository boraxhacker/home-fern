# Datastore Refactor Analysis

## Current Situation

The project currently uses a datastore for its backend. The initial implementation used SQLite, but it was found to have issues with concurrent requests. The project then moved to BadgerDB.

## Requirements

The requirements for the datastore are:

*   **Lightweight:** The datastore should be simple and not require a separate server process.
*   **No external dependencies:** The datastore should be a library that can be embedded in the Go application.
*   **Concurrency:** The datastore must be able to handle concurrent read and write requests from the web application.
*   **Simplicity:** For a home lab project, a simpler solution is preferable.

## Datastore Options

### SQLite (Previous Implementation)

*   **Concurrency Issues:** The observed behavior where a second request would wait for the first to finish is typical for SQLite under concurrent write scenarios. SQLite uses file-level locking, meaning that during a write operation (INSERT, UPDATE, DELETE, DDL), an exclusive lock is often acquired on the entire database file. This blocks other operations, including reads, until the write transaction commits. While `database/sql` in Go handles connection pooling, the underlying SQLite driver's locking mechanism serializes write operations, making it unsuitable for applications requiring high concurrent write throughput.

### bbolt (formerly BoltDB)

*   **Pros:**
    *   Pure Go, easy to embed.
    *   ACID transactions.
    *   Simple API.
    *   Well-regarded in the Go community. Core of etcd.
*   **Cons:**
    *   Single writer, multiple readers. A long-running write transaction can block other writes and reads. This might be a problem for a web application if write transactions are frequent or long-lived.

### BadgerDB

*   **Pros:**
    *   Pure Go, easy to embed.
    *   ACID transactions.
    *   Designed for high performance, especially for write-heavy workloads.
    *   Supports concurrent reads and writes.
*   **Cons:**
    *   More complex API than bbolt.
    *   Higher memory usage compared to bbolt.
    *   The underlying LSM-tree design can lead to write amplification, which might be a concern for some use cases (but probably not for a home lab).

## Analysis

The user mentioned that `bbolt` might be a better choice than `badgerdb`. Let's analyze this.

The key difference is the concurrency model. `bbolt` has a single writer, while `badgerdb` supports concurrent writers. For a web application, concurrent writes are a common scenario. If a user request triggers a write to the database, and multiple users are using the application, then a single-writer model can become a bottleneck.

However, for a "home lab" project, the number of concurrent users is likely to be very low. If write transactions are short and infrequent, the single-writer model of `bbolt` might be perfectly acceptable. The simplicity of `bbolt`'s API and its lower resource usage could be a significant advantage in this context.

Given the user's preference for `bbolt` and the context of a home lab project, it's reasonable to assume that the write load is not going to be a major issue. The simplicity of `bbolt` is a strong argument in its favor.

## Recommendation

I recommend proceeding with **bbolt**.

While BadgerDB offers better performance for write-heavy workloads, its complexity might be overkill for this project. The simplicity of bbolt, both in its API and its operational model, makes it a better fit for a home lab project where ease of maintenance and understanding is important.

The single-writer limitation of bbolt should be kept in mind. If the application evolves to have more complex and long-running write operations, this decision might need to be revisited. For now, it seems like a reasonable trade-off.

## Refactoring Plan

This plan outlines the steps to replace BadgerDB with bbolt, using a single database file with separate buckets for each service as requested.

### Phase 1: Create a Centralized bbolt Datastore Service

1.  **Create a new package:** `internal/datastore`.
2.  **Implement `datastore.go`:**
    *   This file will contain the core bbolt logic.
    *   It will manage a single database connection (`*bbolt.DB`).
    *   **`New(dbPath string) (*Datastore, error)`**: Initializes the bbolt database file.
    *   **`Close()`**: Closes the database connection.
    *   **`View(bucketName string, fn func(b *bbolt.Bucket) error) error`**: A generic wrapper for read-only transactions.
    *   **`Update(bucketName string, fn func(b *bbolt.Bucket) error) error`**: A generic wrapper for read-write transactions. It will create the bucket if it doesn't exist.

### Phase 2: Refactor `ssm` Service

1.  **Modify `internal/ssm/datastore.go`:**
    *   Rename the existing `Datastore` struct to `BadgerDatastore` (or similar) temporarily.
    *   Create a new `Datastore` struct that holds a reference to the new `internal/datastore.Datastore`.
    *   Rewrite all data access methods (`FindParameter`, `InsertParameter`, etc.) to use the new `datastore.View()` and `datastore.Update()` methods, operating on a bucket named `"ssm"`.
    *   Data will be marshaled to JSON before being stored in bbolt.

2.  **Update `internal/ssm/service.go`:**
    *   Adjust the `Service` struct and its initialization to use the new `ssm.Datastore`.

### Phase 3: Refactor `route53` Service

1.  **Locate and Modify `route53` Datastore:**
    *   Identify the file containing the BadgerDB implementation for Route53 (likely `internal/route53/datastore.go`).
    *   Perform the same refactoring as in Phase 2, creating a new `Datastore` that uses the central `internal/datastore` service.
    *   All operations will use a bucket named `"route53"`.

### Phase 4: Update Application Entrypoint

1.  **Modify `cmd/server/main.go` (or equivalent):**
    *   Remove initialization logic for separate BadgerDB instances.
    *   Add initialization for the new central `datastore.Datastore`.
    *   Pass the single `datastore.Datastore` instance to the `ssm` and `route53` service constructors.

### Phase 5: Dependency Management and Cleanup

1.  **Update `go.mod`:**
    *   Run `go get go.etcd.io/bbolt`.
    *   Run `go mod tidy` to remove the `github.com/dgraph-io/badger/v4` dependency.
2.  **Remove Old Code:**
    *   Delete the old BadgerDB-specific datastore files and any other now-unused code.

### Phase 6: Data Migration Strategy

Since we are changing the underlying database format, existing data must be migrated. We will use the existing export/import feature.

1.  **Before Implementation:** With the current application running, call the `/export/all` endpoint to download a `home-fern-export.zip` file containing all data from BadgerDB.
2.  **After Implementation:** Once the refactoring is complete and the new bbolt-powered application is running:
    *   The `importAll` function in `internal/dbfcns/api.go` will implicitly use the new bbolt-backed datastores.
    *   Use the `/import/all` endpoint to upload the `home-fern-export.zip` file.
    *   The application will read the JSON from the zip and write it into the bbolt buckets, effectively migrating the data.
    *   No changes to the import/export API logic should be necessary, as it already operates on the abstracted service layer.
