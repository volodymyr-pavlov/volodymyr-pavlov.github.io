# Bitwise in Go: Doing More with Less

Hey there! Did you know you can stash a ton of boolean values, states, and features inside just *one* integer? By giving
each property its own bit, you can save space and simplify your code. This article looks at how to pull this off in
Go, and compares it against the usual way of cramming booleans and strings into a struct.

## Where You've Seen This Before: File Permissions

You've probably seen this concept in action before. When you run `ls -l` in your terminal, you get something that looks
like this:

```
-rwxr-xr-x  1 user  staff  4096 Sep  9 10:12 build
```

Those ten characters at the start actually represent a single integer! The owner, the group, and everyone else each get
three bits: read, write, and execute. `rwxr-xr-x` translates to `111 101 101` in binary. In octal, that's `755`—yep, the
exact same number you type when you run `chmod 755`.

Opening a file in Go uses this exact same trick:

```go
os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
```

See those pipe characters (`|`)? They combine three separate flags into one argument. Under the hood, Go uses an AND
operation (`flags & os.O_CREATE != 0`) to check if a specific flag is set.

**The golden rule:** Combine things with OR (`|`), test them with AND (`&`).

Go gives you `os.FileMode` for this, which is just a `uint32` under the hood. Its methods use these exact bitwise
operations. It's a simple, elegant pattern: use an integer, name a constant for each bit, OR them to combine, and AND
them to test. The same pattern works for custom Go types.

## The Standard Way

Picture a user account system. A user has a role, a subscription plan, some basic flags (like if
they are verified), a list of active features, and an account state. Usually, the Go struct looks a bit like
this:

```go
const (
    RoleGuest  = "guest"
    RoleMember = "member"
    RoleAdmin  = "admin"

    PlanFree = "free"
    PlanPro  = "pro"

    StateInvited   = "invited"
    StateActive    = "active"
    StateSuspended = "suspended"

    AwaitingEmail   = "email"
    AwaitingReview  = "review"
    AwaitingPayment = "payment"
)

type User struct {
    Role        string
    Plan        string
    IsVerified  bool
    IsTwoFactor bool
    IsBeta      bool
    Features    []string
    State       string
    Awaiting    string
}
```

Updating this user is pretty standard: you set properties, check them with an `if` statement, and maybe flip a boolean.
But what happens when you have a complex rule? You end up writing a whole function for it:

```go
func canExport(u User) bool {
    return (u.Role == RoleMember || u.Role == RoleAdmin) &&
        u.Plan == PlanPro &&
        slices.Contains(u.Features, "bar")
}
```

And what about a background worker updating users? You'd loop through, check a bunch of string fields, and update the
state. The catch? If two goroutines try to update the same user at once, they'll overwrite each other unless you wrap
the whole thing in a clunky mutex lock.

This works, but it's a bit messy. You're juggling three different field types (strings, bools, slices), writing custom
functions for every business rule, risking runtime crashes from typos (like typing `"Admin"` instead of `"admin"`), and
dealing with locking overhead.

## The Cool Way

Now for the magic of doing this all with one integer. A `UserFlag` type is defined as a `uint32`, and `iota` sets up
the bits:

```go
type UserFlag uint32

const (
    // Role.
    UserGuest UserFlag = 1 << iota
    UserMember
    UserAdmin

    // Plan.
    UserFree
    UserPro
    UserEnterprise

    // Account.
    UserVerified
    UserTwoFactor
    UserBeta
    UserNewsletter

    // Features.
    UserFeatureFoo
    UserFeatureBar
    UserFeatureBaz
    UserFeatureQux

    // State.
    UserInvited
    UserActive
    UserSuspended
    UserDeleted

    // Waiting for something.
    UserAwaitingEmail
    UserAwaitingReview
    UserAwaitingPayment
    // Eleven bits are still free!
)
```

`iota` is Go's handy automatic counter. `1 << iota` takes the number 1 and shifts it to the left by that count. This
gives powers of two: 1, 2, 4, 8, 16, and so on. This ensures every single constant gets its very
own, unique bit.

The custom `UserFlag` type also brings strict type safety. The compiler will yell at you if you try to pass a
random integer or misspell `UserAdmin`.

One thing the type does not enforce is exclusivity: nothing stops a user from having both `UserGuest` and `UserAdmin`
set. Group the mutually exclusive bits under a mask, such as `userRoles = UserGuest | UserMember | UserAdmin`, and
change them in one step with `flags = flags&^userRoles | UserAdmin`.

Manipulating them takes just three operators:

* **Turn it on with OR (`|`)**: Combines flags together.
  `flags := UserMember | UserPro | UserVerified`
* **Check it with AND (`&`)**: Sees if a specific bit is active.
  `if flags & UserTwoFactor != 0 { /* do something */ }`
* **Turn it off with AND NOT (`&^`)**: Clears specific bits while leaving the rest completely untouched.
  `flags &^= UserBeta`

Checking multiple flags is a breeze, too. Want to see if a user has *all* required permissions? Compare the AND result
to the required set. Just want to see if they have *any* of them? Check if the AND result isn't zero:

```go
want := UserMember | UserPro
isMemberPro := flags&want == want // Do they have both member AND pro?

oneOf := UserFeatureBar | UserFeatureBaz
isBarOrBaz := flags&oneOf != 0 // Do they have bar OR baz?
```

Suddenly, business rules aren't messy, sprawling functions anymore. They're just values stored in a simple map!

```go
var permissions = map[string]UserFlag{
    "export":  UserMember | UserPro | UserFeatureBar,
    "delete":  UserAdmin,
    "publish": UserBeta | UserFeatureFoo,
}

func allowed(flags UserFlag, action string) bool {
    want, ok := permissions[action]
    return ok && flags&want == want
}
```

Your background worker loop becomes a single `&` check, and updating the user is a neat one-liner that clears the old
state and sets the new one—without accidentally changing their role or plan:

```go
wants := UserAdmin | UserPro | UserAwaitingReview
for _, user := range users {
    if user.Flags&wants == wants {
        review(user)
    }
}

// Clear the review flag and set the payment flag
user.Flags = user.Flags&^UserAwaitingReview | UserAwaitingPayment
```

### The Best Parts

* **No mutex for the flags.** You can use `sync/atomic` and `CompareAndSwap` to safely update the integer across multiple
  goroutines. If something changed while you were working, it just tries again.
* **JSON works out of the box.** When you serialize your struct, `encoding/json` just outputs a clean, simple number
  like `"flags": 1106`.
* **You can make it readable.** You can easily write a custom `String()` method that reads the bits and prints out
  something human-readable for your logs, like `member|pro|verified|foo`.

And that's it! By letting bits do the heavy lifting, your code stays fast and strictly typed, and updating the flags needs no lock.
