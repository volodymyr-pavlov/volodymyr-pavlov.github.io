# Handling Transactions

So, you've got your single-method stores working beautifully. But what happens when you need to touch *two* tables at
once? Say you need to deduct credits from a user's account and create an order at the exact same time. You definitely
can't have one succeed and the other fail—that's how you end up with missing money or free products! You need a
transaction.

Here is how we keep our clean, one-method architecture while wrapping everything up safely in a SQL transaction.

## Swapping Context for a Transaction

Our store interfaces keep their exact same shape—one generic method, using the row as the query. The only difference?
Instead of passing a `context.Context`, we pass a `*sql.Tx`.

Why? Because a transaction object already carries its own context and holds onto the database connection.

```go
type TxFinder[T any] interface {
    RetrieveTx(*sql.Tx, T) (T, error)
}

type TxLocker[T any] interface {
    LockTx(*sql.Tx, T) (T, error) // select ... for update
}

type TxUpdater[T any] interface {
    UpdateTx(*sql.Tx, T) error
}

type TxCreator[T any] interface {
    CreateTx(*sql.Tx, T) (T, error)
}
```

Your database package implements these by using the transaction to run the queries. Meanwhile, an in-memory fake can
just ignore the transaction argument entirely and update its internal map. Easy!

## The Business Rule is Just a Struct

Let's build that ordering system. To place an order, we need to interact with the database itself, plus three specific
store actions: locking an account, updating an account, and creating an order.

Instead of scattering this logic, we create a single struct (our "rule") to hold exactly what we need:

```go
type Ordering struct {
    DB       *sql.DB
    Accounts data.TxLocker[data.Account]
    Update   data.TxUpdater[data.Account]
    Orders   data.TxCreator[data.Order]
}
```

## The "Do Everything" Method

Now, we expose exactly one method on this struct. This method's job is to open the transaction, orchestrate the steps,
and then safely commit (or roll back if things go sideways).

```go
func (o *Ordering) Place(ctx context.Context, accountID uint64, amount int64) (data.Order, error) {
    // 1. Open the transaction
    tx, err := o.DB.BeginTx(ctx, nil)
    if err != nil {
        return data.Order{}, err
    }

    // 2. Defer the rollback! If the commit at the bottom succeeds, this does nothing.
    // If we hit an error and return early, this safely undoes any changes.
    defer tx.Rollback()

    // 3. Lock the account (SELECT FOR UPDATE)
    account, err := o.Accounts.LockTx(tx, data.Account{
        ID: accountID,
    })
    if err != nil {
        return data.Order{}, err
    }

    // 4. Check the business logic
    if account.Balance < amount {
        return data.Order{}, ErrInsufficientBalance
    }

    // 5. Deduct the money
    err = o.Update.UpdateTx(tx, data.Account{
        ID:      accountID,
        Balance: account.Balance - amount,
    })
    if err != nil {
        return data.Order{}, err
    }

    // 6. Create the order
    order, err := o.Orders.CreateTx(tx, data.Order{
        AccountID: accountID,
        Amount:    amount,
    })
    if err != nil {
        return data.Order{}, err
    }

    // 7. Commit the transaction!
    return order, tx.Commit()
}
```

## The Caller's View

The best part about this setup? The caller is completely shielded from the transaction mechanics. When it's time to wire
this up, you just assemble the struct:

```go
ordering := &Ordering{
    DB:       db,
    Accounts: &database.Accounts{},
    Update:   &database.Accounts{},
    Orders:   &database.Orders{},
}

order, err := ordering.Place(ctx, accountID, amount)
```

The struct holds your stores, exposes one clean action, and manages its own state safely. People usually call this a
"service layer," but in our world, it's just another wrapper. It composes a few stores together and does *one specific
thing*, rather than turning into a bloated, monolithic mirror of your database!