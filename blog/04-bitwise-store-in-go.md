# One-Method Finder & Bit Flags

Building a clean architecture in Go doesn't require massive frameworks. You can build a highly scalable, flexible system
using just one generic interface, a typed set of bit flags, and a clever stack of wrappers.

Let's break down how this works using a completely neutral domain: an `Item` that can have a few generic flags: `Foo`,
`Bar`, and `Baz`.

## 1. The Item and Its Bits

Imagine we have a simple `Item` struct that holds an ID, a name, and a numeric `Value`. Instead of adding boolean fields
like `IsFoo` or `IsBar`, we'll use the bit flag trick to store all its properties in a single `uint8` integer.

```go
// package data
type ItemTag uint8

const (
    FlagFoo ItemTag = 1 << iota // 1
    FlagBar                     // 2
    FlagBaz                     // 4
    FlagQux                     // 8
)

type Item struct {
    ID    uint64
    Tags  ItemTag
    Name  string
    Value int64
}
```

By using `1 << iota`, each flag gets its own distinct bit. Because `ItemTag` is a distinct custom type, the compiler
won't let you accidentally pass in a random number or a flag from a different table. You combine flags with OR (`|`),
test them with AND (`&`), and remove them with bit clear (`&^`).

## 2. One Interface to Rule Them All

We keep our database interactions ridiculously simple by defining a generic `Finder` interface with exactly one method:
`Retrieve`.

```go
type Finder[T any] interface {
    Retrieve(context.Context, T) (T, error)
}
```

Because it uses generics (`T any`), the query you send in and the result you get back are the exact same type. Calling
`Retrieve(ctx, data.Item{ID: 7})` basically says, "find the item that looks like this".

Our initial backend can just be a dead-simple in-memory map:

```go
// package cache
type Items struct {
    Rows map[uint64]data.Item
}

func (s *Items) Retrieve(_ context.Context, i data.Item) (data.Item, error) {
    row, ok := s.Rows[i.ID]
    if !ok {
        return data.Item{}, data.ErrNotFound
    }
    return row, nil
}
```

There is no `implements` keyword. The Go compiler just checks that `*cache.Items` has a `Retrieve` with the right
signature when we actually wire it up in `main`.

## 3. The Magic of Decorators

Instead of cramming all our business rules into one massive function, behavior lives in wrappers. A wrapper is just a
struct that holds an inner `Finder` while also acting as a `Finder` itself. It calls the inner finder, gets the data,
and then does its one specific job.

Let's say the `Foo` flag means the item gets a bonus added to its value:

```go
// package foo
type Items struct {
    Decorator data.Finder[data.Item]
    Bonus     int64
}

func (s *Items) Retrieve(ctx context.Context, i data.Item) (data.Item, error) {
    item, err := s.Decorator.Retrieve(ctx, i)
    if err != nil {
        return data.Item{}, err
    }

    // If the Foo bit is set, add the bonus!
    if item.Tags&data.FlagFoo != 0 {
        item.Value += s.Bonus
    }
    return item, nil
}
```

And let's say the `Bar` flag means the item's value gets reduced by a certain percentage:

```go
// package bar
type Items struct {
    Decorator data.Finder[data.Item]
    Penalty   int64
}

func (s *Items) Retrieve(ctx context.Context, i data.Item) (data.Item, error) {
    item, err := s.Decorator.Retrieve(ctx, i)
    if err != nil {
        return data.Item{}, err
    }

    // If the Bar bit is set, apply the penalty!
    if item.Tags&data.FlagBar != 0 {
        item.Value -= item.Value * s.Penalty / 100
    }
    return item, nil
}
```

Each wrapper reads exactly one bit and ignores the rest. The `foo` package doesn't know that `bar` exists, and neither
package knows if the data came from a map or a database. They stack together seamlessly, just like `bufio.NewWriter(w)`
and `gzip.NewWriter(w)` in the standard library.

## 4. Stacking It Up

When it's time to wire up your application, your API handler just asks for the one method it needs:

```go
// package api
type Handler struct {
    Decorator data.Finder[data.Item]
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    item, err := h.Decorator.Retrieve(r.Context(), data.Item{
        ID: id,
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(item)
}
```

And in `main`, you just use a single, nested struct literal to stack them like blocks:

```go
handler := &api.Handler{
    Decorator: &bar.Items{
        Decorator: &foo.Items{
            Decorator: &cache.Items{
                Rows: database,
            },
            Bonus: 150,
        },
        Penalty: 20,
    },
}
```

Read it from the inside out: load the row from memory, apply the `Foo` bonus, apply the `Bar` penalty, render the
result. The execution order is a clear, visible decision. Want to apply the penalty *before* the bonus? Just swap the
lines. Want to add a cache? Add another wrapper. No hidden constructors, just interfaces and exported fields.

## 5. Testing Without the Headache

Because of this stacked design, testing is a breeze. You don't need a mock generator:

```go
type fake data.Item

func (f fake) Retrieve(context.Context, data.Item) (data.Item, error) {
    return data.Item(f), nil
}

func TestFooBonus(t *testing.T) {
    s := &foo.Items{
        Decorator: fake{
            ID:    1,
            Value: 500,
            Tags:  data.FlagFoo,
        },
        Bonus: 150,
    }
    got, _ := s.Retrieve(context.Background(), data.Item{
        ID: 1,
    })
    if got.Value != 650 {
        t.Fatalf("got %d, want 650", got.Value)
    }
}
```

The fake is three lines. It just returns whatever the test puts into it, makes no assertions about how it was called,
and never breaks because another test needed a different setup.

## 6. Growing Pains? What Growing Pains?

As your application grows, this shape stays exactly the same:

* **A third rule arrives:** Let's say `Baz` items cannot be modified. You just create a `baz` package that checks
  `FlagBaz` and skips the other math. You add one more line to the literal stack in `main`. The `foo` and `bar` packages
  aren't even opened.
* **Moving to a database:** The in-memory map becomes a SQL table. `database.Items` implements the exact same single
  method, building a `WHERE` clause from the query fields. Only the innermost line of the literal in `main` changes. The
  rules, handler, and tests are entirely unaffected.
* **Integration Tests:** A test that needs the whole stack just builds the same literal that `main` builds, putting the
  `cache.Items` fake at the bottom, calls the handler, and reads the JSON. No test database containers required.

Every package imports `data` and nothing else. The `data` package imports nothing. `main` is the only file that knows
all the concrete types. The dependency graph keeps its beautiful, simple shape no matter how many wrappers you add!
