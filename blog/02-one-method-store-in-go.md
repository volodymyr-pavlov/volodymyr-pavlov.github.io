# One Interface, One Method

Building a clean data access layer in Go doesn't have to turn into a sprawling mess. By just using a little bit of
generic magic, you can keep things incredibly simple and rely on a single method.

Here is how this minimalist approach works:

*   Instead of writing custom functions for every lookup, you just use the data row itself as your query.
*   Any backend that supports this single action fits right in—whether that is a simple memory map, a full database, or
    a tiny fake for testing.
*   All that extra clutter that usually builds up in a repository over the years? It is almost always just database
    filtering logic in disguise, or it deserves to be broken out into its own separate, single-action interface.

Let's break down exactly why traditional patterns grow out of control and how sticking to just one method fixes the
problem.

## The Repository That Grew

Imagine you're building a user service for an API. Users have an email, a name, and a role. The initial developer
sets up a repository interface like this:

```go
// package database
type UserRepository interface {
    GetByID(ctx context.Context, id uint64) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
    GetByRole(ctx context.Context, role string) ([]*User, error)
}
```

It starts simple: three methods on day one. But then the requirements start piling up. Support needs to look up users by
phone number, so `GetByPhone` is added. Billing needs users on the pro plan, adding `GetByPlan`. Compliance needs
deleted users in a report, adding `GetByIDIncludeDeleted`.

Fast forward a few years, and you have an interface with dozens of methods. Every time you add a new lookup, you have to
update the interface, the database implementation, the in-memory fake, the generated mock, and every single decorator.
A simple pull request touches five different files.

## The One Method Alternative

There's a cleaner way. Instead of an ever-growing list of methods, you can define a single, generic contract:

```go
type Finder[T any] interface {
    Retrieve(context.Context, T) (T, error)
}
```

This method is generic over the row type (`T`). The brilliant part is that the row type is used as *both* the argument
(the query) and the result.

When you want to find a user, you simply pass in a struct with the known fields filled out:

```go
user, err := s.Retrieve(ctx, data.User{
    ID: 42,
})
```

You're essentially telling the store: "Find the user that looks like this."

### Handling the Logic

The backend implementation decides what that query means. For a simple in-memory map, it just means looking up the key:

```go
// package inmem
type Users struct {
    Rows map[uint64]data.User
}

func (s *Users) Retrieve(_ context.Context, u data.User) (data.User, error) {
    row, ok := s.Rows[u.ID]
    if !ok {
        return data.User{}, data.ErrNotFound
    }
    return row, nil
}
```

For a SQL database, the implementation uses a `switch` statement to build the appropriate `WHERE` clause based on which
fields are populated in the query struct:

```go
// package database
type Users struct {
    DB *sql.DB
}

func (s *Users) Retrieve(ctx context.Context, u data.User) (data.User, error) {
    q := `select id, email, name, role from users where `
    var arg any

    // The switch statement handles the dispatch
    switch {
    case u.ID != 0:
        q, arg = q+`id = $1`, u.ID
    case u.Email != "":
        q, arg = q+`email = $1`, u.Email
    default:
        return data.User{}, data.ErrNoArguments
    }

    var out data.User
    err := s.DB.QueryRowContext(ctx, q, arg).Scan(&out.ID, &out.Email, &out.Name, &out.Role)
    if errors.Is(err, sql.ErrNoRows) {
        return data.User{}, data.ErrNotFound
    }
    return out, err
}
```

This `switch` statement is where all those old `GetBy` methods went! If you suddenly need to look up users by phone
number, you just add one more `case` to this function. You don't have to touch the interface, the in-memory backend, or
any wrappers.

The order of the cases determines priority (e.g., ID wins over email), which keeps the logic centralized in the store
rather than scattered across your application.

## The Big Wins

This approach gives you two massive advantages:

1.  **Write It Once:** Because the interface is generic, you declare it one time. `Finder[User]`, `Finder[Session]`, and
    `Finder[Order]` all use the exact same interface. This means you can write a generic logging wrapper or cache *once*
    and use it for every table in your system.
2.  **Trivial Fakes:** You don't need a clunky mock generator anymore. A test double for `Finder[User]` is literally
    three lines:

```go
type fake data.User

func (f fake) Retrieve(context.Context, data.User) (data.User, error) {
    return data.User(f), nil
}
```

This fake just returns whatever you put into it. It doesn't break if another test needs a different setup, keeping your
tests fast and robust.

## Wiring It Together

When it's time to build your API handler, you just declare the `Finder` interface as a dependency:

```go
// package api
type Handler struct {
    Users data.Finder[data.User]
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Call the single method with the query struct
    user, err := h.Users.Retrieve(r.Context(), data.User{
        ID: id,
    })

    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}
```

Notice how clean the dependency graph is:
*   The `api` package imports `data`.
*   The `database` package imports `data`.
*   The `data` package imports nothing.

The `api` package has no idea which database is behind it. The only file that wires everything together is your `main`
application file. And because Go checks interface implementation implicitly, there's no `implements` keyword
anywhere—the compiler just checks that the method signatures match when you assemble the application in `main`.
