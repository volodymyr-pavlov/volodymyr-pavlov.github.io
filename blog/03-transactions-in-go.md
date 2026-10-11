# Handling Transactions in Go

So, you've got your single-method stores working beautifully. But what happens when you need to touch *two* tables at
once? Say a player buys a sword in your game's shop: you need to take their gold and put the sword in their inventory at
the exact same time. You definitely can't have one succeed and the other fail—that's how players lose their gold for
nothing, or walk away with a free legendary sword! You need a transaction.

Here is how the clean, one-method architecture survives wrapping everything up safely in a SQL transaction.

## Swapping Context for a Transaction

The store interfaces keep their exact same shape—one generic method, using the row as the query. The only difference?
Instead of a `context.Context`, they take a `*sql.Tx`.

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
just ignore the transaction argument entirely and update its internal map. Easy! The catch is that such a fake has no
rollback, so tests against it cannot prove that a failed purchase leaves the player's gold untouched. Test that part
against a real database.

## The Business Rule is Just a Struct

Time to build that shop. To sell an item, the rule needs the database itself, plus three specific store actions:
locking a player, updating a player, and adding an item to their inventory.

Instead of scattering this logic, a single struct (the "rule") holds exactly what is needed:

```go
type Shop struct {
    DB        *sql.DB
    Players   data.TxLocker[data.Player]
    Update    data.TxUpdater[data.Player]
    Inventory data.TxCreator[data.Loot]
}
```

## The "Do Everything" Method

Now, the struct exposes exactly one method. This method's job is to open the transaction, orchestrate the steps,
and then safely commit (or roll back if things go sideways).

```go
func (s *Shop) Buy(ctx context.Context, playerID, itemID uint64, price int64) (data.Loot, error) {
    // 1. Open the transaction
    tx, err := s.DB.BeginTx(ctx, nil)
    if err != nil {
        return data.Loot{}, err
    }

    // 2. Defer the rollback! If the commit at the bottom succeeds, this does nothing.
    // If an error causes an early return, this safely undoes any changes.
    defer tx.Rollback()

    // 3. Lock the player (SELECT FOR UPDATE), so a second purchase can't spend the same gold
    player, err := s.Players.LockTx(tx, data.Player{
        ID: playerID,
    })
    if err != nil {
        return data.Loot{}, err
    }

    // 4. Check the business logic
    if player.Gold < price {
        return data.Loot{}, ErrNotEnoughGold
    }

    // 5. Spend the gold. UpdateTx writes the whole row, so it gets the full player,
    // not a partial struct whose zero fields would be saved as real values.
    player.Gold -= price
    err = s.Update.UpdateTx(tx, player)
    if err != nil {
        return data.Loot{}, err
    }

    // 6. Put the item in the inventory
    loot, err := s.Inventory.CreateTx(tx, data.Loot{
        PlayerID: playerID,
        ItemID:   itemID,
    })
    if err != nil {
        return data.Loot{}, err
    }

    // 7. Commit the transaction!
    if err := tx.Commit(); err != nil {
        return data.Loot{}, err
    }
    return loot, nil
}
```

## The Caller's View

The best part about this setup? The caller is completely shielded from the transaction mechanics. When it's time to wire
this up, you just assemble the struct:

```go
shop := &Shop{
    DB:        db,
    Players:   &database.Players{},
    Update:    &database.Players{},
    Inventory: &database.Inventory{},
}

sword, err := shop.Buy(ctx, playerID, swordID, 300)
```

The struct holds your stores, exposes one clean action, and manages its own state safely. People usually call this a
"service layer," but here, it's just another wrapper. It composes a few stores together and does *one specific
thing*, rather than turning into a bloated, monolithic mirror of your database!
