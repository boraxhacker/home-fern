# Enhancing Error Handling in home-fern

This document outlines the current state of error handling in the `home-fern` project and proposes a plan to refactor it towards a more robust, standard, and maintainable approach that is compatible with AWS API conventions.

## 1. Current State of Error Handling

The current error handling mechanism was influenced by older coding styles and has several drawbacks that make it brittle and hard to maintain.

### Key Characteristics:

*   **Custom `ErrorCode` Type:** The project primarily uses a custom integer type, `core.ErrorCode`, to represent errors (e.g., `core.ErrNone`, `core.ErrNotFound`, `core.ErrInternalError`). Functions return this integer instead of the standard `error` type.
*   **Loss of Context:** When a lower-level function returns an `ErrorCode`, the original error from the underlying library (like a database driver) is often logged but otherwise discarded. This loses valuable context, such as the specific database error message, which makes debugging difficult.
*   **Manual Error Translation:** Each data layer has functions like `translateBadgerError` or `translateBboltError` that manually map specific library errors to a `core.ErrorCode`. This requires boilerplate and is not easily extensible.
*   **Brittle String Comparisons:** As identified, the code contains error checks that rely on matching the exact string output of an error, like `err.Error() == "bucket Route53 not found"`. This is highly fragile; if the underlying library changes its error message, the check will fail silently.

### AWS API Compatibility Goal:

The project aims to mimic AWS APIs. The AWS SDK for Go (v2) uses the standard `error` interface. It defines specific error types for different API failures (e.g., `types.NoSuchKey`, `types.InvalidParameterException`). Consumers of the SDK check for these errors using `errors.As()`. The current `ErrorCode` system is incompatible with this approach.

## 2. Proposed Refactoring Plan

The goal is to adopt modern, idiomatic Go error handling (Go 1.13+) using the standard `error` interface, error wrapping, and the `errors` package.

### Phase 1: Establish Standard Error Values

We will define a set of standard, exported error variables in the `core` package. These will replace the `ErrorCode` enum.

**File: `internal/core/errors.go`**
```go
package core

import "errors"

// Standard application errors
var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrInternal      = errors.New("internal server error")
	// ... other general errors
)
```

### Phase 2: Refactor Function Signatures

All functions throughout the application that currently return a `core.ErrorCode` will be refactored to return a standard `error`.

*   **Before:** `func (ds *dataStore) getParameter(key string) (*ParameterData, core.ErrorCode)`
*   **After:** `func (ds *dataStore) getParameter(key string) (*ParameterData, error)`

A successful operation will now return `nil` for the error, which is the standard Go convention, replacing `core.ErrNone`.

### Phase 3: Embrace Error Wrapping

Instead of discarding the original error, we will "wrap" it to add context. This preserves the full error chain for better debugging and programmatic inspection. The `%w` verb in `fmt.Errorf` is used for this.

**Example in `ssm/datastore.go`:**
```go
// Before
func (ds *dataStore) getParameter(key string) (*ParameterData, core.ErrorCode) {
    // ...
    err := ds.ds.View(...)
    if err != nil {
        return nil, translateBboltError(err) // Original 'err' is lost
    }
    return &param, core.ErrNone
}

// After
func (ds *dataStore) getParameter(key string) (*ParameterData, error) {
    var param ParameterData
    err := ds.ds.View(datastore.Ssm, func(b *bbolt.Bucket) error {
        v := b.Get([]byte(key))
        if v == nil {
            // Return our standard error
            return core.ErrNotFound
        }
        if err := json.Unmarshal(v, &param); err != nil {
            // Wrap the JSON error with our own context
            return fmt.Errorf("failed to unmarshal parameter %q: %w", key, err)
        }
        return nil
    })

    if err != nil {
        // Wrap the datastore error
        return nil, fmt.Errorf("could not get parameter %q: %w", key, err)
    }
    return &param, nil
}
```

### Phase 4: Use `errors.Is` and `errors.As` for Checks

We will replace all integer and string-based error checks with the standard `errors` package functions.

*   **`errors.Is(err, target)`:** Use this to check if an error in the chain matches a specific sentinel error value (like `core.ErrNotFound`). This is perfect for checking against the variables defined in Phase 1.
*   **`errors.As(err, target)`:** Use this to check if an error in the chain is of a specific type, and to extract it. This is how we will check for AWS-specific error types.

**Example in a service or API layer:**
```go
// Before
_, errCode := service.dataStore.getParameter(name)
if errCode == core.ErrParameterNotFound {
    // Handle not found
}

// After
_, err := service.dataStore.getParameter(name)
if errors.Is(err, core.ErrNotFound) {
    // Handle not found
}
```

### Phase 5: Implement AWS API-Compliant Error Responses

The final step is to translate our internal Go errors into the XML error responses that the AWS CLI and SDKs expect. This happens at the outermost layer of the application—the HTTP handlers in the `api.go` files.

1.  **Define an Error Response Struct:** Create a struct that can be marshaled into the AWS error XML format.

    ```go
    // In a package like `awslib`
    type AwsErrorResponse struct {
        XMLName xml.Name `xml:"Error"`
        Code    string   `xml:"Code"`
        Message string   `xml:"Message"`
    }
    ```

2.  **Create an Error Handling Middleware/Helper:** This function will inspect the error returned from the service layer and write the appropriate HTTP response.

    ```go
    // In a package like `awslib`
    func WriteAwsError(w http.ResponseWriter, r *http.Request, err error) {
        // Default to internal server error
        httpStatus := http.StatusInternalServerError
        awsError := AwsErrorResponse{Code: "InternalFailure", Message: "An internal error occurred."}

        if errors.Is(err, core.ErrNotFound) {
            httpStatus = http.StatusNotFound
            awsError.Code = "NotFound" // Or a more specific AWS code like "ParameterNotFound"
            awsError.Message = "The requested resource was not found."
        } else if errors.Is(err, core.ErrInvalidInput) {
            httpStatus = http.StatusBadRequest
            awsError.Code = "InvalidParameterValue"
            awsError.Message = err.Error() // Use the error's message for more detail
        }
        // ... add more mappings

        w.WriteHeader(httpStatus)
        w.Header().Set("Content-Type", "application/xml")
        if xmlErr := xml.NewEncoder(w).Encode(awsError); xmlErr != nil {
            // Fallback if XML encoding fails
            http.Error(w, "Failed to serialize error response", http.StatusInternalServerError)
        }
    }
    ```

By following this plan, we will create an error handling system that is robust, easy to debug, and aligns perfectly with both Go best practices and the requirements of the AWS ecosystem.
